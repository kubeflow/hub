# useful paths
MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
PROJECT_PATH := $(patsubst %/,%,$(dir $(MKFILE_PATH)))
PROJECT_BIN := $(PROJECT_PATH)/bin
GO ?= "$(shell which go)"
UI_PATH := $(PROJECT_PATH)/clients/ui
CSI_PATH := $(PROJECT_PATH)/cmd/csi
CONTROLLER_PATH := $(PROJECT_PATH)/cmd/controller

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.33
ENVTEST ?= $(PROJECT_BIN)/setup-envtest

# add tools bin directory
PATH := $(PROJECT_BIN):$(PATH)

# docker executable
DOCKER ?= docker
# default Dockerfile
DOCKERFILE ?= Dockerfile
# container registry, default to github container registry
IMG_REGISTRY ?= ghcr.io
# container image organization
IMG_ORG ?= kubeflow
# container image version
IMG_VERSION ?= main
# container image repository
IMG_REPO ?= hub/server
# container image build path
BUILD_PATH ?= .
# container image
ifdef IMG
	IMG := ${IMG}
else ifdef IMG_REGISTRY
    IMG := ${IMG_REGISTRY}/${IMG_ORG}/${IMG_REPO}
else
    IMG := ${IMG_ORG}/${IMG_REPO}
endif

# Change Dockerfile path depending on IMG_REPO
# UI image uses repo root as context so COPY manifests/... for sample-catalog.yaml works
ifeq ($(IMG_REPO),hub/ui)
    DOCKERFILE := $(UI_PATH)/Dockerfile
	BUILD_PATH := $(PROJECT_PATH)
	ARGS := --build-arg UI_SOURCE_CODE=clients/ui/frontend --build-arg BFF_SOURCE_CODE=clients/ui/bff $(ARGS)
endif

# The BUILD_PATH is still the root
ifeq ($(IMG_REPO),hub/storage-initializer)
    DOCKERFILE := $(CSI_PATH)/Dockerfile.csi
endif

# The BUILD_PATH is still the root
ifeq ($(IMG_REPO),hub/controller)
    DOCKERFILE := $(CONTROLLER_PATH)/Dockerfile.controller
endif

model-registry: build

internal/converter/generated/converter.go: internal/converter/*.go
	${GOVERTER} gen github.com/kubeflow/hub/internal/converter/

.PHONY: gen/converter
gen/converter: internal/converter/generated/converter.go

api/openapi/model-registry.yaml: api/openapi/src/model-registry.yaml api/openapi/src/lib/*.yaml bin/yq
	scripts/merge_openapi.sh model-registry.yaml

api/openapi/catalog.yaml: api/openapi/src/catalog.yaml api/openapi/src/lib/*.yaml $(wildcard api/openapi/src/plugins/*.yaml) bin/yq
	scripts/merge_catalog_specs.sh catalog.yaml

# validate the openapi schema
.PHONY: openapi/validate
openapi/validate: bin/openapi-generator-cli bin/yq
	@scripts/merge_openapi.sh --check model-registry.yaml || (echo "api/openapi/model-registry.yaml is incorrectly formatted. Run 'make api/openapi/model-registry.yaml' to fix it."; exit 1)
	@scripts/merge_catalog_specs.sh --check catalog.yaml || (echo "api/openapi/catalog.yaml is incorrectly formatted. Run 'make api/openapi/catalog.yaml' to fix it."; exit 1)
	$(OPENAPI_GENERATOR) validate -i api/openapi/model-registry.yaml
	$(OPENAPI_GENERATOR) validate -i api/openapi/catalog.yaml

# generate the openapi server implementation
.PHONY: gen/openapi-server
gen/openapi-server: bin/openapi-generator-cli api/openapi/model-registry.yaml api/openapi/catalog.yaml openapi/validate internal/server/openapi/api_model_registry_service.go bin/goimports
	make -C catalog $@

internal/server/openapi/api_model_registry_service.go: bin/openapi-generator-cli api/openapi/model-registry.yaml
	./scripts/gen_openapi_server.sh

# generate the openapi schema model and client
.PHONY: gen/openapi
gen/openapi: bin/openapi-generator-cli api/openapi/model-registry.yaml api/openapi/catalog.yaml openapi/validate pkg/openapi/client.go
	make -C catalog $@

pkg/openapi/client.go: bin/openapi-generator-cli api/openapi/model-registry.yaml clean-pkg-openapi bin/goimports
	${OPENAPI_GENERATOR} generate \
		-i api/openapi/model-registry.yaml -g go -o pkg/openapi --package-name openapi \
		--ignore-file-override ./.openapi-generator-ignore --additional-properties=isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true,generateUnmarshalJSON=false
	$(PROJECT_BIN)/goimports -w pkg/openapi

# Start the MySQL database
.PHONY: start/mysql
start/mysql:
	./scripts/start_mysql_db.sh

# Stop the MySQL database
.PHONY: stop/mysql
stop/mysql:
	./scripts/teardown_mysql_db.sh

# Start the PostgreSQL database
.PHONY: start/postgres
start/postgres:
	./scripts/start_postgres_db.sh

# Stop the PostgreSQL database
.PHONY: stop/postgres
stop/postgres:
	./scripts/teardown_postgres_db.sh

# generate the gorm structs for MySQL
.PHONY: gen/gorm/mysql
gen/gorm/mysql: bin/golang-migrate start/mysql
	@(trap 'cd $(CURDIR) && $(MAKE) stop/mysql' EXIT; \
	$(GOLANG_MIGRATE) -path './internal/platform/db/mysql/migrations' -database 'mysql://root:root@tcp(localhost:3306)/model-registry' up && \
	cd gorm-gen && GOWORK=off go run main.go --db-type mysql --dsn 'root:root@tcp(localhost:3306)/model-registry?charset=utf8mb4&parseTime=True&loc=Local')

# generate the gorm structs for PostgreSQL
.PHONY: gen/gorm/postgres
gen/gorm/postgres: bin/golang-migrate start/postgres
	@(trap 'cd $(CURDIR) && $(MAKE) stop/postgres' EXIT; \
	$(GOLANG_MIGRATE) -path './internal/platform/db/postgres/migrations' -database 'postgres://postgres:postgres@localhost:5432/model-registry?sslmode=disable' up && \
	cd gorm-gen && GOWORK=off go run main.go --db-type postgres --dsn 'postgres://postgres:postgres@localhost:5432/model-registry?sslmode=disable' && \
	cd $(CURDIR) && ./scripts/remove_gorm_defaults.sh)

# generate the gorm structs (defaults to MySQL for backward compatibility)
# Use GORM_DB_TYPE=postgres to generate for PostgreSQL instead
.PHONY: gen/gorm
gen/gorm: bin/golang-migrate
ifeq ($(GORM_DB_TYPE),postgres)
	$(MAKE) gen/gorm/postgres
else
	$(MAKE) gen/gorm/mysql
endif

.PHONY: vet
vet:
	@echo "Running go vet on all packages..."
	@${GO} vet $$(${GO} list ./... | grep -vE 'github.com/kubeflow/hub/internal/(db|platform/db)/filter') && \
	echo "Checking filter packages (parser.go excluded due to participle struct tags)..." && \
	cd internal/db/filter && ${GO} build -o /dev/null . 2>&1 | grep -E "vet:|error:" || echo "✓ Filter package builds successfully" && \
	cd $(CURDIR)/internal/platform/db/filter && ${GO} build -o /dev/null . 2>&1 | grep -E "vet:|error:" || echo "✓ Platform filter package builds successfully"

.PHONY: clean/csi
clean/csi:
	rm -Rf ./mr-storage-initializer

.PHONY: clean-pkg-openapi
clean-pkg-openapi:
	while IFS= read -r file; do rm -f "pkg/openapi/$$file"; done < pkg/openapi/.openapi-generator/FILES
	make -C catalog $@

.PHONY: clean-internal-server-openapi
clean-internal-server-openapi:
	while IFS= read -r file; do rm -f "internal/server/openapi/$$file"; done < internal/server/openapi/.openapi-generator/FILES
	make -C catalog $@

.PHONY: clean
clean: clean-pkg-openapi clean-internal-server-openapi clean/csi
	rm -Rf ./model-registry internal/converter/generated/*.go

.PHONY: clean/odh
clean/odh:
	rm -Rf ./model-registry

bin/envtest:
	GOBIN=$(PROJECT_BIN) ${GO} install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.23

GOLANGCI_LINT ?= ${PROJECT_BIN}/golangci-lint
bin/golangci-lint:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(PROJECT_BIN) v2.9.0

GOVERTER ?= ${PROJECT_BIN}/goverter
bin/goverter:
	GOBIN=$(PROJECT_PATH)/bin ${GO} install github.com/jmattheis/goverter/cmd/goverter@v1.9.2

YQ ?= ${PROJECT_BIN}/yq
bin/yq:
	GOBIN=$(PROJECT_PATH)/bin ${GO} install github.com/mikefarah/yq/v4@v4.45.1

GOLANG_MIGRATE ?= ${PROJECT_BIN}/migrate
bin/golang-migrate:
	GOBIN=$(PROJECT_PATH)/bin ${GO} install -tags 'mysql,postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

GENQLIENT ?= ${PROJECT_BIN}/genqlient
bin/genqlient:
	GOBIN=$(PROJECT_PATH)/bin ${GO} install github.com/Khan/genqlient@v0.7.0

OPENAPI_GENERATOR ?= ${PROJECT_BIN}/openapi-generator-cli
NPM ?= "$(shell which npm)"
bin/openapi-generator-cli:
ifeq (, $(shell which ${NPM} 2> /dev/null))
	@echo "npm is not available please install it to be able to install openapi-generator"
	exit 1
endif
ifeq (, $(shell which ${PROJECT_BIN}/openapi-generator-cli 2> /dev/null))
	@{ \
	set -e ;\
	mkdir -p ${PROJECT_BIN} ;\
	mkdir -p ${PROJECT_BIN}/openapi-generator-installation ;\
	cd ${PROJECT_BIN} ;\
	${NPM} install -g --prefix ${PROJECT_BIN}/openapi-generator-installation @openapitools/openapi-generator-cli ;\
	ln -s openapi-generator-installation/bin/openapi-generator-cli openapi-generator-cli ;\
	}
endif

bin/goimports:
	GOBIN=$(PROJECT_PATH)/bin ${GO} install golang.org/x/tools/cmd/goimports@latest

.PHONY: clean/deps
clean/deps:
	rm -Rf bin/*

.PHONY: deps
deps: bin/golangci-lint bin/goverter bin/openapi-generator-cli bin/envtest

.PHONY: vendor
vendor:
	${GO} mod vendor

.PHONY: update/worksum
update/worksum:
	${GO} clean --modcache
	${GO} mod download

# WARNING: DO NOT DELETE THIS TARGET, USED BY Dockerfile!!!
.PHONY: build/prepare
build/prepare: gen vet lint

# WARNING: DO NOT DELETE THIS TARGET, USED BY Dockerfile!!!
.PHONY: build/compile
build/compile:
	${GO} build -buildvcs=false -o model-registry

# WARNING: DO NOT EDIT THIS TARGET DIRECTLY!!!
# Use build/prepare to add build prerequisites
# Use build/compile to add/edit go source compilation
# WARNING: Editing this target directly WILL affect the Dockerfile image build!!!
.PHONY: build
build: build/prepare build/compile

.PHONY: build/odh
build/odh: vet
	${GO} build -buildvcs=false

.PHONY: build/prepare/csi
build/prepare/csi: build/prepare lint/csi

.PHONY: build/compile/csi
build/compile/csi:
	cd ${CSI_PATH} && ${GO} build -buildvcs=false -o ../../mr-storage-initializer .

.PHONY: build/csi
build/csi: build/prepare/csi build/compile/csi

.PHONY: gen
gen: deps gen/openapi gen/openapi-server gen/converter

.PHONY: lint
lint: bin/golangci-lint
	${GOLANGCI_LINT} run main.go  --timeout 3m
	${GOLANGCI_LINT} run cmd/... internal/... ./pkg/...  --timeout 3m

.PHONY: lint/csi
lint/csi: bin/golangci-lint
	cd ${CSI_PATH} && ${GOLANGCI_LINT} run ./...

.PHONY: test
test:
	${GO} test $$(${GO} list ./internal/... ./pkg/... | grep -v controller)

.PHONY: test-nocache
test-nocache:
	${GO} test $$(${GO} list ./internal/... ./pkg/... | grep -v controller) -count=1

.PHONY: test-cover
test-cover:
	${GO} test $$(${GO} list ./internal/... ./pkg/... | grep -v controller) -coverprofile=coverage.txt
	${GO} tool cover -html=coverage.txt -o coverage.html

.PHONY: run/proxy
run/proxy: gen
	${GO} run main.go proxy --logtostderr=true

.PHONY: proxy
proxy: build
	./model-registry proxy --logtostderr=true

# login to docker
.PHONY: docker/login
docker/login:
ifdef IMG_REGISTRY
	$(DOCKER) login -u "${DOCKER_USER}" -p "${DOCKER_PWD}" "${IMG_REGISTRY}"
else
	$(DOCKER) login -u "${DOCKER_USER}" -p "${DOCKER_PWD}"
endif

# build docker image
.PHONY: image/build
image/build:
	${DOCKER} build ${BUILD_PATH} -f ${DOCKERFILE} -t ${IMG}:$(IMG_VERSION) $(ARGS)

# build docker image using buildx
# PLATFORMS defines the target platforms for the image be built to provide support to multiple
# architectures. (i.e. make docker-buildx). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: image/buildx
image/buildx:
ifeq ($(DOCKER),docker)
	# docker uses builder containers
	- $(DOCKER) buildx rm model-registry-builder
	$(DOCKER) buildx create --use --name model-registry-builder --platform=$(PLATFORMS)
	$(DOCKER) buildx build --push --platform=$(PLATFORMS) --tag ${IMG}:$(IMG_VERSION) -f ${DOCKERFILE} .
	$(DOCKER) buildx rm model-registry-builder
else ifeq ($(DOCKER),podman)
	# podman uses image manifests
	$(DOCKER) manifest create -a ${IMG}
	$(DOCKER) buildx build --platform=$(PLATFORMS) --manifest ${IMG}:$(IMG_VERSION) -f ${DOCKERFILE} .
	$(DOCKER) manifest push ${IMG}
	$(DOCKER) manifest rm ${IMG}
else
	$(error Unsupported container tool $(DOCKER))
endif

# push docker image
.PHONY: image/push
image/push:
	${DOCKER} push ${IMG}:$(IMG_VERSION)

all: model-registry

##  ------------------------------- ##
##  ----  Controller Targets   ---- ##
##  ------------------------------- ##

##@ Development

.PHONY: controller/manifests
controller/manifests: bin/controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=model-registry-manager-role crd webhook paths="{./cmd/controller/...}" output:crd:artifacts:config=manifests/options/controller/crd/bases output:rbac:dir=manifests/kustomize/options/controller/rbac

.PHONY: controller/generate
controller/generate: bin/controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="./cmd/controller/hack/boilerplate.go.txt" paths="{./cmd/controller/...}"

.PHONY: controller/fmt
controller/fmt: ## Run go fmt against code.
	cd cmd/controller && go fmt ./...

.PHONY: controller/vet
controller/vet: ## Run go vet against code.
	cd cmd/controller && go vet ./...

.PHONY: controller/test
controller/test: controller/manifests controller/generate controller/fmt controller/vet bin/envtest ## Run tests.
	cd cmd/controller && \
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(PROJECT_BIN) -p path)" go test $$(go list ./... | grep -vF /e2e) -coverprofile cover.out
	cd pkg/inferenceservice-controller && \
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(PROJECT_BIN) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: controller/build
controller/build: controller/manifests controller/generate controller/fmt controller/vet ## Build manager binary.
	cd cmd/controller && go build -o ../../bin/manager

.PHONY: controller/run
controller/run: controller/manifests controller/generate controller/fmt controller/vet ## Run a controller from your host.
	cd cmd/controller && go run main.go

# If you wish to build the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: controller/docker-build
controller/docker-build: ## Build docker image with the manager.
	$(DOCKER) build -t ${IMG} -f ./cmd/controller/Dockerfile.controller .

.PHONY: controller/docker-push
controller/docker-push: ## Push docker image with the manager.
	$(DOCKER) push ${IMG}

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: controller/docker-buildx
controller/docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' ./cmd/controller/Dockerfile.controller > Dockerfile.cross
	- $(DOCKER) buildx create --name controller-builder
	$(DOCKER) buildx use controller-builder
	- $(DOCKER) buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- $(DOCKER) buildx rm controller-builder
	rm Dockerfile.cross

.PHONY: controller/build-installer
controller/build-installer: controller/manifests controller/generate bin/kustomize ## Generate a consolidated YAML with CRDs and deployment.
	mkdir -p dist
	cd manifests/kustomize/options/controller/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build manifests/kustomize/options/controller/default > dist/install.yaml

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: controller/install
controller/install: controller/manifests bin/kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build manifests/kustomize/options/controller/crd | $(KUBECTL) apply -f -

.PHONY: controller/uninstall
controller/uninstall: controller/manifests bin/kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build manifests/kustomize/options/controller/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: controller/deploy
controller/deploy: controller/manifests bin/kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd manifests/kustomize/options/controller/manager && $(KUSTOMIZE) edit set image ghcr.io/kubeflow/hub/controller=${IMG}:${IMG_VERSION}
	$(KUSTOMIZE) build manifests/kustomize/options/controller/overlays/base | $(KUBECTL) apply -f -

.PHONY: controller/undeploy
controller/undeploy: bin/kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build manifests/kustomize/options/controller/overlays/base | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

##@ Docker Compose

# Auto-detect compose command: prefer podman-compose if available, fallback to docker-compose
COMPOSE_CMD ?= $(shell if command -v podman-compose >/dev/null 2>&1; then echo "podman-compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo "docker compose"; fi)

.PHONY: compose/up
compose/up: ## Start services using docker-compose.yaml with MySQL
	$(COMPOSE_CMD) --profile mysql up

.PHONY: compose/up/postgres
compose/up/postgres: ## Start services using docker-compose.yaml with PostgreSQL
	DB_TYPE=postgres $(COMPOSE_CMD) --profile postgres up

.PHONY: compose/down
compose/down: ## Stop services using docker-compose.yaml
	$(COMPOSE_CMD) --profile mysql --profile postgres down

.PHONY: compose/local/up
compose/local/up: ## Start services using docker-compose-local.yaml with MySQL (builds from source)
	$(COMPOSE_CMD) -f docker-compose-local.yaml --profile mysql up

.PHONY: compose/local/up/postgres
compose/local/up/postgres: ## Start services using docker-compose-local.yaml with PostgreSQL (builds from source)
	DB_TYPE=postgres $(COMPOSE_CMD) -f docker-compose-local.yaml --profile postgres up

.PHONY: compose/local/down
compose/local/down: ## Stop services using docker-compose-local.yaml
	$(COMPOSE_CMD) -f docker-compose-local.yaml --profile mysql --profile postgres down

.PHONY: compose/clean
compose/clean: ## Remove all Docker Compose volumes and networks
	$(COMPOSE_CMD) --profile mysql --profile postgres down -v --remove-orphans
	$(COMPOSE_CMD) -f docker-compose-local.yaml --profile mysql --profile postgres down -v --remove-orphans

##@ Tools

KUBECTL ?= kubectl
CONTROLLER_GEN ?= $(PROJECT_BIN)/controller-gen
KUSTOMIZE ?= $(PROJECT_BIN)/kustomize
CONTROLLER_TOOLS_VERSION ?= v0.16.4
KUSTOMIZE_VERSION ?= v5.5.0

.PHONY: bin/kustomize
bin/kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(PROJECT_BIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: bin/controller-gen
bin/controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(PROJECT_BIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(PROJECT_BIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
