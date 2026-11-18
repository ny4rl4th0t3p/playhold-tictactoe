GOVULNCHECK_VERSION := v1.9.0
BUF_VERSION := v1.75.0
BUF := go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)

# The chapters with messages of their own, each its own buf project, as the reader's game folder would be. Every
# chapter generates the same tictactoe/v1/tictactoe.proto into a Go package of its own; that's only safe because no
# program links two chapters, since Go's protobuf registry is global to a program.
PROTO_CHAPTERS := $(patsubst %/buf.yaml,%,$(wildcard chapter*/buf.yaml))
# The chapters with a browser page, in web/.
WEB_CHAPTERS := $(patsubst %/web/package.json,%,$(wildcard chapter*/web/package.json))

.DEFAULT_GOAL := help

.PHONY: help check build vet gamevet test buflint web vuln gen tidy

help: ## Show the available targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-z]+:.*## / {printf "  %-8s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: build vet gamevet test buflint web vuln ## Run every check, on every chapter

build: ## Build every chapter
	go build ./...

vet: ## Run go vet
	go vet ./...

gamevet: ## Run playhold's determinism checks, on the playhold version in go.mod
	go vet -vettool=$$(go tool -n playholdvet) ./...

test: ## Run every chapter's tests with the race detector, without cached results
	go test -count=1 -race ./...

buflint: ## Lint every chapter's .proto files
	@for ch in $(PROTO_CHAPTERS); do echo "buf lint $$ch"; (cd $$ch && $(BUF) lint) || exit 1; done

# A page's packages are installed only when its node_modules is missing or older than its lockfile, so an install isn't
# deleted and redone on every run; touch marks it current. CI starts clean, so it always installs.
gen: ## Generate every chapter's code from its .proto files: Go, and the page's TypeScript
	@for ch in $(PROTO_CHAPTERS); do \
		if [ -f $$ch/web/package.json ] && [ ! $$ch/web/node_modules -nt $$ch/web/package-lock.json ]; then \
			npm --prefix $$ch/web ci && touch $$ch/web/node_modules || exit 1; \
		fi; \
		echo "buf generate $$ch"; (cd $$ch && $(BUF) generate) || exit 1; \
	done

web: ## Install where needed, then type-check and build every chapter's page
	@for ch in $(WEB_CHAPTERS); do \
		echo "page $$ch"; \
		if [ ! $$ch/web/node_modules -nt $$ch/web/package-lock.json ]; then \
			npm --prefix $$ch/web ci && touch $$ch/web/node_modules || exit 1; \
		fi; \
		npm --prefix $$ch/web run check || exit 1; \
	done

vuln: ## Check dependencies and the standard library for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

tidy: ## Tidy go.mod and go.sum
	go mod tidy