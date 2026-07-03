---
name: Bug report
about: Report a bug or unexpected behavior in logcloak
labels: bug
---

**logcloak version**
<!-- Output of: helm list -n <namespace> or the image tag you are running -->

**Kubernetes version**
<!-- Output of: kubectl version --short -->

**What happened**
<!-- A clear description of the bug -->

**What you expected to happen**

**Steps to reproduce**
<!-- Minimal pod spec, MaskingPolicy YAML, or log input that triggers the bug -->

**Relevant logs**
<!-- kubectl logs -n <namespace> deploy/logcloak (webhook logs) -->
<!-- kubectl logs <pod> -c logcloak-sidecar (sidecar logs) -->

**Additional context**
