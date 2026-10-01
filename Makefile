.DEFAULT_GOAL := help

.PHONY: help build test tidy fmt vet lint smoke hooks-install ci init

GOWORK ?= off

help: ## List available make targets
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

build: ## Build runnerconcierge CLI into bin/
	GOWORK=$(GOWORK) go build -o bin/runnerconcierge ./cmd/runnerconcierge

test: ## Run unit tests with race detector
	GOWORK=$(GOWORK) go test -race -count=1 ./...

tidy: ## Run go mod tidy
	GOWORK=$(GOWORK) go mod tidy

fmt: ## Format Go sources with gofmt
	gofmt -w .

vet: ## Run go vet
	GOWORK=$(GOWORK) go vet ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

hooks-install: ## Use .githooks/pre-commit
	chmod +x .githooks/pre-commit
	git config core.hooksPath .githooks

init: build ## Seed user config.yaml
	./bin/runnerconcierge init

smoke: build ## Smoke-test CLI surfaces
	./bin/runnerconcierge help
	./bin/runnerconcierge version
	./bin/runnerconcierge doctor
	./bin/runnerconcierge unknown || test $$? = 2

ci: tidy fmt vet test build smoke ## CI aggregate
