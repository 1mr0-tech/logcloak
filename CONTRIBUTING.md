# Contributing to logcloak

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.22+ | Build and test |
| Docker Desktop | any recent | Build multi-platform images |
| kubectl | 1.28+ | Manual cluster interaction |
| k3d | 5.x | Local Kubernetes cluster |
| golangci-lint | 1.57+ | Linting (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`) |
| setup-envtest | latest | Integration test binaries (`go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest`) |

## Getting started

```sh
git clone https://github.com/1mr0-tech/logcloak
cd logcloak
go mod download
```

## Running tests

### Unit tests

```sh
make test
# or with verbose output
go test ./... -race -v
```

### Integration tests (envtest)

The integration suite in `test/integration/` runs the admission webhook against a real fake Kubernetes API server. It requires `KUBEBUILDER_ASSETS` to point to envtest binaries.

```sh
# One-time setup: download envtest binaries for Kubernetes 1.30
setup-envtest use 1.30 --bin-dir /tmp/envtest-bins

# Export the path (add to your shell profile)
export KUBEBUILDER_ASSETS=$(setup-envtest use 1.30 --bin-dir /tmp/envtest-bins -p path)

# Run integration tests
go test ./test/integration/... -v
```

If `KUBEBUILDER_ASSETS` is not set the suite exits 0 with a printed reminder — it does not fail CI by default.

### Lint

```sh
make lint
```

## Building

```sh
# Compile all four binaries for the local OS/arch
make build
# Output: bin/logcloak-{webhook,controller,sidecar,cli}

# Build multi-platform Docker images (does not push)
make docker-build
```

## Local cluster setup with k3d

```sh
# Create a single-node cluster
k3d cluster create logcloak-dev --agents 1

# Load locally built images into the cluster
k3d image import ghcr.io/1mr0-tech/logcloak-webhook:dev -c logcloak-dev

# Install via Helm
helm upgrade --install logcloak charts/logcloak \
  --namespace logcloak --create-namespace \
  --set image.tag=dev
```

See `GUIDE.md` for full end-to-end installation and usage walkthrough.

## Project structure

```
cmd/          # Binary entrypoints (webhook, controller, sidecar, cli)
pkg/          # Shared packages
  masker/     # Regex replacement engine
  metrics/    # Prometheus counters and histograms
  patterns/   # Built-in named PII patterns
  regex/      # RE2 safety validator
  rules/      # MaskingPolicy CRD types, Merge, Serialize/Deserialize
  sentinel/   # [LOGCLOAK-DROP] formatting
  webhook/    # Admission handler, JSON Patch builder, TLS
charts/       # Helm chart
test/
  integration/  # envtest-based admission webhook tests
  k8s/          # Manifests for manual cluster testing
```

## Pull request process

1. Fork the repository and create a branch from `main`.
2. Run `make test && make lint` — both must pass before opening a PR.
3. Keep PRs focused: one logical change per PR.
4. Write a clear description of *why* the change is needed, not just what it does.
5. Add or update tests for any behavioural change.
6. A maintainer will review within 7 days. Expect at least one round of feedback.

## Code style

- No comments unless the *why* is non-obvious (a hidden constraint, a subtle invariant, a workaround).
- No docstrings on obvious functions — well-named identifiers are enough.
- Error strings are lowercase with no trailing punctuation.
- Prefer returning errors over panicking except in `main()` and `TestMain`.
- All regex patterns must pass `pkg/regex.Validate` before use.

## Commit messages

Follow conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `chore:`. First line ≤ 72 characters.
