# logcloak Hands-On Guide

A complete walkthrough of installing logcloak and configuring log masking — from scratch to verified masked output. Every command and output in this guide is real, captured from a live k3s cluster.

---

## What logcloak does

logcloak intercepts pod logs at the source. When a pod is deployed in an opted-in namespace, logcloak automatically injects a sidecar that sits between your app's stdout and the container runtime. Your app writes logs normally — the sidecar masks sensitive data before those logs are ever written to disk.

```
App stdout → FIFO (in-memory) → logcloak sidecar → masked stdout → /var/log/containers/
```

- `kubectl logs <pod>` shows masked output
- `/var/log/containers/` on the node contains only masked output  
- Raw PII never reaches disk at any point
- Zero application code changes required

---

## Prerequisites

- Kubernetes 1.24+ (k3s, GKE, EKS, or AKS)
- Helm 3.x

---

## Step 1 — Add the Helm repository

```bash
helm repo add logcloak https://1mr0-tech.github.io/logcloak
helm repo update
```

**Output:**
```
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "logcloak" chart repository
Update Complete. ⎈Happy Helming!⎈
```

Verify the chart is available:

```bash
helm search repo logcloak
```

**Output:**
```
NAME              CHART VERSION  APP VERSION  DESCRIPTION
logcloak/logcloak 0.2.2          0.2.2        Real-time Kubernetes pod log masker — redacts P...
```

---

## Step 2 — Install logcloak

```bash
helm install logcloak logcloak/logcloak \
  --namespace logcloak \
  --create-namespace
```

**Output:**
```
NAME: logcloak
LAST DEPLOYED: Thu Apr 23 13:44:48 2026
NAMESPACE: logcloak
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

Wait for the pods to be ready:

```bash
kubectl rollout status deployment/logcloak -n logcloak
```

**Output:**
```
Waiting for deployment "logcloak" rollout to finish: 0 of 2 updated replicas are available...
Waiting for deployment "logcloak" rollout to finish: 1 of 2 updated replicas are available...
deployment "logcloak" successfully rolled out
```

Verify both containers (webhook + controller) are running in each pod:

```bash
kubectl get pods -n logcloak
```

**Output:**
```
NAME                        READY   STATUS    RESTARTS   AGE
logcloak-68c64b8fb4-67gmb   2/2     Running   0          50s
logcloak-68c64b8fb4-bj6gp   2/2     Running   0          50s
```

### About TLS — no configuration needed

logcloak automatically generates a self-signed certificate on first startup, stores it in a Kubernetes Secret (`logcloak-tls`), and patches its own `MutatingWebhookConfiguration` with the CA. You never need to configure certificates.

To verify TLS is working:

```bash
kubectl get mutatingwebhookconfiguration logcloak-webhook \
  -o jsonpath='{.webhooks[0].clientConfig.caBundle}' \
  | base64 -d | openssl x509 -noout -subject -dates
```

**Output:**
```
subject=CN = logcloak-ca
notBefore=Apr 23 08:14:04 2026 GMT
notAfter=Apr 20 08:15:04 2036 GMT
```

The certificate is valid for 10 years and auto-renewed on pod restart if it is missing.

---

## Step 3 — Opt a namespace into masking

logcloak uses opt-in by default. Only pods in labelled namespaces get the sidecar injected. This prevents logcloak from interfering with system namespaces or unrelated workloads.

```bash
kubectl create namespace production

kubectl label namespace production logcloak.io/inject=true
```

**Output:**
```
namespace/production created
namespace "production" labeled
```

Verify the label:

```bash
kubectl get namespace production --show-labels
```

**Output:**
```
NAME         STATUS   AGE   LABELS
production   Active   17s   kubernetes.io/metadata.name=production,logcloak.io/inject=true
```

> Any pod created in this namespace from this point forward will automatically have the logcloak sidecar injected. Existing running pods are not affected — they need to be restarted to pick up injection.

---

## Choosing your masking approach

logcloak has two ways to define what gets masked. Use them together — they are designed to complement each other.

| | MaskingPolicy | Pod Annotation |
|---|---|---|
| **Who sets it** | Platform / security team | Developer |
| **Scope** | Applies to all matching pods in a namespace | Applies to one pod only |
| **Use for** | Company-wide PII rules (emails, phone numbers, tokens) that must always be enforced | App-specific patterns (order IDs, session tokens, internal reference formats) |
| **Can override MaskingPolicy?** | — | No. Annotations extend the policy, they cannot remove rules from it |
| **Requires cluster access?** | Yes (`kubectl apply` in namespace) | No (just add annotations to your deployment YAML) |

**The rule:** If a pattern must be masked across every service in a namespace, put it in a `MaskingPolicy`. If it is unique to one service's log format, use a pod annotation.

---

## Step 4 — MaskingPolicy (platform team)

A `MaskingPolicy` is a custom Kubernetes resource that defines masking rules for all pods in a namespace that match a selector. No selector means it applies to every pod in the namespace.

Create the following file and apply it:

```yaml
# masking-policy.yaml
apiVersion: logcloak.io/v1alpha1
kind: MaskingPolicy
metadata:
  name: pii-baseline
  namespace: production
spec:
  patterns:
    - name: email
      builtin: email
    - name: phone-in
      builtin: phone-in
    - name: otp
      builtin: otp-6digit
    - name: credit-card
      builtin: credit-card
    - name: auth-token
      regex: 'Bearer\s+[A-Za-z0-9\-._~+/]+=*'
  redactWith: "[REDACTED]"
```

```bash
kubectl apply -f masking-policy.yaml
```

**Output:**
```
maskingpolicy.logcloak.io/pii-baseline created
```

Verify it was created:

```bash
kubectl get maskingpolicy pii-baseline -n production
```

**Output:**
```
NAME           AGE
pii-baseline   5s
```

### Built-in pattern names

| Name | What it matches |
|---|---|
| `email` | RFC 5321 email addresses |
| `phone-in` | Indian mobile numbers (+91 variants) |
| `phone-us` | US phone numbers |
| `otp-6digit` | Standalone 6-digit numeric codes |
| `credit-card` | 13–19 digit card numbers |
| `jwt` | eyJ… JWT tokens |
| `ipv4-private` | RFC 1918 private IP addresses |
| `uuid` | UUID v4 |
| `iban` | International Bank Account Numbers (all countries) |
| `ssn` | US Social Security Numbers (XXX-XX-XXXX) |
| `phone-e164` | E.164 international phone (+12025550104) |

### Targeting specific services with a selector

If you only want the policy to apply to pods from a specific service, add a `selector`:

```yaml
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: payment-service
  patterns:
    - name: credit-card
      builtin: credit-card
```

Without a selector (as in the example above), the policy applies to every pod in the namespace.

---

## Step 5 — Test MaskingPolicy masking

Deploy a pod that logs typical sensitive data:

```yaml
# payment-service.yaml
apiVersion: v1
kind: Pod
metadata:
  name: payment-service
  namespace: production
spec:
  restartPolicy: Never
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh","-c"]
    args:
    - |
      while true; do
        echo "[INFO] Processing payment for user@shop.com card=4111111111111111 otp=847291"
        echo "[INFO] Auth: Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ1c2VyMTIzIn0.abc123"
        echo "[INFO] Customer phone: +919876543210"
        echo "[INFO] Order shipped to 192.168.1.45"
        sleep 2
      done
```

```bash
kubectl apply -f payment-service.yaml
```

Verify the sidecar was injected (you should see both `app` and `logcloak` containers):

```bash
kubectl get pod payment-service -n production \
  -o jsonpath='{.spec.containers[*].name}'
```

**Output:**
```
app logcloak
```

Check the logs:

```bash
kubectl logs payment-service -n production
```

**Output:**
```
[INFO] Processing payment for [REDACTED] card=[REDACTED]otp=[REDACTED]
[INFO] Auth: [REDACTED]
[INFO] Customer phone: [REDACTED]
[INFO] Order shipped to 192.168.1.45
[INFO] Processing payment for [REDACTED] card=[REDACTED]otp=[REDACTED]
[INFO] Auth: [REDACTED]
[INFO] Customer phone: [REDACTED]
[INFO] Order shipped to 192.168.1.45
```

Note that `192.168.1.45` is NOT masked — it is a private IP address and the `ipv4-private` pattern was not included in this policy. Add it to the MaskingPolicy if you want it redacted.

---

## Step 6 — Pod Annotations (developer)

Pod annotations let developers extend masking with app-specific patterns without needing to modify the cluster-wide policy. Add `logcloak.io/` annotations to your pod or deployment spec.

There are two annotation types:

### Use a built-in pattern

```yaml
annotations:
  logcloak.io/patterns: "uuid,pan-in"
```

List one or more built-in pattern names separated by commas.

### Add a custom regex

```yaml
annotations:
  logcloak.io/regex-<name>: '<your-regex>'
```

Replace `<name>` with a short identifier for the pattern. The regex must be RE2-compatible (no lookaheads, no backreferences).

### Example — order service with custom patterns

```yaml
# order-service.yaml
apiVersion: v1
kind: Pod
metadata:
  name: order-service
  namespace: production
  annotations:
    logcloak.io/patterns: "uuid,pan-in"
    logcloak.io/regex-order-id: 'ORD-[0-9]{8}'
    logcloak.io/regex-session: 'sess_[a-zA-Z0-9]{32}'
spec:
  restartPolicy: Never
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh","-c"]
    args:
    - |
      while true; do
        echo "[INFO] Order ORD-20260423 placed by user@shop.com"
        echo "[INFO] Session sess_a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4 started"
        echo "[INFO] Customer PAN: ABCDE1234F txn=550e8400-e29b-41d4-a716-446655440000"
        sleep 2
      done
```

```bash
kubectl apply -f order-service.yaml
```

```bash
kubectl logs order-service -n production
```

**Output:**
```
[INFO] Order [REDACTED] placed by [REDACTED]
[INFO] Session [REDACTED] started
[INFO] Customer PAN: [REDACTED] txn=[REDACTED]
```

Both the MaskingPolicy rules (email) and the annotation rules (order-id, session, uuid, pan-in) are applied to every log line. The annotation rules extend — not replace — the policy rules.

---

## Step 7 — Verify raw logs on the node are clean

Confirm that raw PII never reached the node's log files:

```bash
sudo grep -r 'user@shop.com\|card=4111\|otp=847291\|9876543210\|ABCDE1234F\|ORD-202604' \
  /var/log/containers/ 2>/dev/null | wc -l
```

**Output:**
```
0
```

Zero matches. The node filesystem contains only masked output.

---

## Step 8 — Reject invalid custom regex at admission

logcloak validates custom regex patterns at pod admission time. If a developer adds an invalid or unsafe regex (lookaheads, backreferences), the pod is rejected with a human-readable error — it never starts with a broken pattern.

Try deploying a pod with an unsafe regex:

```bash
kubectl run bad-pod -n production \
  --image=busybox:1.36 \
  --annotations='logcloak.io/regex-bad=(?=lookahead)' \
  -- sh -c 'echo test'
```

**Output:**
```
Error from server: admission webhook "mutate.logcloak.io" denied the request:
invalid regex in annotation "logcloak.io/regex-bad": unsafe regex construct "(?=" detected
— only RE2-compatible patterns allowed
```

The pod is blocked before it starts.

---

## Step 9 — Service mesh and non-standard sidecars

logcloak is safe alongside Istio, Linkerd, and any other service mesh. Meshes intercept network traffic at the kernel level — logcloak intercepts stdout. They don't cross paths.

### Automatically skipped containers

These are never wrapped regardless of what you configure:

`istio-proxy` · `linkerd-proxy` · `envoy` · `envoy-sidecar` · `kuma-sidecar` · `consul-sidecar` · `vault-agent` · `config-reloader`

### Excluding non-standard sidecars

Got a postgres sidecar? A redis container? A monitoring agent? Tell logcloak to leave them alone:

```yaml
metadata:
  annotations:
    logcloak.io/exclude-containers: "postgres,redis,datadog-agent"
```

Full example — three containers, only `app` gets wrapped:

```yaml
metadata:
  annotations:
    logcloak.io/exclude-containers: "postgres"
spec:
  containers:
  - name: app        # ← logcloak wraps this one
    image: myapp:latest
  - name: postgres   # ← excluded via annotation
    image: postgres:16
  - name: istio-proxy  # ← excluded automatically
    image: istio/proxyv2:1.20
```

---

## Step 10 — kubectl logs behaviour reference

| Command | What you see |
|---|---|
| `kubectl logs <pod>` | Masked output (logcloak sidecar is the default container) |
| `kubectl logs <pod> -c logcloak` | Same masked output |
| `kubectl logs <pod> -c app` | Empty — app stdout is redirected to the FIFO, not captured by containerd |
| `/var/log/containers/<pod>_<ns>_logcloak-*.log` | Masked output only |
| `/var/log/containers/<pod>_<ns>_app-*.log` | Empty file |

---

## Step 10 — Uninstall

```bash
helm uninstall logcloak -n logcloak
kubectl delete namespace logcloak
kubectl delete namespace production
```

MaskingPolicy resources in other namespaces are removed automatically when their namespace is deleted. The CRD is removed by the Helm uninstall.

---

## Pattern helper tool

Use the interactive helper at **https://1mr0-tech.github.io/logcloak/tool/** to:

- Select built-in patterns and preview masking against a real log line
- Generate regex from sample data (e.g. paste `ORD-20260423` and get `ORD-\d+`)
- Get copy-ready MaskingPolicy YAML or pod annotation snippet
- See the `kubectl label` command for your namespace

---

## RBAC for platform teams

Two ClusterRoles are provided for managing MaskingPolicies without cluster-admin:

```bash
# Grant a user full MaskingPolicy management
kubectl create clusterrolebinding ops-admin \
  --clusterrole=logcloak-admin \
  --user=ops@company.com

# Grant a user read-only MaskingPolicy view
kubectl create clusterrolebinding dev-viewer \
  --clusterrole=logcloak-viewer \
  --user=dev@company.com
```

> `logcloak-admin` and `logcloak-viewer` ClusterRoles are available from v0.2.3.

---

## Quick reference

```bash
# Add repo
helm repo add logcloak https://1mr0-tech.github.io/logcloak && helm repo update

# Install
helm install logcloak logcloak/logcloak --namespace logcloak --create-namespace

# Opt a namespace in
kubectl label namespace <name> logcloak.io/inject=true

# Create a MaskingPolicy
kubectl apply -f masking-policy.yaml

# Check masking is working
kubectl logs <pod> -n <namespace>

# List all masking policies in a namespace
kubectl get maskingpolicies -n <namespace>
```
