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
	gofmt -s -w .
	goimports -w .
	golangci-lint run --fix
	npx @biomejs/biome@latest check --write --unsafe .
	go mod download
	go mod tidy

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
	@go list -u -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{if .Update}} [{{.Update.Version}}]{{end}}{{end}}' all | grep "\[" || echo "All dependencies are up to date"

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
