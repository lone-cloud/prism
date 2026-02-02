.PHONY: all build build-linux run dev fmt lint vet clean install-tools deps docker-build docker-run docker-down release

BINARY_NAME=prism
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GOBIN?=$(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

all: fmt lint build

build:
	go build -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o $(BINARY_NAME) .

build-linux:
	GOOS=linux GOARCH=arm64 go build -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT)" -o $(BINARY_NAME)-linux-arm64 .

run: build
	./$(BINARY_NAME)

dev:
	go run .

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
	@echo "Installing signal-cli..."
	@bash scripts/install-signal-cli.sh

deps:
	go mod download
	go mod tidy

check-updates:
	@go list -u -m all | grep -v "indirect" | grep "=>" || echo "All dependencies are up to date"

docker-build:
	docker build -t prism:$(VERSION) .

docker-run:
	docker compose -f docker-compose.dev.yml up -d

docker-down:
	docker compose -f docker-compose.dev.yml down

release:
	@if [ ! -f VERSION ]; then \
		echo "Error: VERSION file not found"; \
		exit 1; \
	fi
	@VERSION=$$(cat VERSION); \
	echo "Releasing v$$VERSION..."; \
	git tag -a "v$$VERSION" -m "Release v$$VERSION"; \
	git push origin "v$$VERSION"; \
	echo "Tag pushed. GitHub Actions will build and push Docker image."
