# Kubernetes integration test manifests

These manifests are for manual validation against a real cluster. They are not run in CI — use the envtest suite in `test/integration/` for automated tests.

## Prerequisites

```sh
# Create a local cluster
k3d cluster create logcloak-dev --agents 1

# Install logcloak
helm upgrade --install logcloak ../../charts/logcloak \
  --namespace logcloak --create-namespace

# Label the production namespace for injection
kubectl create namespace production
kubectl label namespace production logcloak.io/inject=true

# Apply the baseline MaskingPolicy
kubectl apply -f masking-policy.yaml
```

## Manifests

| File | Purpose |
|------|---------|
| `masking-policy.yaml` | Baseline MaskingPolicy with email, OTP, credit-card, JWT, phone patterns |
| `stress-pod.yaml` | Single pod writing ~100 lines/sec with mixed PII and plain lines |
| `stress-batch.yaml` | Deployment of 10 stress pods (~1,000 lines/sec cluster-wide) |
| `app-with-postgres.yaml` | Multi-container pod; postgres excluded via annotation |
| `distroless-app.yaml` | Distroless base image; no shell in the container |
| `jvm-app.yaml` | JVM container with multi-line stack traces |

## Running the stress test

```sh
kubectl apply -f stress-batch.yaml

# Watch pod resource usage
watch -n2 kubectl top pods -n production -l app=stress-batch

# Check metrics on the sidecar (replace pod name)
SIDECAR=$(kubectl get pods -n production -l app=stress-batch -o name | head -1)
kubectl exec -n production $SIDECAR -c logcloak -- \
  wget -qO- http://localhost:9090/metrics | grep logcloak_
```

### Pass criteria

- `logcloak_dropped_lines_total` remains 0 throughout the run
- `logcloak_processing_duration_seconds` p99 < 1 ms
- Sidecar container memory stays below 64 MiB (`kubectl top pod`)
- No `[LOGCLOAK-DROP]` lines in `kubectl logs`

```sh
kubectl logs -n production -l app=stress-batch -c logcloak | grep LOGCLOAK-DROP | wc -l
# Expected: 0
```

## Running the distroless test

```sh
kubectl apply -f distroless-app.yaml
kubectl logs distroless-app -n production -c logcloak -f

# Verify PII is masked
kubectl logs distroless-app -n production -c logcloak | grep REDACTED
# Verify no raw email addresses leak
kubectl logs distroless-app -n production -c logcloak | grep '@' | grep -v REDACTED
# Expected: no output
```

## Running the JVM test

```sh
kubectl apply -f jvm-app.yaml
kubectl logs jvm-app -n production -c logcloak -f

# PII lines should be masked
kubectl logs jvm-app -n production -c logcloak | grep REDACTED
# Stack trace lines should pass through unmodified
kubectl logs jvm-app -n production -c logcloak | grep "at com.example"
```

## Cleanup

```sh
kubectl delete -f .
k3d cluster delete logcloak-dev
```
