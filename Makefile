go_test ?= -
ifeq (, $(shell which gotest))
	go_test=go test
else
	go_test=gotest
endif

##@ Build
.PHONY: build
build: ## Build the finance-api binary
	@go build -o finance-api ./cmd/finance-api

.PHONY: run
run: build ## Build and run the server
	@./finance-api

##@ Test
.PHONY: test
test: ## Run tests
	@$(go_test) $(TESTFLAGS) -race -count=1 -timeout=60s ./...

##@ Check
.PHONY: check
check: lint checkfmt ## Run all checks

.PHONY: lint
lint: ## Run golangci-lint
	@golangci-lint run ./...

.PHONY: checkfmt
checkfmt: ## Check go fmt output
	@out=$$(gofmt -d $$(find . -name '*.go' -not -path './vendor/*' -print)); \
	if [ -n "$${out}" ]; then \
		echo "Code is not formatted!"; echo "$${out}"; echo; \
		exit 1; \
	fi

.PHONY: fmt
fmt: ## Format code
	@gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -print)

##@ Other
help:  ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.DEFAULT_GOAL := help
