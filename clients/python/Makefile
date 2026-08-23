all: install tidy

IMG_VERSION ?= latest
IMG ?= ghcr.io/kubeflow/hub/server
BUILD_IMAGE ?= true # whether to build the MR server image

.PHONY: install
install:
	../../bin/openapi-generator-cli generate -i ../../api/openapi/model-registry-v1.yaml -g python -o src/ --package-name mr_openapi --additional-properties=library=asyncio,generateSourceCodeOnly=true,useOneOfDiscriminatorLookup=true --template-dir templates/
	mv src/mr_openapi_README.md src/mr_openapi/README.md
	git apply patches/*
	poetry install

.PHONY: clean
clean:
	rm -rf src/mr_openapi

.PHONY: deploy-latest-mr
deploy-latest-mr:
	cd ../../ && \
	$(if $(filter true,$(BUILD_IMAGE)),\
		IMG_VERSION=${IMG_VERSION} IMG=${IMG} make image/build ARGS="--load$(if ${DEV_BUILD}, --target dev-build)" && \
	) \
	LOCAL=1 IMG=$(IMG):$(IMG_VERSION) ./scripts/deploy_on_kind.sh
# TODO RHOAIENG-30453 align consistency ./scripts/deploy_on_kind.sh uses IMG with :tag, Vs, Makefile(s) and ci/GHA we use IMG without trailing :tag
	kubectl port-forward -n kubeflow services/model-registry-service 8080:8080 > /dev/null 2>&1 & echo $$! >> .port-forwards.pid
	@echo "Waiting for model-registry port-forward (8080)..."
	@for i in $$(seq 1 30); do curl -sf http://localhost:8080/api/model_registry/v1/registered_models > /dev/null 2>&1 && break || sleep 1; done
	@curl -sf http://localhost:8080/api/model_registry/v1/registered_models > /dev/null 2>&1 || \
		(echo "ERROR: model-registry not reachable on localhost:8080 after 30s"; exit 1)

.PHONY: deploy-test-minio
deploy-test-minio:
	cd ../../ && ./scripts/deploy_minio_on_kind.sh
	kubectl port-forward -n minio svc/minio 9000:9000 > /dev/null 2>&1 & echo $$! >> .port-forwards.pid
	@echo "Waiting for minio port-forward (9000)..."
	@for i in $$(seq 1 30); do curl -sf http://localhost:9000/minio/health/live > /dev/null 2>&1 && break || sleep 1; done
	@curl -sf http://localhost:9000/minio/health/live > /dev/null 2>&1 || \
		(echo "ERROR: minio not reachable on localhost:9000 after 30s"; exit 1)

.PHONY: deploy-local-registry
deploy-local-registry:
	cd ../../ && ./scripts/deploy_local_kind_registry.sh
	kubectl port-forward service/distribution-registry-test-service -n local-oci-registry-ns 5001:5001 > /dev/null 2>&1 & echo $$! >> .port-forwards.pid
	@echo "Waiting for local-registry port-forward (5001)..."
	@for i in $$(seq 1 30); do curl -sf http://localhost:5001/v2/ > /dev/null 2>&1 && break || sleep 1; done
	@curl -sf http://localhost:5001/v2/ > /dev/null 2>&1 || \
		(echo "ERROR: local registry not reachable on localhost:5001 after 30s"; exit 1)

.PHONY: test-e2e
test-e2e: deploy-latest-mr deploy-local-registry deploy-test-minio
	@echo "Starting test-e2e"
	-$(MAKE) test-e2e-run; STATUS=$$?
	$(MAKE) test-e2e-cleanup
	@exit $$STATUS

.PHONY: test-fuzz
test-fuzz: deploy-latest-mr deploy-local-registry deploy-test-minio
	@echo "Cleaning database before fuzz tests..."
	cd ../../ && ./scripts/cleanup.sh
	poetry install --all-extras
	@echo "Starting stateless fuzz tests with parallel execution"; \
	set -a; . ../../scripts/manifests/minio/.env; set +a; \
	poetry run pytest tests/fuzz_api/model_registry/test_mr_stateless.py tests/fuzz_api/model_catalog/test_catalog_stateless.py --fuzz -n auto -v --hypothesis-show-statistics; \
	STATELESS=$$?; \
	echo "Cleaning database before stateful fuzz tests..."; \
	cd ../../ && ./scripts/cleanup.sh && cd clients/python; \
	echo "Running stateful fuzz tests sequentially"; \
	set -a; . ../../scripts/manifests/minio/.env; set +a; \
	poetry run pytest tests/fuzz_api/model_registry/test_mr_stateful.py tests/fuzz_api/model_catalog/test_catalog_stateful.py --fuzz -v --hypothesis-show-statistics; \
	STATEFUL=$$?; \
	rm -f ../../scripts/manifests/minio/.env; \
	$(MAKE) test-e2e-cleanup; \
	if [ $$STATELESS -ne 0 ] || [ $$STATEFUL -ne 0 ]; then \
		echo "Fuzz tests failed (stateless=$$STATELESS, stateful=$$STATEFUL)"; exit 1; \
	fi

.PHONY: test-e2e-run
test-e2e-run:
	@echo "Ensuring all extras are installed..."
	poetry install --all-extras
	@echo "Running tests..."
	@set -a; . ../../scripts/manifests/minio/.env; set +a; \
	poetry run pytest --e2e -s -rA --html=report.html --self-contained-html
	@rm -f ../../scripts/manifests/minio/.env

.PHONY: test-e2e-cleanup
test-e2e-cleanup:
	@echo "Cleaning up database..."
	cd ../../ && ./scripts/cleanup.sh
	@echo "Cleaning up port-forward processes..."
	@if [ -f .port-forwards.pid ]; then \
		kill $$(cat .port-forwards.pid) || true; \
		rm -f .port-forwards.pid; \
	fi

.PHONY: undeploy-minio
undeploy-minio:
	@echo "Undeploy Minio..."
	cd ../../ && ./scripts/undeploy_minio.sh

.PHONY: undeploy-local-kind-registry
undeploy-local-kind-registry:
	@echo "Undeploy Local Kind Registry..."
	cd ../../ && ./scripts/undeploy_local_registry.sh

.PHONY: test
test:
	poetry run pytest -s -rA

.PHONY: lint
lint:
	poetry run ruff check
	poetry run mypy .

.PHONY: tidy
tidy:
	# Fix code issues everywhere, including src/mr_openapi/ (excluded by default in pyproject.toml)
	poetry run ruff check --fix-only --unsafe-fixes . src/mr_openapi
	poetry run ruff format src src/mr_openapi

.PHONY: build
build: install tidy
	poetry build

.PHONY: publish
publish: build
	poetry publish --build -u ${PYPI_USERNAME} -p ${PYPI_PASSWORD}

.PHONY: update
update:
	poetry lock
