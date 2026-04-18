.PHONY: all build start dev fix install-tools check-updates release release-dev

BINARY_NAME=prism
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
GOBIN?=$(shell command -v go >/dev/null 2>&1 && go env GOPATH || echo "${HOME}/go")/bin
export PATH := $(GOBIN):$(PATH)

all: fix build

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) .

start: build
	./$(BINARY_NAME)

dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

fix:
	go mod download
	go mod tidy
	gofmt -s -w .
	goimports -w .
	golangci-lint run --fix
	npx @biomejs/biome@latest check --write --unsafe .

install-tools:
	@echo "Installing Go tools to $(GOBIN)..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0
	@echo "Installing signal-cli native binary..."
	@ARCH=$$(uname -m); \
	case $$ARCH in \
		x86_64) SIGNAL_ARCH=amd64 ;; \
		aarch64) SIGNAL_ARCH=arm64 ;; \
		*) echo "Unsupported architecture: $$ARCH"; exit 1 ;; \
	esac; \
	gunzip -c signal-cli/signal-cli-$${SIGNAL_ARCH}.gz > /tmp/signal-cli && \
	sudo mv /tmp/signal-cli /usr/local/bin/signal-cli && \
	sudo chmod +x /usr/local/bin/signal-cli && \
	signal-cli --version && \
	echo "signal-cli installed successfully to /usr/local/bin/signal-cli"

check-updates:
	@echo "=== Go module updates ==="
	@go list -u -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{if .Update}} -> {{.Update.Version}}{{end}}{{end}}' all | grep " -> " || echo "All Go dependencies are up to date"
	@echo ""
	@echo "=== Dockerfile base image updates ==="
	@for image in $$(grep -E '^FROM ' Dockerfile | awk '{print $$2}' | grep -v 'AS'); do \
		echo "Checking $$image..."; \
		name=$$(echo $$image | cut -d: -f1); \
		tag=$$(echo $$image | cut -d: -f2); \
		digest=$$(curl -sf "https://hub.docker.com/v2/repositories/library/$$name/tags/$$tag" | jq -r '.digest // empty'); \
		if [ -z "$$digest" ]; then \
			echo "  $$image: could not fetch digest (offline or image not found)"; \
			continue; \
		fi; \
		cache_key=$$(echo $$image | tr ':/' '--'); \
		stored=$$(grep "^$$cache_key=" .image-digests 2>/dev/null | cut -d= -f2); \
		if [ -z "$$stored" ]; then \
			echo "$$cache_key=$$digest" >> .image-digests; \
			echo "  $$image: digest saved for future comparisons"; \
		elif [ "$$stored" = "$$digest" ]; then \
			echo "  $$image: up to date"; \
		else \
			sed -i "s|^$$cache_key=.*|$$cache_key=$$digest|" .image-digests; \
			echo "  $$image: UPDATE AVAILABLE (image content has changed, consider rebuilding)"; \
		fi; \
	done

release:
	@if [ ! -f VERSION ]; then \
		echo "Error: VERSION file not found"; \
		exit 1; \
	fi
	@VERSION=$$(cat VERSION); \
	echo "Releasing v$$VERSION..."; \
	git tag -a "v$$VERSION" -m "Release v$$VERSION"; \
	git push origin "v$$VERSION"; \
	gh workflow run release.yml

release-dev:
	@echo "Triggering dev Docker image build..."
	gh workflow run release-dev.yml
