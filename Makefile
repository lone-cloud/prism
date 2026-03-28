.PHONY: all build build-linux run dev lint fix vet clean install-tools deps check-updates update update-all docker-build docker-run docker-down release

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

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf data/

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
		current_digest=$$(docker inspect --format='{{index .RepoDigests 0}}' $$image 2>/dev/null | cut -d@ -f2); \
		docker pull -q $$image > /dev/null 2>&1; \
		latest_digest=$$(docker inspect --format='{{index .RepoDigests 0}}' $$image 2>/dev/null | cut -d@ -f2); \
		if [ -z "$$current_digest" ]; then \
			echo "  $$image: pulled (no prior local image to compare)"; \
		elif [ "$$current_digest" = "$$latest_digest" ]; then \
			echo "  $$image: up to date"; \
		else \
			echo "  $$image: UPDATE AVAILABLE"; \
			echo "    local:  $$current_digest"; \
			echo "    latest: $$latest_digest"; \
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
