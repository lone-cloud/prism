.PHONY: all build build-linux run dev fmt lint vet clean install-tools deps check-updates update update-all docker-build docker-run docker-down release

BINARY_NAME=prism
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GOBIN?=$(shell command -v go >/dev/null 2>&1 && go env GOPATH || echo "${HOME}/go")/bin
export PATH := $(GOBIN):$(PATH)

all: fmt lint build

build:
	go build -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o $(BINARY_NAME) .

start: build
	./$(BINARY_NAME)

dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

fmt:
	gofmt -s -w .
	goimports -w .

lint:
	golangci-lint run

fix:
	golangci-lint run --fix

vet:
	go vet ./...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf data/

install-tools:
	@echo "Installing Go tools to $(GOBIN)..."
	go install golang.org/x/tools/cmd/goimports@latest
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v2.8.0

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

docker-build:
	docker build -t prism:$(VERSION) .

docker-up:
	docker compose -f docker-compose.dev.yml up -d

docker-down:
	docker compose -f docker-compose.dev.yml down

docker-up-proton:
	docker compose -f docker-compose.dev.yml up -d protonmail-bridge

docker-up-signal:
	docker compose -f docker-compose.dev.yml up -d signal-cli

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

release-signal:
	@echo "Triggering signal-cli image release via GitHub Actions..."
	gh workflow run release-signal-cli.yml