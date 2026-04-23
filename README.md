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
