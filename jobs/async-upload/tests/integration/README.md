# Integration Tests for Async-Upload Job

This directory contains integration tests for the async-upload job functionality, specifically testing the complete workflow from model creation through job execution and validation.

## Installation

To run the integration tests, you need to install the integration test dependencies:

```bash
# Install all dependencies including integration test dependencies
poetry install --with integration

# Or install just the integration group
poetry install --only integration

# No external CLI tools required - everything is pure Python!
```

## Dependencies Added

The integration tests require the following additional dependencies:

### Main Dependencies (added to `[tool.poetry.dependencies]`)

- **`requests`**: For HTTP calls (downloading models for upload to SeaweedFS)
- **`pyyaml`**: For YAML processing (kustomization files)

### Integration Test Dependencies (added to `[tool.poetry.group.integration.dependencies]`)

- **`kubernetes`**: Official Python client for Kubernetes API operations

### Pure Python Approach

The integration tests use a pure Python approach without external dependencies:

- **No subprocess calls**: All operations use Python libraries
- **No kustomize CLI**: YAML patching is done using pure Python dict operations
- **No shell commands**: Everything is handled through Python APIs

## Running the Tests

```bash
# Run integration tests only
poetry run pytest --integration tests/integration/ -v

# Run with environment variables
MR_HOST_URL=http://my-registry:8080 poetry run pytest --integration tests/integration/ -v

# Run all tests including integration
poetry run pytest --integration tests/ -v
```

## Test Requirements

The integration tests require:

1. **Model Registry service** running (default: `http://localhost:8080`)
2. **SeaweedFS service** running (default: `http://localhost:8333`)
3. **Kubernetes cluster** with kubectl configured
4. **OCI registry** for job artifact storage

## Environment Variables

- `MR_HOST_URL`: Model Registry URL (default: `http://localhost:8080`)
- `CONTAINER_IMAGE_URI`: Container image for the async-upload job (default: `ghcr.io/kubeflow/hub/job/async-upload:latest`)

## What the Tests Do

The integration tests validate the complete async-upload job workflow:

1. **Model Registry Setup**: Creates RegisteredModel, ModelVersion, and placeholder ModelArtifact
2. **File Operations**: Downloads ONNX model and uploads to SeaweedFS using pure Python
3. **Kubernetes Job**: Creates and applies job using pure Python YAML patching (no kustomize CLI)
4. **Validation**: Verifies job completion and artifact state updates using kubernetes client

## Debugging Failed Tests

If tests fail, check:

1. **Services are running**: Model Registry, SeaweedFS, Kubernetes cluster
2. **Connectivity**: Can reach all required services
3. **Permissions**: Kubernetes permissions for job creation
4. **Logs**: Integration test captures and displays pod logs on failure
