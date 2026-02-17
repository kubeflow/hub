PORT ?= 4000
MOCK_K8S_CLIENT ?= false
MOCK_MR_CLIENT ?= false
MOCK_MR_CATALOG_CLIENT ?= false
DEV_MODE ?= false
DEV_MODE_MODEL_REGISTRY_PORT ?= 8080
DEV_MODE_CATALOG_PORT ?= 8081
DEPLOYMENT_MODE ?= kubeflow
AUTH_METHOD ?= internal
AUTH_TOKEN_HEADER ?= Authorization
AUTH_TOKEN_PREFIX ?= Bearer
INSECURE_SKIP_VERIFY ?= false
#frontend static assets root directory
STATIC_ASSETS_DIR ?= ./static
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.29.3
LOG_LEVEL ?= info
ALLOWED_ORIGINS ?= ""
DEBUG ?= false
GCFLAGS_DEBUG := -gcflags="all=-N -l"

.PHONY: all
all: build

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: fmt
fmt: ## Applies the correct code style to source files using go fmt.
	go fmt ./...

.PHONY: clean ## Deletes previously built binaries.
clean:
	rm -Rf ./bin

.PHONY: lint
lint: golangci-lint ## Run golangci-lint to automatically check source code for programmatic and stylistic errors.
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint to automatically check source code for programmatic and stylistic errors and, additionally perform fixes where possible.
	$(GOLANGCI_LINT) run --fix

.PHONY: vet
vet:  . ## Runs static analysis tools on source files and reports suspicious constructs that could be bugs or syntactical errors.
	go vet ./...

.PHONY: test
test: fmt vet envtest ## Runs the full test suite.
	ENVTEST_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	go test ./...

.PHONY: build
build: fmt vet openapi test ## Builds the project to produce a binary executable. OpenAPI spec is generated before build.
ifeq ($(DEBUG), true) ## If DEBUG is true, build with debugging symbols
	go build $(GCFLAGS_DEBUG) -o bin/bff ./cmd
else
	go build -o bin/bff ./cmd
endif

.PHONY: run
run: fmt vet openapi envtest ## Runs the project. OpenAPI spec is generated before run.
	trap 'exit 0' INT; \
	ENVTEST_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	go run ./cmd --port=$(PORT) --auth-method=${AUTH_METHOD} --auth-token-header=$(AUTH_TOKEN_HEADER) --auth-token-prefix="$(AUTH_TOKEN_PREFIX)" --static-assets-dir=$(STATIC_ASSETS_DIR) --mock-k8s-client=$(MOCK_K8S_CLIENT) --mock-mr-client=$(MOCK_MR_CLIENT) --mock-mr-catalog-client=$(MOCK_MR_CATALOG_CLIENT)  --dev-mode=$(DEV_MODE) --dev-mode-model-registry-port=$(DEV_MODE_MODEL_REGISTRY_PORT) --dev-mode-catalog-port=$(DEV_MODE_CATALOG_PORT)  --deployment-mode=$(DEPLOYMENT_MODE) --log-level=$(LOG_LEVEL) --allowed-origins=$(ALLOWED_ORIGINS) --insecure-skip-verify=$(INSECURE_SKIP_VERIFY)

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
ENVTEST ?= $(LOCALBIN)/setup-envtest-$(ENVTEST_VERSION)
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint-$(GOLANGCI_LINT_VERSION)

## Tool Versions
GOLANGCI_LINT_VERSION ?= v2.12.2
ENVTEST_VERSION ?= release-0.19

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,${GOLANGCI_LINT_VERSION})

SWAG_VERSION ?= v1.16.4
SWAG ?= $(LOCALBIN)/swag-$(SWAG_VERSION)

.PHONY: swag
swag: $(SWAG) ## Download swag locally if necessary.
$(SWAG): $(LOCALBIN)
	$(call go-install-tool,$(SWAG),github.com/swaggo/swag/cmd/swag,$(SWAG_VERSION))

ALL_GO_DIRS := $(shell find . -type f -name '*.go' -exec dirname {} \; | sed 's|^\./||' | sort -u)
ALL_GO_DIRS_NO_CMD := $(shell echo "$(ALL_GO_DIRS)" | tr ' ' '\n' | grep -v '^cmd$$' | paste -sd, -)
SWAG_DIRS := cmd,$(ALL_GO_DIRS_NO_CMD)

.PHONY: openapi
openapi: swag ## Generate OpenAPI spec from annotations
	$(SWAG) fmt -g cmd/main.go -d $(SWAG_DIRS)
	$(SWAG) init --parseDependency -q -g main.go -d $(SWAG_DIRS) -o openapi --outputTypes go,json,yaml --requiredByDefault


# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f $(1) ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv "$$(echo "$(1)" | sed "s/-$(3)$$//")" $(1) ;\
}
endef
