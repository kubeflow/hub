# Makefile for Model Catalog Python Client

all: install tidy

.PHONY: help
help:
	@echo "Model Catalog Python Client Targets:"
	@echo ""
	@echo "  Setup & Build:"
	@echo "    install          - Generate OpenAPI client and install dependencies"
	@echo "    generate         - Regenerate catalog API client from OpenAPI spec"
	@echo "    clean            - Remove generated code and caches"
	@echo "    build            - Build the package"
	@echo ""
	@echo "  Testing:"
	@echo "    test-e2e         - Run E2E tests (requires running catalog)"
	@echo "    test-fuzz        - Run fuzz tests (requires running catalog)"
	@echo ""
	@echo "  Code Quality:"
	@echo "    lint             - Run linters (ruff + mypy)"
	@echo "    tidy             - Auto-fix code style issues"
	@echo ""
	@echo "  Nox Sessions (multi-Python testing):"
	@echo "    nox              - Run default nox sessions"
	@echo "    nox-lint         - Run lint on all Python versions"
	@echo "    nox-e2e          - Run E2E tests on all Python versions"
	@echo ""
	@echo "  Deployment (K8s with Kustomize):"
	@echo "    deploy           - Full local deployment (Kind + build + deploy)"
	@echo "    deploy-kind      - Create Kind cluster"
	@echo "    deploy-build     - Build Docker image"
	@echo "    deploy-load      - Load image into Kind cluster"
	@echo "    deploy-k8s       - Deploy to existing cluster (no Kind, no build)"
	@echo "    deploy-apply     - Apply kustomize manifests"
	@echo "    deploy-forward   - Start port-forward"
	@echo "    deploy-restart   - Rebuild and restart catalog (quick dev cycle)"
	@echo "    deploy-cleanup   - Remove deployment and Kind cluster"

# Configuration (can be overridden: make deploy CATALOG_PORT=9090)
DOCKER ?= docker
export CATALOG_NAMESPACE ?= model-catalog
export CATALOG_IMAGE ?= model-registry:catalog-test
export CLUSTER_NAME ?= catalog-e2e
export CATALOG_PORT ?= 8081

# Kind requires localhost/ prefix for podman-built images
ifeq ($(DOCKER),podman)
    KIND_IMAGE := localhost/$(CATALOG_IMAGE)
else
    KIND_IMAGE := $(CATALOG_IMAGE)
endif

# Kustomize overlay for E2E testing
KUSTOMIZE_E2E := ../../../manifests/kustomize/options/catalog/overlays/e2e
LOCALBIN := $(shell pwd)/bin
KUSTOMIZE := $(LOCALBIN)/kustomize
KUSTOMIZE_VERSION ?= 5.5.0

.PHONY: install
install: generate
	poetry install

.PHONY: generate
generate:
	@echo "Generating Catalog API client from OpenAPI spec..."
	@mkdir -p src/catalog_openapi
	../../../bin/openapi-generator-cli generate \
		-i ../../../api/openapi/catalog-v1.yaml \
		-g python \
		-o src/ \
		--package-name catalog_openapi \
		--additional-properties=library=urllib3,generateSourceCodeOnly=true,useOneOfDiscriminatorLookup=true
	@# Remove generated test and docs
	@rm -rf src/catalog_openapi/test src/catalog_openapi/docs
	@rm -f src/catalog_openapi_README.md
	@if [ -d patches ] && [ -n "$$(ls -A patches/*.patch 2>/dev/null)" ]; then \
		echo "Applying patches..."; \
		for patch_file in patches/*.patch; do \
			echo "  Applying $$patch_file..."; \
			git apply "$$patch_file" || exit 1; \
		done; \
	fi
	@echo "[OK] Generated catalog_openapi package"

.PHONY: clean
clean:
	@echo "Cleaning generated code and caches..."
	rm -rf src/catalog_openapi/
	rm -rf .pytest_cache/
	rm -rf .ruff_cache/
	rm -rf .mypy_cache/
	rm -rf .nox/
	rm -rf .coverage
	rm -rf .coverage.*
	rm -rf htmlcov/
	rm -rf dist/
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete
	@echo "[OK] Cleaned"

.PHONY: test-e2e
test-e2e:
	CATALOG_URL=$${CATALOG_URL:-http://localhost:$(CATALOG_PORT)} poetry run pytest --e2e -v -rA

.PHONY: test-fuzz
test-fuzz:
	CATALOG_URL=$${CATALOG_URL:-http://localhost:$(CATALOG_PORT)} poetry run pytest --fuzz -v

.PHONY: lint
lint:
	poetry run ruff check .
	poetry run mypy .

.PHONY: tidy
tidy:
	@echo "Fixing code style issues..."
	-poetry run ruff check --fix-only . src/catalog_openapi 2>/dev/null
	-poetry run ruff format src tests 2>/dev/null

.PHONY: build
build: install tidy
	poetry build

.PHONY: update
update:
	poetry lock

# Nox sessions for multi-Python version testing
.PHONY: nox
nox:
	poetry run nox

.PHONY: nox-lint
nox-lint:
	poetry run nox -s lint

.PHONY: nox-e2e
nox-e2e:
	poetry run nox -s e2e

# =============================================================================
# Deployment Targets
# =============================================================================

# Full local deployment: Kind cluster + build + deploy everything
.PHONY: deploy
deploy: deploy-kind deploy-build deploy-load deploy-k8s deploy-forward
	@echo ""
	@echo "========================================"
	@echo "[OK] Deployment complete!"
	@echo "========================================"
	@echo ""
	@echo "Catalog URL: http://localhost:$(CATALOG_PORT)"
	@echo ""
	@echo "Run tests:   make test-e2e"
	@echo "View logs:   kubectl logs -f deployment/model-catalog-server -n $(CATALOG_NAMESPACE)"
	@echo "Cleanup:     make deploy-cleanup"

# Deploy to existing cluster (no Kind, no build)
.PHONY: deploy-k8s
deploy-k8s: deploy-namespace deploy-apply

# Create Kind cluster
.PHONY: deploy-kind
deploy-kind:
	@echo "Setting up Kind cluster: $(CLUSTER_NAME)"
	@if kind get clusters 2>/dev/null | grep -q "$(CLUSTER_NAME)"; then \
		echo "Cluster $(CLUSTER_NAME) already exists, using it"; \
		kubectl config use-context "kind-$(CLUSTER_NAME)"; \
	else \
		kind create cluster -n "$(CLUSTER_NAME)"; \
	fi

# Build Docker image
.PHONY: deploy-build
deploy-build:
	@echo "Building Docker image: $(CATALOG_IMAGE)"
	cd ../../.. && $(DOCKER) build --progress=plain -t "$(CATALOG_IMAGE)" -f Dockerfile .
	@echo "[OK] Build complete"

# Load image into Kind cluster
.PHONY: deploy-load
deploy-load:
	@echo "Loading image into Kind cluster..."
	kind load docker-image -n "$(CLUSTER_NAME)" "$(CATALOG_IMAGE)" 2>&1 | tail -5
ifeq ($(DOCKER),podman)
	@echo "Using podman, image will be referenced as $(KIND_IMAGE)"
endif

# Create namespace
.PHONY: deploy-namespace
deploy-namespace:
	@echo "Creating namespace: $(CATALOG_NAMESPACE)"
	kubectl create namespace "$(CATALOG_NAMESPACE)" --dry-run=client -o yaml | kubectl apply -f -

# Install kustomize with pinned version
# Note: Retry logic is needed because the install script may hit GitHub API rate limits,
# especially when multiple jobs run in parallel in CI/CD (e.g., GitHub Actions).
.PHONY: deploy-kustomize
deploy-kustomize: $(LOCALBIN)
	@if [ ! -f $(KUSTOMIZE) ]; then \
		echo "Installing kustomize $(KUSTOMIZE_VERSION) to $(LOCALBIN)..."; \
		for i in 1 2 3; do \
			echo "Attempt $$i of 3..."; \
			curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash -s $(KUSTOMIZE_VERSION) $(LOCALBIN) && break || \
			{ echo "Installation failed, retrying in 5 seconds..."; sleep 5; }; \
		done; \
		if [ ! -f $(KUSTOMIZE) ]; then \
			echo "ERROR: Kustomize installation failed after 3 attempts"; \
			exit 1; \
		fi; \
		echo "[OK] Kustomize $(KUSTOMIZE_VERSION) installed"; \
	else \
		echo "Kustomize already installed at $(KUSTOMIZE)"; \
	fi

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Apply kustomize manifests (includes postgres, secrets, configmaps, catalog)
.PHONY: deploy-apply
deploy-apply: deploy-namespace deploy-kustomize
	@echo "Applying kustomize manifests..."
	cd "$(KUSTOMIZE_E2E)" && \
	$(KUSTOMIZE) edit set image "ghcr.io/kubeflow/hub/server=$(KIND_IMAGE)"
	kubectl apply -k "$(KUSTOMIZE_E2E)"
	@echo "Waiting for PostgreSQL to be ready..."
	kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgres -n "$(CATALOG_NAMESPACE)" --timeout=120s
	@echo "[OK] PostgreSQL is ready"
	@echo "Waiting for Catalog to be ready..."
	kubectl wait --for=condition=available deployment/model-catalog-server -n "$(CATALOG_NAMESPACE)" --timeout=120s
	@echo "[OK] Catalog service is ready"

# Start port-forward
.PHONY: deploy-forward
deploy-forward:
	@echo "Starting port-forward on port $(CATALOG_PORT)..."
	@# Stop any existing port-forward first to avoid stale PIDs
	@if [ -f .port-forward.pid ]; then \
		kill $$(cat .port-forward.pid) 2>/dev/null || true; \
		rm -f .port-forward.pid; \
	fi
	@kubectl port-forward -n "$(CATALOG_NAMESPACE)" svc/model-catalog "$(CATALOG_PORT)":8080 > /dev/null 2>&1 & echo $$! > .port-forward.pid
	@sleep 3
	@if curl -s "http://localhost:$(CATALOG_PORT)/api/model_catalog/v1/sources" > /dev/null 2>&1; then \
		echo "[OK] Catalog accessible at http://localhost:$(CATALOG_PORT)"; \
	else \
		echo "[WARN] Port-forward may not be ready. Try: kubectl port-forward -n $(CATALOG_NAMESPACE) svc/model-catalog $(CATALOG_PORT):8080"; \
	fi

# Stop port-forward
.PHONY: deploy-forward-stop
deploy-forward-stop:
	@echo "Stopping port-forward..."
	@if [ -f .port-forward.pid ]; then \
		kill $$(cat .port-forward.pid) 2>/dev/null || true; \
		rm -f .port-forward.pid; \
	fi

# Cleanup everything
.PHONY: deploy-cleanup
deploy-cleanup: deploy-forward-stop
	@echo "Cleaning up deployment..."
	@if kubectl get namespace "$(CATALOG_NAMESPACE)" &>/dev/null; then \
		kubectl delete namespace "$(CATALOG_NAMESPACE)" --timeout=60s || true; \
	fi
	@if kind get clusters 2>/dev/null | grep -q "$(CLUSTER_NAME)"; then \
		echo "Deleting Kind cluster: $(CLUSTER_NAME)"; \
		kind delete cluster -n "$(CLUSTER_NAME)"; \
	fi
	@echo "[OK] Cleanup complete"

# Shortcut: redeploy catalog only (after code changes)
.PHONY: deploy-restart
deploy-restart: deploy-build deploy-load
	kubectl rollout restart deployment/model-catalog-server -n "$(CATALOG_NAMESPACE)"
	kubectl rollout status deployment/model-catalog-server -n "$(CATALOG_NAMESPACE)"
	@echo "[OK] Catalog restarted"
