DEFAULT_ENV_FILE := .env
ifneq ("$(wildcard $(DEFAULT_ENV_FILE))","")
include ${DEFAULT_ENV_FILE}
export $(shell sed 's/=.*//' ${DEFAULT_ENV_FILE})
endif

DEV_ENV_FILE := .env.development
ifneq ("$(wildcard $(DEV_ENV_FILE))","")
include ${DEV_ENV_FILE}
export $(shell sed 's/=.*//' ${DEV_ENV_FILE})
endif

ENV_FILE := .env.local
ifneq ("$(wildcard $(ENV_FILE))","")
include ${ENV_FILE}
export $(shell sed 's/=.*//' ${ENV_FILE})
endif

.PHONY: all
all: build

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)


############ Dev Environment ############

.PHONY: dev-install-dependencies
dev-install-dependencies:
	cd frontend && npm install

.PHONY: dev-bff
dev-bff:
	trap 'exit 0' INT; \
	cd bff && make run PORT=4000 MOCK_K8S_CLIENT=true MOCK_MR_CLIENT=true MOCK_MR_CATALOG_CLIENT=true DEV_MODE=true DEPLOYMENT_MODE=standalone 

.PHONY: dev-frontend
dev-frontend:
	cd frontend && DEPLOYMENT_MODE=standalone STYLE_THEME=mui-theme npm run start:dev

.PHONY: dev-start
dev-start:
	@trap 'exit 0' INT; \
	make -j 2 dev-bff dev-frontend

########### Dev Kubeflow ############
.PHONY: dev-start-kubeflow
dev-start-kubeflow:
	@trap 'exit 0' INT; \
	make -j 2 dev-bff-kubeflow dev-frontend-kubeflow

.PHONY: dev-frontend-kubeflow
dev-frontend-kubeflow:
	cd frontend && DEPLOYMENT_MODE=kubeflow STYLE_THEME=mui-theme npm run start:dev

.PHONY: dev-bff-kubeflow
dev-bff-kubeflow:
	cd bff && make run PORT=4000 MOCK_K8S_CLIENT=false MOCK_MR_CLIENT=false DEV_MODE=true DEPLOYMENT_MODE=kubeflow DEV_MODE_MODEL_REGISTRY_PORT=8085 DEV_MODE_CATALOG_PORT=8086

########### Dev Federated ###########
FEDERATED_FRONTEND_PORT ?= 9100
FEDERATED_BFF_PORT ?= 4000

.PHONY: dev-start-federated
dev-start-federated:
	@trap 'exit 0' INT; \
	make -j 2 dev-bff-federated dev-frontend-federated

.PHONY: dev-frontend-federated
dev-frontend-federated:
	cd frontend && AUTH_METHOD=user_token DEPLOYMENT_MODE=federated STYLE_THEME=patternfly PORT=$(FEDERATED_FRONTEND_PORT) PROXY_PORT=$(FEDERATED_BFF_PORT) npm run start:dev

.PHONY: dev-bff-federated
dev-bff-federated:
	cd bff && make run PORT=$(FEDERATED_BFF_PORT) MOCK_K8S_CLIENT=false MOCK_MR_CLIENT=false MOCK_MR_CATALOG_CLIENT=false DEV_MODE=true DEPLOYMENT_MODE=federated DEV_MODE_MODEL_REGISTRY_PORT=8085 DEV_MODE_CATALOG_PORT=8086 AUTH_METHOD=user_token AUTH_TOKEN_HEADER=x-forwarded-access-token AUTH_TOKEN_PREFIX="" INSECURE_SKIP_VERIFY=true

############ Build ############

DOCKER_BUILD_ARGS := --build-arg UI_SOURCE_CODE=clients/ui/frontend --build-arg BFF_SOURCE_CODE=clients/ui/bff

.PHONY: docker-build
docker-build:
	$(CONTAINER_TOOL) build -f Dockerfile $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=kubeflow --build-arg STYLE_THEME=mui-theme -t ${IMG_UI} ../..

.PHONY: docker-build-standalone
docker-build-standalone:
	$(CONTAINER_TOOL) build -f Dockerfile.standalone $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=standalone --build-arg STYLE_THEME=mui-theme -t ${IMG_UI_STANDALONE} ../..

.PHONY: docker-build-standalone-release
docker-build-standalone-release:
	$(CONTAINER_TOOL) build -f Dockerfile.standalone --target release $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=standalone --build-arg STYLE_THEME=mui-theme -t ${IMG_UI_STANDALONE} ../..

.PHONY: docker-build-federated
docker-build-federated:
	$(CONTAINER_TOOL) build -f Dockerfile $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=federated --build-arg STYLE_THEME=patternfly -t ${IMG_UI_FEDERATED} ../..

.PHONY: docker-buildx
docker-buildx:
	docker buildx build -f Dockerfile $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=kubeflow --build-arg STYLE_THEME=mui-theme --platform ${PLATFORM} -t ${IMG_UI} --push ../..

.PHONY: docker-buildx-standalone
docker-buildx-standalone:
	docker buildx build -f Dockerfile.standalone $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=standalone --build-arg STYLE_THEME=mui-theme --platform ${PLATFORM} -t ${IMG_UI_STANDALONE} --push ../..

.PHONY: docker-buildx-federated
docker-buildx-federated:
	docker buildx build -f Dockerfile $(DOCKER_BUILD_ARGS) --build-arg DEPLOYMENT_MODE=federated --build-arg STYLE_THEME=patternfly --platform ${PLATFORM} -t ${IMG_UI_FEDERATED} --push ../..

############ Push ############

.PHONY: docker-push
docker-push:
	${CONTAINER_TOOL} push ${IMG_UI}

.PHONY: docker-push-standalone
docker-push-standalone:
	${CONTAINER_TOOL} push ${IMG_UI_STANDALONE}

.PHONY: docker-push-federated
docker-push-federated:
	${CONTAINER_TOOL} push ${IMG_UI_FEDERATED}

############ Deployment ############

.PHONY: kind-deployment
kind-deployment:
	./scripts/deploy_kind_cluster.sh

.PHONY: kubeflow-deployment
kubeflow-deployment:
	./scripts/deploy_kubeflow_cluster.sh

############ Build ############	
.PHONY: frontend-build
frontend-build:
	cd frontend && DEPLOYMENT_MODE=kubeflow npm run build:prod

.PHONY: frontend-build-standalone
frontend-build-standalone:
	cd frontend && DEPLOYMENT_MODE=standalone npm run build:prod

.PHONY: bff-build
bff-build:
	cd bff && make build

.PHONY: openapi
openapi: ## Generate the BFF OpenAPI spec from Go annotations.
	$(MAKE) -C bff openapi

.PHONY: build
build: frontend-build bff-build

############ Run mocked ########
.PHONY: run-local-mocked
run-local-mocked: frontend-build-standalone bff-build
	rm -r ./bff/static-local-run && cp -r ./frontend/dist/ ./bff/static-local-run/ && cd bff && make run STATIC_ASSETS_DIR=./static-local-run MOCK_K8S_CLIENT=true DEV_MODE=true
