# KEP-0004: Async Upload Registry Abstraction

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Caveats](#caveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Package layout](#package-layout)
  - [Abstract interface](#abstract-interface)
  - [Factory](#factory)
  - [Kubeflow implementation](#kubeflow-implementation)
  - [Configuration](#configuration)
  - [Migration path for mr_client.py](#migration-path-for-mr_clientpy)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Introduce an abstract registry interface in the async-upload job so it can operate against
any model registry backend (Kubeflow Model Registry, MLflow, etc.) rather than being
hard-coded to the Kubeflow Model Registry Python client (`model-registry`). Model signing
continues to use `model_registry.signing` regardless of backend, as signing is orthogonal
to registry operations.

## Motivation

The [async-upload job](https://github.com/kubeflow/model-registry/tree/main/jobs/async-upload)
currently imports and directly uses the Kubeflow Model Registry Python client for all
registry operations: creating registered
models, model versions, and model artifacts, as well as querying and updating artifact
state. This tight coupling means organizations using a different model registry (e.g.
MLflow Model Registry) cannot use the async-upload job without forking it.

The job's download, upload, and signing logic is already registry-agnostic. Only the
registry CRUD operations in `mr_client.py` and client instantiation in `entrypoint.py`
are Kubeflow-specific. Abstracting these behind an interface enables multi-registry
support with minimal disruption to the existing codebase.

### Goals

- Define a Python interface (protocol or ABC) for the registry operations the async-upload
  job requires: create, read, update, and delete of registered models, model versions,
  and model artifacts
- Implement the Kubeflow Model Registry backend as the first (and default) implementation
  of this interface
- Provide a factory function that selects the backend at runtime based on configuration
- Maintain full backward compatibility — existing deployments using Kubeflow Model Registry
  require no configuration changes
- Keep model signing (`model_registry.signing`) independent of the registry backend

### Non-Goals

- Shipping a production-ready MLflow backend (a proof-of-concept may be included to
  validate the interface design, but production support is a follow-up effort)
- Abstracting storage utilities (`save_to_oci_registry`, `_connect_to_s3`) — these are
  upload/download concerns, not registry concerns
- Changing the signing workflow or its dependency on `model-registry[signing]`
- Modifying the Go server, OpenAPI specs, or any component outside `jobs/async-upload/`

## Proposal

Add a `registry/` subpackage under `jobs/async-upload/job/` containing:

1. **An abstract interface** defining the registry operations the job needs
2. **A Kubeflow implementation** wrapping the existing `model-registry` client
3. **A factory function** that returns the correct implementation based on config

Refactor `mr_client.py` and `entrypoint.py` to call through the interface rather than
directly importing `model_registry.ModelRegistry`.

### User Stories

#### Platform engineer using Kubeflow Model Registry

As a platform engineer running Kubeflow with Model Registry, I want the async-upload job
to continue working exactly as it does today with no configuration changes after this
refactor.

#### Platform engineer using MLflow

As a platform engineer whose organization uses MLflow Model Registry, I want to configure
the async-upload job to register models in MLflow so I can use Kubeflow's async upload
pipeline without migrating to Kubeflow Model Registry.

#### Contributor adding a new registry backend

As a contributor, I want a clear interface and an existing implementation to reference so
I can add support for a new model registry backend without understanding the full job
internals.

### Caveats

- The current code uses private APIs of the Kubeflow Model Registry client
  (`client._register_model`, `client._api.get_registered_model_by_id`, etc.). The
  abstraction provides an opportunity to encapsulate these behind a stable interface,
  reducing fragility even for the Kubeflow-only case.
- Different registries have different entity models. The interface must define
  backend-agnostic data classes for `RegisteredModel`, `ModelVersion`, and
  `ModelArtifact` that capture the common subset, with backend-specific extensions
  possible via custom properties.
- `ArtifactState` (PENDING, LIVE, UNKNOWN) is a Kubeflow concept. The interface should
  define its own state enum. Backends that don't natively support artifact states can
  implement state tracking as a no-op or via custom properties.

### Risks and Mitigations

- **Risk**: The abstract interface may not map cleanly to all registry backends (e.g.
  MLflow has a different entity hierarchy).
  **Mitigation**: Start with the Kubeflow interface shape and validate against MLflow
  before stabilizing the interface. Use the MLflow implementation as a second data point
  before declaring the interface stable.

- **Risk**: Introducing an abstraction layer adds indirection and maintenance overhead.
  **Mitigation**: The abstraction is narrow (approximately 9 methods). The Kubeflow
  implementation is a thin wrapper around existing code. Net new code is small.

- **Risk**: Backend-agnostic data classes lose type information or features specific to
  one registry.
  **Mitigation**: Data classes include a `custom_properties: dict` field for
  backend-specific metadata. Backends can also extend the base classes if needed.

## Design Details

### Package layout

```
jobs/async-upload/job/
  registry/
    __init__.py              # Exports: ModelRegistryClient, create_registry_client
    interface.py             # Abstract interface + backend-agnostic data classes
    factory.py               # create_registry_client(config) -> ModelRegistryClient
    kubeflow.py              # KubeflowRegistryClient implementation
```

### Abstract interface

```python
from __future__ import annotations

import enum
from dataclasses import dataclass, field
from typing import Protocol, runtime_checkable


class ArtifactState(enum.Enum):
    UNKNOWN = "unknown"
    PENDING = "pending"
    LIVE = "live"


@dataclass
class RegisteredModel:
    id: str
    name: str
    owner: str | None = None
    description: str | None = None
    custom_properties: dict = field(default_factory=dict)


@dataclass
class ModelVersion:
    id: str
    name: str
    registered_model_id: str
    author: str | None = None
    description: str | None = None
    custom_properties: dict = field(default_factory=dict)


@dataclass
class ModelArtifact:
    id: str
    name: str
    uri: str | None = None
    state: ArtifactState = ArtifactState.UNKNOWN
    model_format_name: str | None = None
    model_format_version: str | None = None
    custom_properties: dict = field(default_factory=dict)


@runtime_checkable
class ModelRegistryClient(Protocol):
    """Interface for model registry operations used by the async-upload job."""

    def create_registered_model(
        self,
        name: str,
        owner: str | None = None,
        description: str | None = None,
        custom_properties: dict | None = None,
    ) -> RegisteredModel: ...

    def create_model_version(
        self,
        registered_model: RegisteredModel,
        version_name: str,
        author: str | None = None,
        description: str | None = None,
        custom_properties: dict | None = None,
    ) -> ModelVersion: ...

    def create_model_artifact(
        self,
        model_version: ModelVersion,
        artifact_name: str,
        uri: str,
        model_format_name: str | None = None,
        model_format_version: str | None = None,
        storage_key: str | None = None,
        storage_path: str | None = None,
        service_account_name: str | None = None,
        custom_properties: dict | None = None,
    ) -> ModelArtifact: ...

    def get_registered_model_by_id(self, model_id: str) -> RegisteredModel: ...

    def get_registered_model_by_name(self, name: str) -> RegisteredModel: ...

    def get_model_version_by_params(
        self, registered_model_id: str, name: str
    ) -> ModelVersion: ...

    def get_model_artifact_by_id(self, artifact_id: str) -> ModelArtifact: ...

    def update_model_artifact(self, artifact: ModelArtifact) -> ModelArtifact: ...

    def delete_registered_model(self, model_id: str) -> None: ...
```

### Factory

```python
def create_registry_client(config: RegistryConfig) -> ModelRegistryClient:
    backend = getattr(config, "backend", "kubeflow")
    if backend == "kubeflow":
        from job.registry.kubeflow import KubeflowRegistryClient
        return KubeflowRegistryClient(config)
    raise ValueError(
        f"Unknown registry backend: {backend!r}. Supported: kubeflow"
    )
```

Lazy imports keep backend-specific dependencies optional. An MLflow backend would add
`mlflow` to the match without requiring `mlflow` to be installed for Kubeflow users.

### Kubeflow implementation

`KubeflowRegistryClient` wraps the existing `model_registry.ModelRegistry` client,
translating between the backend-agnostic data classes and the Kubeflow-specific types.
This is largely a reorganization of the current `mr_client.py` code.

### Configuration

Add a `--registry-backend` CLI argument (default: `kubeflow`):

```
--registry-backend {kubeflow}    # extensible to mlflow, etc.
```

Also settable via environment variable `MODEL_SYNC_REGISTRY_BACKEND`.

The existing `--registry-*` arguments (`--registry-server-address`, `--registry-port`,
`--registry-is-secure`, etc.) are specific to the Kubeflow backend. If `--registry-backend`
is set to something other than `kubeflow`, the job should warn if Kubeflow-specific args
are also provided. Other backends can define their own configuration via environment
variables or a `--registry-config` argument that accepts a JSON blob, allowing
backend-specific configuration without polluting the shared CLI argument namespace.

### Migration path for mr_client.py

Current `mr_client.py` functions (`validate_and_get_model_registry_client`,
`create_model_and_artifact`, `set_artifact_pending`, `update_model_artifact_uri`, etc.)
are refactored to accept a `ModelRegistryClient` instead of `ModelRegistry`. The
functions themselves remain — they orchestrate multi-step workflows (validate-then-create,
set-pending-then-upload-then-set-live) that sit above the interface.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Unit Tests

- `jobs/async-upload/tests/test_registry_interface.py`: Verify data class construction
  and `ArtifactState` enum behavior
- `jobs/async-upload/tests/test_registry_kubeflow.py`: Test `KubeflowRegistryClient`
  methods with mocked `model_registry.ModelRegistry`
- `jobs/async-upload/tests/test_registry_factory.py`: Test factory returns correct
  backend, raises on unknown backend
- `jobs/async-upload/tests/test_mr_client.py`: Update existing tests to mock against
  `ModelRegistryClient` protocol instead of `ModelRegistry` directly

#### E2E tests

Existing E2E tests (`make test-e2e` in `jobs/async-upload/`) exercise the full workflow
against a real Kubeflow Model Registry. These continue to validate the Kubeflow backend
end-to-end. Backend-specific E2E tests for future backends (e.g. MLflow) will be added
with those implementations.

### Graduation Criteria

N/A — this is an internal refactor with no user-facing API changes. The interface
stabilizes when a second backend implementation validates the abstraction.

## Implementation History

- 2026-04-02: KEP created

## Drawbacks

- Adds a layer of indirection for users who only ever need Kubeflow Model Registry.
  However, the indirection is thin (data class translation + method delegation) and the
  current code already uses private APIs that could break on upgrades, so the wrapper
  provides stability benefits even in the single-backend case.
- The interface is designed around Kubeflow's entity model (RegisteredModel → ModelVersion
  → ModelArtifact). Registries with a different hierarchy (e.g. MLflow's
  RegisteredModel → ModelVersion with artifacts as a property) may require adapters that
  feel unnatural.

## Alternatives

### 1. Adapter pattern without a shared interface

Each backend could provide its own adapter with backend-specific method signatures, and
`mr_client.py` would switch on backend type with `isinstance` checks.

Rejected because: this defeats the purpose of the abstraction. Callers would need to know
about every backend, and adding a new backend would require changes throughout
`mr_client.py`.

### 2. Configuration-driven approach (no code abstraction)

Map registry operations to HTTP calls defined in configuration (e.g. "to create a model,
POST to this URL with this payload template").

Rejected because: registry APIs differ in authentication, entity models, error handling,
and state management. A generic HTTP template cannot capture these differences without
becoming a DSL more complex than the interface it replaces.

### 3. Keep Kubeflow-only, let other registries fork

Users wanting MLflow support fork the job and replace `mr_client.py`.

Rejected because: this fragments the codebase, duplicates maintenance, and prevents
upstream contributions from benefiting all backends.
