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
  - [Intent support across backends](#intent-support-across-backends)
  - [Migration path for mr_client.py](#migration-path-for-mr_clientpy)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [References](#references)
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
registry operations: creating registered models, model versions, and model artifacts, as
well as querying and updating artifact state. This tight coupling means organizations
using a different model registry (e.g. MLflow Model Registry) cannot use the async-upload
job without forking it.

The job's download, upload, and signing logic is already registry-agnostic. Only the
registry CRUD operations in `mr_client.py` and client instantiation in `entrypoint.py`
are Kubeflow-specific. Abstracting these behind an interface enables multi-registry
support with minimal disruption to the existing codebase.

### Goals

- Define a Python interface (protocol or ABC) for the registry operations the async-upload
  job requires
- Fold the ModelArtifact concept into ModelVersion at the interface level, since MLflow
  and other registries don't have a separate artifact entity
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
- Different registries have different entity models. Kubeflow has a three-level hierarchy
  (RegisteredModel → ModelVersion → ModelArtifact) while MLflow has two levels
  (RegisteredModel → ModelVersion, with the artifact URI stored directly on the version).
  The interface uses the two-level model, and the Kubeflow backend manages its artifact
  entity internally.
- `ArtifactState` (PENDING, LIVE, UNKNOWN) is a Kubeflow concept. The interface defines
  its own state enum. Backends that don't natively support artifact states can implement
  state tracking as a no-op or via custom properties/tags.

### Risks and Mitigations

- **Risk**: The abstract interface may not map cleanly to all registry backends.
  **Mitigation**: Validate the interface against an MLflow proof-of-concept before
  stabilizing. Use the MLflow implementation as a second data point before declaring
  the interface stable.

- **Risk**: Introducing an abstraction layer adds indirection and maintenance overhead.
  **Mitigation**: The abstraction is narrow (5 methods). The Kubeflow implementation is
  a thin wrapper around existing code. Net new code is small.

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

The initial 9-method interface was simplified to 5 methods after analyzing actual usage
patterns and cross-registry compatibility. See
[interface-simplification-analysis.md](interface-simplification-analysis.md) for the full
analysis.

Key simplifications:
- **ModelArtifact folded into ModelVersion** — MLflow doesn't have a separate artifact
  entity, and in the async-upload job artifacts are always 1:1 with versions. The
  Kubeflow backend manages its artifact entity internally.
- **`get_by_id` and `get_by_name` merged** into one `get_registered_model` method that
  accepts either identifier.
- **`create_model_version` and `create_model_artifact` merged** into one call that
  creates a version with its URI and state in a single operation.
- **`delete_registered_model` removed** — not called anywhere in the current codebase.

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
    uri: str | None = None
    state: ArtifactState = ArtifactState.UNKNOWN
    author: str | None = None
    description: str | None = None
    custom_properties: dict = field(default_factory=dict)
    # Kubeflow-specific: the backend can store an artifact_id internally
    artifact_id: str | None = None


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
        uri: str,
        author: str | None = None,
        description: str | None = None,
        state: ArtifactState = ArtifactState.UNKNOWN,
        model_format_name: str | None = None,
        model_format_version: str | None = None,
        storage_key: str | None = None,
        storage_path: str | None = None,
        service_account_name: str | None = None,
        custom_properties: dict | None = None,
    ) -> ModelVersion: ...

    def get_registered_model(
        self,
        model_id: str | None = None,
        name: str | None = None,
    ) -> RegisteredModel | None: ...

    def get_model_version(
        self,
        registered_model_id: str,
        version_name: str,
    ) -> ModelVersion | None: ...

    def update_model_version(
        self,
        model_version_id: str,
        state: ArtifactState | None = None,
        uri: str | None = None,
    ) -> ModelVersion: ...
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

The Kubeflow backend internally manages the ModelArtifact entity:
- `create_model_version` creates both a ModelVersion and a ModelArtifact, returning a
  `ModelVersion` with the `artifact_id` field populated.
- `update_model_version` translates state and URI updates to artifact-level operations
  (get artifact by ID, mutate, upsert).

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

### Intent support across backends

The async-upload job supports three intents: `create_model`, `create_version`, and
`update_artifact`. Not all intents are portable across backends due to fundamental
differences in how registries handle mutability.

| | Kubeflow | MLflow |
|---|---|---|
| Mutate artifact/version URI | Yes | No (`source` is immutable) |
| Delete a version | No | Yes |
| `create_model` intent | Supported | Supported |
| `create_version` intent | Supported | Supported |
| `update_artifact` intent | Supported | **Not supported** |

The `update_artifact` intent relies on URI mutability: set state to PENDING, re-upload
the model, then update the existing artifact with the new URI and set state to LIVE. This
works naturally in Kubeflow but has no direct equivalent in MLflow.

While an MLflow backend could simulate `update_artifact` by creating a new ModelVersion
with the new URI and archiving or deleting the old one, this has significant caveats:
- Version number changes (MLflow auto-increments), breaking anything referencing the old
  version number (deployment configs, aliases like `@champion`, CI pipelines)
- Creation timestamp resets, losing the history of when the model was first registered
- MLflow run linkage and tags are not automatically carried over
- What looks like a URI update in Kubeflow looks like a delete + create in MLflow,
  confusing external systems watching version events

Backends should declare which intents they support, and the job should validate this at
startup before doing any expensive download/upload work. Unsupported intents should fail
with a clear error message directing users to a supported alternative (e.g. "MLflow does
not support update_artifact — use create_version instead").

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
- 2026-04-14: Simplified interface from 9 methods to 5, added intent support analysis

## Drawbacks

- Adds a layer of indirection for users who only ever need Kubeflow Model Registry.
  However, the indirection is thin (data class translation + method delegation) and the
  current code already uses private APIs that could break on upgrades, so the wrapper
  provides stability benefits even in the single-backend case.
- The `update_artifact` intent cannot be supported by all backends (e.g. MLflow), meaning
  backend choice constrains available functionality. This is an inherent limitation of
  abstracting over registries with different mutability models.

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

### 4. Full 9-method interface mirroring Kubeflow's entity model

The initial design (see [v1-initial-proposal.md](v1-initial-proposal.md)) proposed 9
methods with a separate `ModelArtifact` data class. This was simplified after analysis
showed that MLflow and other registries don't have a separate artifact entity, and the
async-upload job always uses artifacts 1:1 with versions.

Rejected because: the larger interface surface is harder to implement for new backends,
and the `ModelArtifact` abstraction doesn't map to registries other than Kubeflow.

## References

- [v1-initial-proposal.md](v1-initial-proposal.md) — initial 9-method interface design
- [interface-simplification-analysis.md](interface-simplification-analysis.md) — analysis
  of usage patterns and cross-registry compatibility that led to the simplified interface
