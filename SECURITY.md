# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.2.x   | ✅        |
| < 0.2   | ❌        |

## Reporting a Vulnerability

Email **imran.roshan.official@gmail.com** with the subject line:

```
[logcloak] Security Vulnerability Report
```

Include:
- A description of the vulnerability and its potential impact
- Steps to reproduce (Kubernetes version, logcloak version, pod spec, MaskingPolicy YAML if relevant)
- Whether you have a suggested fix

**Do not open a public GitHub issue for security vulnerabilities.**

### Response timeline

| Step | SLA |
|------|-----|
| Acknowledgement | 48 hours |
| Severity triage | 7 days |
| Fix or mitigation | 30 days for critical, 90 days for moderate/low |
| Public disclosure | Coordinated with reporter; default 90 days after initial report |

We follow coordinated disclosure. If you need to disclose sooner for any reason, please discuss this in your initial report.

## Scope

### In scope

- **Masker bypass** — a regex pattern that should mask PII but does not
- **Webhook injection tampering** — an attacker able to manipulate which rules are injected into the sidecar
- **RE2 validator bypass** — a pattern accepted by the validator that causes ReDoS or catastrophic backtracking
- **Information leakage via drop sentinel** — `[LOGCLOAK-DROP]` lines that expose structure of the original line
- **FIFO side-channel** — a container in the same pod reading the raw FIFO before the masker processes it
- **TLS issues** — weak cipher suites, certificate validation bypasses in the admission webhook TLS

### Out of scope

- Vulnerabilities in the underlying Kubernetes cluster or cloud provider
- Social engineering of project maintainers
- Issues in dependencies that have already been publicly disclosed and have no logcloak-specific exploit path
- Vulnerabilities that require `cluster-admin` access (the attacker already controls the cluster)

## Known security properties

These are intentional design choices, not vulnerabilities:

- **Fail-closed masking** — if a log line takes longer than 5ms to process, it is dropped and replaced with `[LOGCLOAK-DROP]`. Raw PII never reaches stdout on timeout.
- **RE2-only regex** — the webhook rejects patterns containing lookaheads, lookbehinds, or backreferences. This prevents ReDoS in the masker goroutine.
- **Memory-backed FIFO** — the shared volume between app and sidecar is `emptyDir: medium: Memory`, so log data never touches node disk.
- **Annotation validation** — custom regex annotations are validated at admission time. Pods with invalid patterns are denied.
- **No rule removal via annotations** — pod annotations can only extend MaskingPolicy rules, never remove them.
