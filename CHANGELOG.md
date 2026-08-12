# Changelog

All notable changes to logcloak are documented here.

Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/).

---

## [0.6.1] - 2026-08-12

### Fixed

- **CI on `main` was broken across four independent points.** `cmd/cli/audit.go` (added in 0.6.0) had unchecked
  `fmt.Fprintf`/`fmt.Fprintln` return values failing the Lint job's errcheck pass. `cmd/controller` and
  `pkg/metrics` had no test files, which triggered a `go: no such tool 'covdata'` failure in CI's Go toolchain
  under `go test -race -coverprofile=...`; both now have real tests. The DCO Sign-off job was pinned to a
  `dco-check` action version that never existed, then (once fixed) failed to authenticate against the GitHub
  API, then (once authenticated) correctly rejected dependabot's bot-authored commits due to an email mismatch
  between commit author and sign-off — all three are now fixed.

### Changed

- **Upgraded to Go 1.25** (`go.mod`, CI workflows, `build/Dockerfile`) to pick up `golang.org/x/net` v0.55.0,
  which requires Go 1.25 per its own `go.mod`.

---

## [0.6.0] - 2026-07-27

### Added

- **`logcloak audit <policy.yaml> [file]` CLI command.** Checks a specific `MaskingPolicy` against a real log
  sample and reports any built-in PII category the full pattern library detects that the policy's configured
  `builtin:` patterns don't cover. Each gap is shown exactly as `kubectl logs` would render it under that
  policy, plus copy-paste YAML to close it. Answers "would my policy actually catch this?" before deploying,
  rather than after an incident. Scope: checks builtin-pattern coverage only — custom `regex:` rule quality
  and JSON `field:` rules aren't part of the gap comparison.

---

## [0.5.1] - 2026-07-25

### Fixed

- **`MaskingPolicy` CRD rejected the documented `field:` pattern type.** The README's own JSON field masking example (`patterns[].field: password`) failed with `strict decoding error: unknown field "spec.patterns[0].field"` because `charts/logcloak/crds/maskingpolicies.logcloak.io.yaml` never declared `field` in its OpenAPI schema, even though the Go-side `PatternSpec.Field` and masking logic have supported it since JSON field masking shipped in 0.4.0. The pod-annotation path (`logcloak.io/fields`) was unaffected. CRD-based cluster-wide JSON field masking now works as documented.

---

## [0.5.0] - 2026-07-25

### Added

- **PodDisruptionBudget.** The webhook deployment now ships with a `PodDisruptionBudget` (`minAvailable: 1` by default, configurable via `podDisruptionBudget.*` Helm values) so voluntary node drains can't take down all webhook replicas at once.
- **`seccompProfile: RuntimeDefault`** applied to webhook, controller, and sidecar containers as part of Pod Security Standards hardening.
- **TLS certificate rotation.** The webhook now rotates its self-signed cert automatically before expiry and exposes `logcloak_tls_cert_expiry_seconds` as a Prometheus gauge. Fixes the prior race where multiple replicas could generate conflicting certs on first boot (see 0.4.1 fix, now hardened further).
- **Structured JSON logging (`slog`)** across webhook and controller, replacing ad hoc `fmt`/`log` output.
- **`ServiceMonitor` template**, disabled by default (`serviceMonitor.enabled: false`), for clusters running the Prometheus Operator.
- **Integration test coverage** for `cmd/sidecar` and `cmd/webhook` binaries.

### Changed

- CI lint pipeline migrated to `golangci-lint-action@v7` for Node 24 runner compatibility; fixed resulting `errcheck` violations in sidecar and webhook test code.
- Added `.dockerignore` to fix Docker build context transfer failures caused by extended attributes on macOS.

---

## [0.4.1] - 2026-06-14

### Fixed

- **TLS cert race in multi-replica deployments.** With `replicaCount > 1`, both pods could enter `EnsureTLS` simultaneously before the secret existed, each generating a different self-signed cert. The second pod would overwrite the secret with its cert, leaving both pods serving different in-memory certs while only one CA was in the `caBundle`. The kube-apiserver would then reject roughly half of webhook calls with "bad certificate", causing pods to start without sidecar injection. `storeTLSSecret` now reads and returns the existing bundle on conflict instead of overwriting it, so all replicas share one cert that matches the `caBundle`.

---

## [0.4.0] - 2026-06-14

### Added

- **JSON field masking.** Services that log in JSON can now mask values by field name, regardless of value format. Use the `logcloak.io/fields` pod annotation for per-pod rules or the `field:` spec in a `MaskingPolicy` for cluster-wide enforcement. Field masking recurses into nested objects and arrays and runs before regex rules, so both can be active simultaneously. Applies to log lines that begin with `{`.
- **`logcloak scan` CLI command.** Scans any log source (stdin or file) against all 10 built-in patterns and prints a per-line hit report and summary. Useful for auditing existing services before enabling masking. No Kubernetes cluster required.
- **Competitor comparison table in README.** Documents the architectural difference between logcloak (pre-capture interception) and pipeline-based tools (Vector, Fluent Bit, Cribl, CloudWatch) that mask after logs hit disk.

### Fixed

- **`phone-us` false positive on UUID digit sequences.** The pattern matched 10-digit sequences at the end of UUIDs (e.g. `446655440000` in `550e8400-e29b-41d4-a716-446655440000`) because it lacked word-boundary anchors. Added `\b` at both ends. UUIDs are now also ordered before `phone-us` in the test MaskingPolicy so they are redacted before `phone-us` inspects the line.

### Changed

- Built-in pattern tables in README, documentation.md, and GUIDE.md corrected. Removed non-existent patterns (`phone-in`, `aadhaar`, `pan-in`) and replaced with the actual library (`phone-e164`, `iban`, `ssn`).
- Stale `sidecar.auditLog` and `sidecar.processingTimeoutMs` Helm values and their documentation removed. Both were no-ops since the v0.3.3 synchronous masking change.
- Self-signed TLS certificate validity updated in docs from 10 years to 1 year, reflecting the security hardening in v0.3.4.

---

## [0.3.4] - 2026-05-05

### Security

- **Critical masking bypass fixed.** `ReplaceAllString` was used for regex substitution, allowing a crafted `redactWith` value like `$1` to expand capture groups and leak the matched PII into the output. Changed to `ReplaceAllLiteralString` across all masking paths.
- **RE2 validation applied to MaskingPolicy CRD patterns.** Custom regex fields in CRD specs are now validated with `regex.Validate()` at sync time, the same way annotation patterns are validated at admission.
- **Webhook request body capped at 4 MiB.** `MaxBytesReader` prevents unbounded memory growth from oversized `AdmissionReview` payloads.
- **TLS server hardened.** Added `ReadHeaderTimeout`, `ReadTimeout`, and `WriteTimeout` to prevent slow-client connection exhaustion.
- **Self-signed cert validity reduced from 10 years to 1 year.**
- **RBAC scoped correctly.** Moved `secrets get/create/update` from `ClusterRole` to a namespace-scoped `Role`. The `logcloak-tls` secret is only ever accessed in the release namespace.

---

## [0.3.3] - 2026-05-01

### Fixed

- **Masking runs synchronously.** Removed the goroutine-per-line timeout model introduced in an earlier iteration. Masking now runs inline without spawning goroutines, which eliminated a class of race conditions and simplified the processing path.

---

## [0.3.2] - 2026-04-30

### Fixed

- Removed `caBundle` from the Helm chart template. The webhook self-patches `caBundle` at runtime on startup. Having it in the template caused `helm upgrade` to reset the bundle and break TLS verification until the next pod restart.

---

## [0.3.1] - 2026-04-30

### Fixed

- Webhook metrics server was not starting. The metrics `ListenAndServe` call was missing from the startup path.
- Sidecar processing timeout is now configurable via `sidecar.processingTimeoutMs` Helm value (subsequently removed in v0.4.0 after synchronous masking made it a no-op).

---

## [0.3.0] - 2026-04-30

### Added

- **Per-rule Prometheus metrics.** `logcloak_lines_masked_total` is now labeled by rule name, so operators can see which patterns are firing and at what rate.
- **`envtest` integration test suite** in `test/integration/`. Covers opt-in namespace injection, skip annotation, invalid regex rejection, multi-container pods, MaskingPolicy selector filtering, annotation extension, and policy deletion.
- `SECURITY.md` with CVE disclosure process, response SLA, scope, and security properties.
- `CONTRIBUTING.md` with prerequisites, unit and integration test instructions, k3d setup, and PR process.
- `docs/failure-modes.md` documenting sidecar OOMKill, FIFO back-pressure, masking timeout, webhook unavailability, controller crash, policy deletion, and distroless entrypoint edge cases. Includes PII-exposure column.

---

## [0.2.5] - 2026-04-23

### Added

- Hands-on guide (`GUIDE.md`) with step-by-step install outputs and masking verification from a live k3s cluster.
- Service mesh compatibility documentation and `logcloak.io/exclude-containers` annotation to skip non-standard sidecar containers.
- ASCII logo in README.

### Changed

- Replaced India-specific patterns (`phone-in`, `aadhaar`, `pan-in`) with a global built-in set: `phone-e164`, `iban`, `ssn`.

### Fixed

- Service mesh proxy containers (`istio-proxy`, `linkerd-proxy`, `envoy`, `kuma-sidecar`, `consul-sidecar`, `vault-agent`, `config-reloader`) are now automatically excluded from entrypoint wrapping regardless of pod spec.
- golangci-lint `errcheck` and unused variable warnings resolved.

---

## [0.2.3] - 2026-04-23

### Added

- `logcloak-admin` and `logcloak-viewer` ClusterRoles for managing MaskingPolicies without cluster-admin.
- Helm test pod that calls `/healthz` to verify the webhook is reachable after install.
- DNS egress added to the webhook NetworkPolicy so it can resolve the Kubernetes API server hostname.

---

## [0.2.2] - 2026-04-23

### Fixed

- FIFO created with mode `0666` so app containers running as non-root UIDs can write to it regardless of the sidecar UID.

---

## [0.2.1] - 2026-04-23

### Fixed

- Removed TLS secret from Helm chart templates. The webhook creates and manages the secret at runtime. Having it in the chart caused conflicts on reinstall.

---

## [0.2.0] - 2026-04-23

### Added

- Mutating admission webhook with TLS self-bootstrap (generates self-signed cert on startup, patches `caBundle` automatically).
- MaskingPolicy CRD controller with label selector support and rule cache with configurable TTL.
- Pod injection logic: FIFO creation, entrypoint wrapping, sidecar injection, skip annotation, default container annotation.
- CLI with `validate` and `preview` subcommands.
- Prometheus metrics for lines processed, masked, dropped, processing duration, webhook admissions, and errors.
- Drop sentinel written to stdout when a line cannot be processed.
- NetworkPolicy restricting webhook ingress to the kube-apiserver.
- GitHub Actions release pipeline: builds and pushes four images (`webhook`, `controller`, `sidecar`, `cli`) for `linux/amd64` and `linux/arm64`, packages and publishes the Helm chart to GitHub Pages, creates a GitHub release with pre-built binaries. Trivy vulnerability scan results uploaded to GitHub Security.

### Fixed

- Webhook name must have three dot-separated segments to satisfy Kubernetes admission registration validation.
- CI pipeline: upgraded Go to 1.24, fixed Helm chart publishing to `gh-pages` branch.

---

## [0.1.0] - 2026-04-22

### Added

- Initial project scaffold.
- `pkg/masker`: RE2-safe regex masking with fail-closed drop sentinel.
- `pkg/patterns`: 10 built-in patterns (email, phone-e164, phone-us, otp-6digit, credit-card, jwt, ipv4-private, uuid, iban, ssn).
- `pkg/rules`: MaskingPolicy types, annotation parser, and rule merger with CRD-first deduplication.
- `pkg/regex`: RE2 validator that rejects lookaheads, lookbehinds, and backreferences at admission time.
- `pkg/sentinel`: Drop sentinel writer.
- Helm chart scaffold, README, and documentation.
- Trivy vulnerability scanning in CI with SARIF upload to GitHub Security.

---

[0.4.1]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.4.1
[0.4.0]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.4.0
[0.3.4]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.3.4
[0.3.3]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.3.3
[0.3.2]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.3.2
[0.3.1]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.3.1
[0.3.0]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.3.0
[0.2.5]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.2.5
[0.2.3]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.2.3
[0.2.2]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.2.2
[0.2.1]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.2.1
[0.2.0]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.2.0
[0.1.0]: https://github.com/1mr0-tech/logcloak/releases/tag/v0.1.0
