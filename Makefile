go_test ?= -
ifeq (, $(shell which gotest))
	go_test=go test
else
	go_test=gotest
endif

semgrep ?= -
ifeq (,$(shell which semgrep))
	semgrep=echo "-- Running inside Docker --"; docker run --rm -v $$(pwd):/src returntocorp/semgrep:0.86.5
else
	semgrep=semgrep
endif

##@ Build
.PHONY: build
build: ## Build HTTP API
	@go build $(LDFLAGS) -mod=vendor .

.PHONY: gogenerate
gogenerate: ## Generate stringer and mock files
	@go generate -mod=vendor ./...

##@ Test
.PHONY: test
test: ## Run tests
	@$(go_test) $(TESTFLAGS) -race -mod=vendor -count=1 -timeout=45s ./...

##@ Check
.PHONY: check
check: unparam staticcheck vet semgrep checkfmt ## Run all checks

.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@# Ignore below checks because of oapi-codegen generated files
	@staticcheck -checks=inherit,-ST1005,-SA1029,-SA4006 ./...

.PHONY: vet
vet: ## Run go vet
	@go vet -mod=vendor ./...

.PHONY: unparam
unparam: ## Run unparam
	@unparam ./...

.PHONY: semgrep
semgrep: ## Run semgrep
	@$(semgrep) --quiet --metrics=off --config="r/dgryski.semgrep-go" .

.PHONY: checkcodegen
checkcodegen: gogenerate
	@git diff --exit-code --

.PHONY: checkgomod
checkgomod: ## Check go.mod file
	@go mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: checkfmt
checkfmt: ## Check go fmt output
	@out=$$(gofmt -d $$(find . -path ./vendor -prune -o -name '*.go' -print)); \
	if [ -n "$${out}" ]; then \
		echo "Code is not formatted!"; echo "$${out}"; echo; \
		exit 1; \
	fi

##@ Other
help:  ## Display help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
.DEFAULT_GOAL := help