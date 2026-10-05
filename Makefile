.DEFAULT_GOAL := help

.PHONY: help build test tidy fmt vet lint smoke hooks-install ci init e2e-live gitlab-list gitlab-setup

GOWORK ?= off

help: ## Show make targets and common workflows
	@echo "runnerconcierge"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-22s %s\n", $$1, $$2}'
	@echo ""
	@echo "  GitLab runners:"
	@echo "    1. make init"
	@echo "    2. make gitlab-setup"
	@echo "    3. make gitlab-list"
	@echo ""
	@echo "  Doctor, status, cleanup, repair-service, and JSON inventory live in the CLI."
	@echo "  Run: bin/runnerconcierge help"
	@echo "       bin/runnerconcierge runners gitlab list --help"
	@echo "       bin/runnerconcierge runners gitlab setup --help"

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

hooks-install: ## Install .githooks/pre-commit
	chmod +x .githooks/pre-commit
	git config core.hooksPath .githooks

init: build ## Seed operator config (~/.config/runnerconcierge)
	./bin/runnerconcierge init

gitlab-list: build ## Interactive GitLab runner picker (runners gitlab list)
	./bin/runnerconcierge runners gitlab list

gitlab-setup: build ## GitLab runner setup wizard (runners gitlab setup)
	./bin/runnerconcierge runners gitlab setup

smoke: build ## Smoke-test CLI surfaces
	./bin/runnerconcierge | grep -q 'Documentation for agents'
	./bin/runnerconcierge help
	./bin/runnerconcierge version
	./bin/runnerconcierge doctor
	./bin/runnerconcierge runners gitlab --help | grep -q 'GitLab runners'
	./bin/runnerconcierge runners gitlab list --help | grep -q 'inspect'
	./bin/runnerconcierge runners gitlab setup --help | grep -q 'resume'
	./bin/runnerconcierge setup || test $$? = 2
	./bin/runnerconcierge cleanup --help | grep -q 'Prune stale'
	./bin/runnerconcierge unknown || test $$? = 2

ci: tidy fmt vet test build smoke ## CI aggregate (tidy, fmt, vet, test, build, smoke)

e2e-live: ## Live status/repair E2E (darwin/windows; not PR CI)
	GOWORK=$(GOWORK) go test -tags=e2e_live -count=1 -timeout 30m ./internal/e2elive/...
