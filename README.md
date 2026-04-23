```
░██                                 ░██████  ░██                       ░██
░██                                ░██   ░██ ░██                       ░██
░██          ░███████   ░████████ ░██        ░██  ░███████   ░██████   ░██    ░██
░██         ░██    ░██ ░██    ░██ ░██        ░██ ░██    ░██       ░██  ░██   ░██
░██         ░██    ░██ ░██    ░██ ░██        ░██ ░██    ░██  ░███████  ░███████
░██         ░██    ░██ ░██   ░███  ░██   ░██ ░██ ░██    ░██ ░██   ░██  ░██   ░██
░██████████  ░███████   ░█████░██   ░██████  ░██  ░███████   ░█████░██ ░██    ░██
                              ░██
                        ░███████
```

Real-time Kubernetes pod log masker. Redacts PII, OTPs, tokens, and sensitive data before logs reach `kubectl logs`, `/var/log/containers/`, or any downstream log store. Zero changes to your application code.

Works out of the box on **k3s, GKE, EKS, and AKS**.

---

## The problem

Your app logs are probably leaking data you didn't mean to expose.

A developer adds a debug line: `log.info("Processing order for {}", user.email)`. That line ships to Datadog, gets indexed in Elasticsearch, ends up in an S3 bucket, forwarded to your SIEM, and read by six people who were only supposed to see latency metrics. Nobody noticed. This is the default state of most Kubernetes deployments.

**The actual attack surface is not your database — it's your logs.**

Most teams have strong access controls on databases and strong encryption at rest, then inadvertently ship plaintext PII through every log pipeline they operate.

### "But anyone with kubectl exec can see everything anyway"

Yes — and that is a separate control. `kubectl exec` requires explicit RBAC permissions that are typically restricted in production, audited, and often completely disabled. `kubectl logs` access is handed out far more broadly — to developers, support teams, CI/CD pipelines, and log aggregation agents.

logcloak defends the log surface specifically:

| Where PII leaks | logcloak stops it |
|---|---|
| `kubectl logs` in a dev's terminal | ✅ |
| `/var/log/containers/` on the node | ✅ |
| Log shippers (Fluent Bit, Fluentd, Promtail) | ✅ — they read the node files |
| Third-party platforms (Datadog, Splunk, Elastic) | ✅ — they receive the shipped files |
| S3/GCS log archives | ✅ |
| CI/CD pipelines that capture stdout | ✅ |
| `kubectl exec` into a running container | ❌ — not in scope, use RBAC |
| Application memory / heap dumps | ❌ — not in scope |

### Who this is for

- Teams that must comply with **GDPR, HIPAA, or PCI-DSS** and cannot audit every log statement in every service
- Platforms where **developers have log access but must not see customer PII**
- Any company shipping logs to a **third-party vendor** and wanting to limit what leaves the cluster
- Teams that have tried "just don't log PII" as a policy and found it doesn't survive contact with a deadline

logcloak is a defense-in-depth control. It works alongside RBAC, encryption at rest, and network policy — it does not replace them.

---

## How it works

logcloak injects a masker sidecar into opted-in pods via a Mutating Admission Webhook. The app container's stdout/stderr is redirected through an in-memory named pipe (never touches disk) to the sidecar, which applies regex masking and outputs clean logs to its own stdout. The container runtime only ever captures masked output.

```
app stdout → [in-memory FIFO] → logcloak sidecar → masked stdout → /var/log/containers/
```

`kubectl logs <pod>` returns masked output by default.

## Quick start

```bash
# 1. Install logcloak
helm repo add logcloak https://1mr0-tech.github.io/logcloak
helm repo update
helm install logcloak logcloak/logcloak --namespace logcloak --create-namespace

# 2. Opt in a namespace
kubectl label namespace my-app logcloak.io/inject=true

# 3. Annotate your pod
```

```yaml
metadata:
  annotations:
    logcloak.io/patterns: "email,otp-6digit,jwt"
    logcloak.io/regex-order-id: "ORD-[0-9]{8}"
```

```bash
# 4. Check logs — PII is masked
kubectl logs my-pod
# [2026-04-22T10:15:30Z] user [REDACTED:email] placed order ORD-[REDACTED:order-id]
```

## Built-in patterns

| Name | Matches |
|---|---|
| `email` | Email addresses |
| `phone-in` | Indian mobile numbers (+91 variants) |
| `phone-us` | US phone numbers |
| `otp-6digit` | 6-digit OTP codes |
| `credit-card` | 13–19 digit card numbers |
| `jwt` | JWT tokens |
| `ipv4-private` | RFC 1918 private IPs |
| `uuid` | UUID v4 |
| `aadhaar` | 12-digit Aadhaar numbers |
| `pan-in` | Indian PAN card format |

## Cluster-wide policies (MaskingPolicy CRD)

```yaml
apiVersion: logcloak.io/v1alpha1
kind: MaskingPolicy
metadata:
  name: pii-baseline
  namespace: production
spec:
  selector:
    matchLabels:
      app.kubernetes.io/part-of: checkout-service
  patterns:
    - name: email
      builtin: email
    - name: otp
      builtin: otp-6digit
    - name: auth-token
      regex: "Bearer\\s+[A-Za-z0-9\\-._~+/]+=*"
```

## Drop sentinel

When a line cannot be processed, logcloak never passes it through. A visible sentinel appears in `kubectl logs`:

```
[LOGCLOAK-DROP] 2026-04-22T10:15:30Z | reason=regex_timeout | pod=my-app | line suppressed
```

## Security

- Raw logs never touch the node filesystem (in-memory FIFO only)
- RE2 regex engine — immune to ReDoS attacks
- Fail-closed: processing failures suppress the line, never expose it
- Sidecar runs as non-root UID 65534, read-only filesystem

## Installation options

```bash
# With cert-manager (GKE/EKS/AKS)
helm install logcloak logcloak/logcloak \
  --namespace logcloak --create-namespace \
  --set tls.certManager=true

# Self-signed TLS (k3s / air-gapped)
helm install logcloak logcloak/logcloak \
  --namespace logcloak --create-namespace \
  --set tls.mode=self-signed

# Strict mode — reject pods if webhook is unavailable
helm install logcloak logcloak/logcloak \
  --namespace logcloak --create-namespace \
  --set webhook.failurePolicy=Fail
```

## Uninstall

```bash
helm uninstall logcloak -n logcloak
kubectl delete crd maskingpolicies.logcloak.io
kubectl label namespace <your-namespace> logcloak.io/inject-
```

## License

MIT
