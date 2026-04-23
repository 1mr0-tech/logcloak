# logcloak

Real-time Kubernetes pod log masker. Redacts PII, OTPs, tokens, and any sensitive data from pod logs before they are written to disk or shown in `kubectl logs`.

Works out of the box on k3s, GKE, EKS, and AKS with no changes to your application code.

---

## How it works

logcloak injects a masker sidecar into opted-in pods via a Mutating Admission Webhook. The app container's stdout and stderr are redirected through an in-memory named pipe to the sidecar, which applies regex masking rules and outputs clean logs to its own stdout. The container runtime only ever captures the sidecar's masked output.

- `kubectl logs <pod>` returns masked output by default
- `/var/log/containers/` on the node contains only masked output
- Raw PII never touches disk at any point

---

## Installation

### Prerequisites

- Kubernetes 1.24+ (k3s, GKE, EKS, or AKS)
- Helm 3.x

### Install

```bash
helm repo add logcloak https://1mr0-tech.github.io/logcloak
helm repo update
helm install logcloak logcloak/logcloak \
  --namespace logcloak \
  --create-namespace
```

### With cert-manager (recommended for managed clusters)

```bash
helm install logcloak logcloak/logcloak \
  --namespace logcloak \
  --create-namespace \
  --set tls.certManager=true
```

### Verify installation

```bash
kubectl get pods -n logcloak
kubectl get crd maskingpolicies.logcloak.io
```

---

## Opting in a namespace

logcloak only injects into namespaces with the opt-in label. This prevents interference with system namespaces.

```bash
kubectl label namespace <your-namespace> logcloak.io/inject=true
```

All new pods created in that namespace will automatically have the masker sidecar injected.

> **Note:** Existing running pods are not affected. Restart them to pick up injection.

---

## Defining masking rules

There are two ways to define rules — they stack together.

### Option 1: MaskingPolicy CRD (cluster/team-wide rules)

Managed by your platform or security team. Apply to all pods matching a label selector within a namespace.

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
  redactWith: "[REDACTED]"
```

```bash
kubectl apply -f maskingpolicy.yaml
```

Empty `selector` matches all pods in the namespace.

### Option 2: Pod annotations (per-pod developer rules)

Developers declare sensitive fields directly on their pod manifest. These rules are added on top of any matching MaskingPolicy — they cannot remove CRD-level rules.

```yaml
metadata:
  annotations:
    logcloak.io/patterns: "email,otp-6digit,jwt"
    logcloak.io/regex-order-id: "ORD-[0-9]{8}"
    logcloak.io/regex-session: "sess_[a-zA-Z0-9]{32}"
```

`logcloak.io/patterns` enables built-in patterns by name (comma-separated).
`logcloak.io/regex-<name>` defines a custom RE2 regex pattern.

---

## Built-in patterns

Enable these by name in `MaskingPolicy.spec.patterns[].builtin` or in the `logcloak.io/patterns` annotation.

| Name | What it matches |
|---|---|
| `email` | RFC 5321 email addresses |
| `phone-e164` | E.164 international phone numbers (+12025550104) |
| `phone-us` | US phone number formats |
| `otp-6digit` | Standalone 6-digit numeric OTP codes |
| `credit-card` | 13–19 digit card numbers |
| `jwt` | JWT tokens (`eyJ...`) |
| `ipv4-private` | RFC 1918 private IP addresses |
| `uuid` | UUID v4 |
| `iban` | International Bank Account Numbers (all countries) |
| `ssn` | US Social Security Numbers (XXX-XX-XXXX) |

---

## Custom regex rules

Custom patterns must be valid RE2 expressions. Lookaheads, lookbehinds, and backreferences are rejected at pod admission time to prevent ReDoS attacks.

```yaml
# In MaskingPolicy
patterns:
  - name: internal-token
    regex: "tok_[a-zA-Z0-9]{40}"

# In pod annotation
logcloak.io/regex-internal-token: "tok_[a-zA-Z0-9]{40}"
```

If a pod is submitted with an invalid regex, the admission webhook rejects it with a clear error message:

```
Error from server: admission webhook denied the request:
logcloak: invalid regex "tok_(?=abc)" in annotation logcloak.io/regex-foo:
unsafe regex construct "(?=" detected — only RE2-compatible patterns allowed
```

---

## Viewing logs

```bash
# Masked output (default — shows sidecar output)
kubectl logs <pod>

# Also masked (explicit sidecar container)
kubectl logs <pod> -c logcloak

# Empty — app stdout is redirected to the masker pipe
kubectl logs <pod> -c <app-container>
```

---

## Skipping injection

To exclude a specific pod from injection even in an opted-in namespace:

```yaml
metadata:
  annotations:
    logcloak.io/skip: "true"
```

---

## Service mesh and sidecar compatibility

logcloak is safe to run alongside Istio, Linkerd, Consul Connect, AWS App Mesh, and Kuma. Service meshes intercept network traffic at the socket/iptables layer — entirely separate from stdout — so there is no functional conflict.

### Automatically skipped containers

logcloak never wraps these containers regardless of the pod spec:

| Container name | Service mesh / tool |
|---|---|
| `istio-proxy` | Istio |
| `linkerd-proxy` | Linkerd |
| `envoy` | AWS App Mesh, standalone Envoy |
| `envoy-sidecar` | Consul Connect |
| `kuma-sidecar` | Kuma |
| `consul-sidecar` | Consul |
| `vault-agent` | Vault Agent Injector |
| `config-reloader` | Common infrastructure pattern |

### Excluding non-standard sidecars

For sidecars not in the list above — a postgres container, a redis sidecar, a monitoring agent — use the `logcloak.io/exclude-containers` annotation:

```yaml
metadata:
  annotations:
    logcloak.io/exclude-containers: "postgres,redis,datadog-agent"
```

Comma-separated. Spaces around names are fine. Only the listed containers are excluded — all other app containers are still wrapped.

```yaml
# Example: app with a postgres sidecar and Istio
metadata:
  annotations:
    logcloak.io/exclude-containers: "postgres"
spec:
  containers:
  - name: app          # wrapped by logcloak ✓
    image: myapp:latest
  - name: postgres     # excluded via annotation ✓
    image: postgres:16
  - name: istio-proxy  # excluded automatically ✓
    image: istio/proxyv2:1.20
```

---

## Drop sentinel

When a log line cannot be processed (regex timeout, sidecar error), logcloak never passes the raw line through. Instead it writes a sentinel to stdout that is visible in `kubectl logs`:

```
[LOGCLOAK-DROP] 2026-04-22T10:15:30Z | reason=regex_timeout | pod=my-app-7d9f | line suppressed
```

You can alert on `LOGCLOAK-DROP` in your log aggregation system to detect processing failures.

---

## Observability

logcloak exposes Prometheus metrics on port `9090` from the sidecar container.

| Metric | Description |
|---|---|
| `logcloak_lines_processed_total` | Total log lines processed |
| `logcloak_lines_masked_total` | Lines where at least one pattern matched |
| `logcloak_dropped_lines_total` | Lines dropped (by reason) |
| `logcloak_processing_duration_seconds` | Per-line processing latency histogram |
| `logcloak_webhook_admissions_total` | Webhook admission outcomes |
| `logcloak_webhook_errors_total` | Webhook failures |

### Enable audit log

To emit a structured redaction event to sidecar stderr each time a pattern fires:

```bash
helm upgrade logcloak logcloak/logcloak --set sidecar.auditLog=true
```

---

## Configuration reference

| Value | Default | Description |
|---|---|---|
| `tls.certManager` | `false` | Use cert-manager for TLS (recommended for managed clusters) |
| `tls.mode` | `self-signed` | TLS mode: `self-signed`, `certManager`, `bring-your-own` |
| `webhook.failurePolicy` | `Ignore` | `Ignore`: pods start uninjected if webhook is down. `Fail`: blocks pod creation |
| `webhook.timeoutSeconds` | `5` | Webhook response timeout |
| `sidecar.auditLog` | `false` | Emit structured redaction events to sidecar stderr |
| `sidecar.processingTimeoutMs` | `5` | Max milliseconds per log line before drop |
| `sidecar.maxLineSizeBytes` | `1048576` | Lines larger than this are dropped with `reason=line_too_long` |
| `sidecar.resources.requests.cpu` | `5m` | CPU request per injected pod |
| `sidecar.resources.requests.memory` | `20Mi` | Memory request per injected pod |
| `controller.ruleCacheTTL` | `30s` | How often the controller re-syncs rules |
| `metrics.enabled` | `true` | Enable Prometheus metrics endpoint |
| `metrics.port` | `9090` | Metrics port |

---

## Supported clusters

| Cluster | TLS recommendation | Notes |
|---|---|---|
| k3s | `tls.mode=self-signed` | Works with default containerd logging |
| GKE | `tls.certManager=true` | cert-manager available via GKE add-on |
| EKS | `tls.certManager=true` | Install cert-manager before logcloak |
| AKS | `tls.certManager=true` | cert-manager available via AKS add-on |

---

## Uninstall

```bash
helm uninstall logcloak -n logcloak
kubectl delete crd maskingpolicies.logcloak.io
kubectl label namespace <your-namespace> logcloak.io/inject-
```
