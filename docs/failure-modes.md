# logcloak Failure Modes

This document describes what happens when individual components of logcloak fail. Honest failure documentation is a prerequisite for production adoption.

---

## 1. Sidecar OOMKill

**What happens:** The kernel kills the `logcloak` sidecar container when it exceeds its memory limit. The app container continues running.

**Effect on logs:** The FIFO (`/masker-pipe/app.pipe`) is still open on the write end by the app. Once the sidecar process dies there is no reader. The app's next `write()` to the FIFO will block as soon as the kernel's pipe buffer (default 64 KB, 8 MiB in this case because the emptyDir is memory-backed) fills. **The app hangs.**

**Recovery:** Kubernetes restarts the sidecar container (restartPolicy=Always by default). Once the sidecar comes back the FIFO is drained and the app unblocks. Restart latency is typically 1–10 seconds depending on backoff.

**Mitigation:**
- Set sidecar memory request/limit to at least 64 MiB (Helm default). Monitor `container_memory_working_set_bytes` on the `logcloak` container.
- Alert on `kube_pod_container_status_restarts_total{container="logcloak"}` crossing a threshold.
- If the app cannot tolerate blocking, consider setting `restartPolicy: OnFailure` on the pod and treating the OOM as a pod-level failure — this is the safer choice for PCI-DSS environments.

---

## 2. FIFO buffer fills (app back-pressure)

**What happens:** The sidecar processes lines slower than the app writes them. The memory-backed emptyDir buffers up to 8 MiB of unread pipe data. When the buffer is full, app `write()` calls block in the kernel.

**Effect on logs:** The app pauses until the sidecar catches up. No data is lost; no PII is exposed.

**When this occurs:** Sustained log throughput above ~5,000 lines/second with complex regex patterns, or if the sidecar is CPU-throttled below what it needs.

**Mitigation:**
- Monitor `logcloak_processing_duration_seconds` — p99 above 1ms is a warning sign.
- Increase sidecar CPU limit if throttling is observed (`kubectl top pod`).
- Reduce the number of regex patterns or simplify complex patterns.
- The 8 MiB FIFO size covers ~80,000 lines of 100-byte log messages — most workloads never come close.

---

## 3. Per-line masking timeout (5 ms)

**What happens:** The masker goroutine does not complete within 5 ms. The sidecar drops the line and writes a `[LOGCLOAK-DROP reason=timeout pod=<name> ns=<namespace> ts=<rfc3339>]` sentinel to stdout instead.

**Effect on logs:** One line of raw log is silently discarded. The sentinel is visible in `kubectl logs` and in log aggregators. The metric `logcloak_dropped_lines_total{reason="timeout"}` increments.

**When this occurs:** Only on pathological regex patterns (catastrophic backtracking). The RE2 validator prevents this for user-supplied patterns. Could occur with very long lines (> 512 KB) against patterns with many alternatives.

**Mitigation:**
- The RE2 validator blocks backtracking-prone patterns at admission time.
- Alert on `logcloak_dropped_lines_total` increasing.
- If drop rate is non-zero, check `logcloak_processing_duration_seconds` for high-latency lines.

---

## 4. Webhook unavailable

**What happens:** The logcloak webhook pod is down (crash, rollout, node eviction) and cannot respond to admission requests.

**Effect on new pods:** The Helm chart sets `failurePolicy: Ignore` by default. New pods are admitted and started **without masking**. Existing running pods are unaffected.

**Mitigation:**
- The Helm chart deploys 2 webhook replicas by default for HA.
- For strict compliance environments, change `failurePolicy` to `Fail`. New pods will be blocked from starting while the webhook is unavailable. This is the correct posture for PCI-DSS level 1.
- Monitor `logcloak_webhook_admissions_total{result="error"}` and alert on any non-zero value.

---

## 5. Controller crash

**What happens:** The `logcloak-controller` pod crashes or is unavailable.

**Effect:** The controller's only current role is reconciling `MaskingPolicy` objects and logging. It does not serve the admission webhook. A controller crash has **no immediate effect** on running pods or log masking.

**Effect on new pods:** The webhook reads MaskingPolicies directly from the Kubernetes API via the `k8s.io/client-go` client, not through the controller. New pods continue to receive correct masking rules as long as the API server is healthy.

**Recovery:** Kubernetes restarts the controller automatically.

---

## 6. MaskingPolicy deleted while pods are running

**What happens:** An operator deletes a `MaskingPolicy` that running pods depend on.

**Effect:** Already-running pods are unaffected. The rules were serialised into the `LOGCLOAK_RULES` environment variable at admission time. The sidecar reads this variable once at startup. There is no live reload.

**Effect on new pods:** New pods admitted after the deletion will not receive rules from the deleted policy. This is intentional — rules are immutable per pod lifecycle.

---

## 7. LOGCLOAK_RULES env var too large

**What happens:** An unusually large number of regex patterns causes the serialised `LOGCLOAK_RULES` JSON to exceed the environment variable size limit (131,071 bytes on Linux).

**Effect:** Pod admission fails with an error from the Kubernetes API server, not from logcloak. The pod is not started.

**When this occurs:** Extremely unlikely in practice. A typical rule (name + pattern + replacement) serialises to ~100 bytes. You would need ~1,300 rules to approach the limit.

**Mitigation:** Consolidate patterns into a single MaskingPolicy. Use named builtins rather than duplicated inline regexes.

---

## 8. Distroless or scratch image with no shell

**What happens:** The sidecar injection rewrites the app container's entrypoint to redirect stdout through the FIFO. For images with no `command` in the pod spec, the webhook falls back to `exec "$0" "$@"`, which relies on the image's `ENTRYPOINT`. This works correctly for distroless images (which always define an `ENTRYPOINT`) but may produce unexpected behaviour if the image has neither a pod-spec command nor a Dockerfile `ENTRYPOINT`.

**Effect:** If no entrypoint is resolvable, the app container fails to start with `Error: exec: "": executable file not found`.

**Mitigation:** Always specify `command` in the pod spec for scratch-based images, or add an `ENTRYPOINT` to the Dockerfile.

---

## Summary table

| Failure | App logs | PII exposed? | Auto-recovers? |
|---------|----------|--------------|----------------|
| Sidecar OOMKill | App blocks until sidecar restarts | No | Yes (kubelet restart) |
| FIFO buffer full | App blocks (backpressure) | No | Yes (sidecar catches up) |
| Masking timeout | Line dropped, sentinel emitted | No | N/A (per-line) |
| Webhook down (Ignore) | New pods start unmasked | **Yes** | Yes (webhook restarts) |
| Webhook down (Fail) | New pods blocked | No | Yes (webhook restarts) |
| Controller crash | No effect on running pods | No | Yes (kubelet restart) |
| MaskingPolicy deleted | No effect on running pods | No | N/A |
