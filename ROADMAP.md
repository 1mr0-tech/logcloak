# logcloak Roadmap

The honest path from proof-of-concept to a project people actually use and talk about.

---

## Where we are

**v0.3.0** — Phase 1 complete. Envtest integration suite, per-rule Prometheus metrics, SECURITY.md, CONTRIBUTING.md, failure mode documentation, stress and distroless/JVM test manifests. Ready for public outreach.

---

## Phase 1 — Make it trustworthy
**Target: 4–6 weeks**

Nobody adopts a log-intercepting sidecar they don't trust. Trust needs to be earned explicitly before going public.

### Stability work

- [x] **Integration tests with `envtest`** — `test/integration/` covers opt-in namespace, skip annotation, invalid regex rejection, multi-container pods, MaskingPolicy selector filtering, annotation extension, and policy deletion.
- [x] **High-volume stress testing** — `test/k8s/stress-batch.yaml` runs 10 concurrent pods at ~100 lines/sec each. `test/k8s/README.md` documents pass criteria (zero drops, p99 < 1ms, sidecar memory < 64Mi).
- [x] **Distroless / JVM / multi-init image testing** — `test/k8s/distroless-app.yaml` and `test/k8s/jvm-app.yaml` with documented verification steps. Unit test `TestBuildPatch_NoCommandNoArgs` covers the entrypoint fallback.
- [x] **Failure mode documentation** — `docs/failure-modes.md` covers sidecar OOMKill, FIFO back-pressure, masking timeout, webhook unavailability, controller crash, policy deletion, and distroless entrypoint edge case. Includes a summary table with PII-exposure column.
- [x] **`SECURITY.md`** — CVE disclosure email, response SLA, scope definition, coordinated disclosure policy, and documented security properties.
- [x] **`CONTRIBUTING.md`** — prerequisites, unit and integration test instructions, k3d local cluster setup, PR process, code style.
- [x] **`ROADMAP.md`** (this file) — committed and kept updated.

---

## Phase 2 — Get the first 3 real users
**Target: 2–3 months of active outreach**

Three companies running logcloak in staging or production is worth more than a thousand GitHub stars. Everything else depends on this.

### Where to find them

- [ ] **Kubernetes Slack** (`kubernetes.slack.com`) — post in `#sig-security` and `#kubernetes-users`. Two sentences: the problem, what you built. Don't pitch. Ask if anyone has hit the problem. Let the conversation do the rest.
- [ ] **One really good blog post** — not a product announcement, a technical post about the problem. Suggested title: *"Why your Kubernetes logs are a GDPR liability even if your database isn't."* Publish on dev.to or Medium, then submit as a guest post to the [CNCF blog](https://www.cncf.io/blog/). One accepted CNCF guest post reaches 50,000 engineers.
- [ ] **HackerNews "Show HN"** — *"Show HN: I built a Kubernetes sidecar that masks PII before logs reach kubectl."* Be honest about limitations. HN rewards that.
- [ ] **r/kubernetes and r/devops** — same message, different audience. DevOps people think about compliance more than developers do.
- [ ] **GDPR-fined companies** — public record. Their engineering teams are actively searching for solutions.

---

## Phase 3 — Build community signal
**Target: months 2–4**

### GitHub hygiene

- [ ] **Respond to every issue within 24 hours** for the first 6 months. Speed of response is the single biggest predictor of repeat contributors.
- [ ] **Enable GitHub Discussions** — issues are for bugs, discussions are where community forms.
- [ ] **Set up a Discord or Slack invite** — people want to ask questions before filing an issue.
- [ ] **Label backlog issues** `good first issue` and `help wanted` with detailed descriptions. Vague issues get zero contributors; specific ones get pull requests.

### Find a co-maintainer

- [ ] **Recruit one co-maintainer** from the Kubernetes security community. One-maintainer projects make companies nervous. You only need one person who reviews PRs and has commit access. People are more willing than you think if they believe in the project.

---

## Phase 4 — Conference talks
**Target: months 3–6**

This is where "I built a project" becomes "I'm known for this project."

### Start small

- [ ] **Local Kubernetes meetup** — 15-minute demo. Every major city has one. Refines the pitch, gets immediate feedback.
- [ ] **CNCF online meetup** — CNCF runs free online events and frequently needs speakers. Low barrier, high-quality audience.

### Aim for

- [ ] **CloudNativeSecurityCon** — security-focused, less competitive CFP than KubeCon. Submit a talk.
- [ ] **KubeCon EU or NA** — CFP acceptance rate ~15% but submitting forces you to articulate the value. What makes a winning talk:
  - Real production case study from a named company
  - Numbers: "X% of pods were logging PII before logcloak, zero after"
  - A live demo that works
  - A problem the audience recognises from their own work

---

## Phase 5 — CNCF Sandbox application
**Target: 12–24 months**

Minimum bar based on current CNCF due diligence:

- [ ] 3+ production users willing to be named publicly
- [ ] 2+ active maintainers
- [ ] Security disclosure process (`SECURITY.md`)
- [ ] Governance document (`GOVERNANCE.md`)
- [ ] Apache 2.0 or MIT licence ✅ (already MIT)
- [ ] Active community — issues, PRs, not a dead repo

Apply via PR to [github.com/cncf/toc](https://github.com/cncf/toc). Getting in doesn't make you famous. The journey to get there — the blog posts, the talks, the users — is what builds the reputation.

---

## Technical backlog (features deferred until Phase 1 is complete)

These are real gaps. Do not start new features before the Phase 1 stability work is done.

- [x] `envtest` integration test suite
- [ ] Helm test pod that validates masking end-to-end (not just `/healthz`)
- [ ] Cosign image signing in release pipeline
- [ ] `ServiceMonitor` CRD for Prometheus scraping
- [x] Structured log support (JSON field-level masking via `logcloak.io/fields` annotation and `field:` MaskingPolicy spec — shipped in v0.4.0)
- [ ] OpenTelemetry trace ID passthrough (don't mask trace IDs that look like UUIDs)
- [ ] Annual cert rotation CronJob for self-signed TLS mode

---

## What will kill the project — watch for these

**Neglect.** If you disappear for 3 months, the community disappears too.

**Scope creep.** logcloak does one thing: masks PII in logs. The moment you add "log analytics" or "threat detection" you become a worse version of existing products.

**Overpromising.** If a company adopts logcloak because you claimed it works everywhere and it breaks their distroless app, they will blog about it. The Kubernetes community is small and tight-knit.

**Skipping the boring work.** Documentation, issue triage, changelog entries. 90% of maintenance is invisible and unglamorous. The projects that survive are run by people who do it anyway.

---

## Realistic timeline

| Milestone | Realistic timeframe |
|---|---|
| Phase 1 complete — trustworthy codebase | 4–6 weeks |
| First 3 production users | 2–3 months |
| First conference talk (local or online) | 1–2 months |
| KubeCon talk accepted | 6–18 months |
| CNCF Sandbox application | 12–24 months |
| Known in the Kubernetes security community | 2–3 years of consistent work |

The people famous for Kubernetes projects spent years doing unglamorous work before anyone knew their names. The technical work is the easy part. The community work is the job.
