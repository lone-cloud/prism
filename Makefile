.PHONY: all build build-linux run dev lint fix vet clean install-tools deps check-updates update update-all docker-build docker-run docker-down release

BINARY_NAME=prism
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GOBIN?=$(shell command -v go >/dev/null 2>&1 && go env GOPATH || echo "${HOME}/go")/bin
export PATH := $(GOBIN):$(PATH)

all: fix build

build:
	go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o $(BINARY_NAME) .

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

vet:
	go vet ./...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf data/

install-tools:
	@echo "Installing Go tools to $(GOBIN)..."
	go install golang.org/x/tools/cmd/goimports@latest
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.8.0
	@echo "Installing signal-cli native binary..."
	@ARCH=$$(uname -m); \
	case $$ARCH in \
		x86_64) SIGNAL_ARCH=amd64 ;; \
		aarch64) SIGNAL_ARCH=arm64 ;; \
		*) echo "Unsupported architecture: $$ARCH"; exit 1 ;; \
	esac; \
	TMP_DIR=$$(mktemp -d); \
	cd $$TMP_DIR && \
	curl -L -o signal-cli.gz \
		https://media.projektzentrisch.de/temp/signal-cli/signal-cli_ubuntu2004_$${SIGNAL_ARCH}.gz && \
	gunzip signal-cli.gz && \
	sudo mv signal-cli /usr/local/bin/signal-cli && \
	sudo chmod +x /usr/local/bin/signal-cli && \
	cd - && \
	rm -rf $$TMP_DIR && \
	signal-cli --version && \
	echo "signal-cli installed successfully to /usr/local/bin/signal-cli"

deps:
	go mod download
	go mod tidy

check-updates:
	@go list -u -m -f '{{if not .Indirect}}{{.Path}} {{.Version}}{{if .Update}} [{{.Update.Version}}]{{end}}{{end}}' all | grep "\[" || echo "All dependencies are up to date"

update:
	go get -u ./...
	go mod tidy

update-all:
	go get -u all
	go mod tidy

docker-up:
	docker compose -f docker-compose.dev.yml up -d

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
