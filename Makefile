GOVULNCHECK_VERSION := v1.9.0

.DEFAULT_GOAL := help

.PHONY: help check build vet gamevet test vuln tidy

help: ## Show the available targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-z]+:.*## / {printf "  %-8s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: build vet gamevet test vuln ## Run every check, on every chapter

build: ## Build every chapter
	go build ./...

vet: ## Run go vet
	go vet ./...

gamevet: ## Run playhold's determinism checks, on the playhold version in go.mod
	go vet -vettool=$$(go tool -n playholdvet) ./...

test: ## Run every chapter's tests with the race detector, without cached results
	go test -count=1 -race ./...

vuln: ## Check dependencies and the standard library for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy