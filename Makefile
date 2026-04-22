REGISTRY ?= ghcr.io/1mr0-tech
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARIES  = webhook controller sidecar cli

.PHONY: build test lint docker-build docker-push helm-package clean

build:
	@for bin in $(BINARIES); do \
		echo "Building logcloak-$$bin..."; \
		CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=$(VERSION)" \
			-o bin/logcloak-$$bin ./cmd/$$bin; \
	done

test:
	go test ./... -race -coverprofile=coverage.out

lint:
	golangci-lint run ./...

docker-build:
	@for bin in $(BINARIES); do \
		docker buildx build \
			--platform linux/amd64,linux/arm64 \
			--build-arg BINARY=$$bin \
			--build-arg VERSION=$(VERSION) \
			-t $(REGISTRY)/logcloak-$$bin:$(VERSION) \
			-t $(REGISTRY)/logcloak-$$bin:latest \
			-f build/Dockerfile \
			.; \
	done

docker-push:
	@for bin in $(BINARIES); do \
		docker buildx build \
			--platform linux/amd64,linux/arm64 \
			--build-arg BINARY=$$bin \
			--build-arg VERSION=$(VERSION) \
			-t $(REGISTRY)/logcloak-$$bin:$(VERSION) \
			-t $(REGISTRY)/logcloak-$$bin:latest \
			-f build/Dockerfile \
			--push \
			.; \
	done

helm-package:
	helm package charts/logcloak --destination dist/

helm-publish: helm-package
	@echo "Publishing Helm chart to gh-pages..."
	@helm repo index dist/ --url https://1mr0-tech.github.io/logcloak
	@git checkout gh-pages 2>/dev/null || git checkout --orphan gh-pages
	@cp dist/*.tgz dist/index.yaml . 2>/dev/null || true
	@git add *.tgz index.yaml
	@git commit -m "chore: release chart $(VERSION)"
	@git push origin gh-pages
	@git checkout main

clean:
	rm -rf bin/ dist/ coverage.out
