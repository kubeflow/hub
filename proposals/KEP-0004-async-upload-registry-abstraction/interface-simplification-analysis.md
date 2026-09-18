# Simplifying the ModelRegistryClient Interface

## Current interface (9 methods)

The KEP-0004 interface proposes 9 methods derived directly from the Kubeflow client calls
in `mr_client.py`:

```
create_registered_model(name, owner, description, custom_properties) -> RegisteredModel
create_model_version(registered_model, version_name, author, ...) -> ModelVersion
create_model_artifact(model_version, artifact_name, uri, ...) -> ModelArtifact
get_registered_model_by_id(model_id) -> RegisteredModel
get_registered_model_by_name(name) -> RegisteredModel
get_model_version_by_params(registered_model_id, name) -> ModelVersion
get_model_artifact_by_id(artifact_id) -> ModelArtifact
update_model_artifact(artifact) -> ModelArtifact
delete_registered_model(model_id) -> None
```

## How each method is actually used

Tracing through `mr_client.py` to see what the job actually does with each method:

### `get_registered_model_by_name(name)`
- **Used in**: `validate_create_model_intent` (line 60)
- **Purpose**: Check if a model already exists before creating one
- **Pattern**: Existence check — only cares about "exists or not"

### `get_registered_model_by_id(model_id)`
- **Used in**: `validate_create_version_intent` (line 79), `create_version_and_artifact` (line 132)
- **Purpose**: Verify the parent model exists, then pass it to version creation
- **Pattern**: Fetch-then-use — needs the object for subsequent calls

### `get_model_version_by_params(registered_model_id, name)`
- **Used in**: `validate_create_version_intent` (line 88)
- **Purpose**: Check if a version already exists before creating one
- **Pattern**: Existence check — only cares about "exists or not"

### `get_model_artifact_by_id(artifact_id)`
- **Used in**: `set_artifact_pending` (line 39), `update_model_artifact_uri` (line 165)
- **Purpose**: Fetch artifact, mutate state/URI, then upsert it back
- **Pattern**: Fetch-mutate-save — always followed by `update_model_artifact`

### `create_registered_model(...)`
- **Used in**: `_create_registered_model` (line 186)
- **Purpose**: Create a new model

### `create_model_version(...)`
- **Used in**: `_create_version_and_artifact_for_model` (line 207)
- **Purpose**: Create a new version under a model

### `create_model_artifact(...)`
- **Used in**: `_create_version_and_artifact_for_model` (line 220)
- **Purpose**: Create an artifact under a version, then immediately set state to LIVE

### `update_model_artifact(artifact)`
- **Used in**: `set_artifact_pending` (line 45), `update_model_artifact_uri` (line 173),
  `_create_version_and_artifact_for_model` (line 239)
- **Purpose**: Persist state/URI changes to an artifact

### `delete_registered_model(model_id)`
- **Not actually used in production code**. Was listed in the interface but is not called
  anywhere in `mr_client.py`. May have been for cleanup on failure but currently unused.

## MLflow comparison

MLflow's model registry has a simpler entity model:

| Kubeflow | MLflow | Notes |
|----------|--------|-------|
| RegisteredModel | RegisteredModel | Similar concept |
| ModelVersion | ModelVersion | Similar concept |
| ModelArtifact | *(none)* | MLflow stores URI directly on ModelVersion (`source` field) |
| ArtifactState (PENDING/LIVE) | ModelVersion status (PENDING_REGISTRATION/READY) + stages (deprecated) | State lives on the version, not a separate artifact |

Key differences:
- **No ModelArtifact entity in MLflow** — the artifact URI is a property of ModelVersion
- **State management** is on ModelVersion, not on a separate artifact
- **Lookup by name, not ID** — MLflow identifies models by name and versions by number
- **Tags instead of custom_properties** — same concept, different name

## Proposed simplified interface (5 methods)

### Insight 1: ModelArtifact can be folded into ModelVersion

MLflow doesn't have ModelArtifact at all. In Kubeflow, ModelArtifact is always 1:1 with
ModelVersion in practice (at least in this job). We can make the artifact URI and state
properties of ModelVersion in our interface, and let the Kubeflow backend internally
manage the separate artifact entity.

### Insight 2: Separate get-by-id and get-by-name can be one method

`get_registered_model` can accept either an ID or a name. The backend resolves it. This
also maps cleanly to MLflow which only uses names.

### Insight 3: delete_registered_model is unused

Drop it. If we need it later, we add it.

### Insight 4: The fetch-mutate-save pattern for artifacts can become a single update call

Instead of `get_artifact -> mutate -> upsert`, expose `set_model_version_state(id, state, uri)`.

### Proposed interface

```python
class ModelRegistryClient(Protocol):

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

### Data classes

```python
class RegisteredModel:
    id: str
    name: str
    owner: str | None = None
    description: str | None = None
    custom_properties: dict = field(default_factory=dict)

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
```

### How each mr_client.py call maps to the new interface

| Current code | New interface call | Notes |
|---|---|---|
| `client._api.get_registered_model_by_params(name)` | `client.get_registered_model(name=name)` | Existence check |
| `client._api.get_registered_model_by_id(id)` | `client.get_registered_model(model_id=id)` | Fetch by ID |
| `client._api.get_model_version_by_params(rm_id, name)` | `client.get_model_version(rm_id, name)` | Existence check |
| `client._api.get_model_artifact_by_id(id)` + mutate + `upsert_model_artifact()` | `client.update_model_version(mv_id, state=..., uri=...)` | Kubeflow backend translates to artifact ops |
| `client._register_model(...)` | `client.create_registered_model(...)` | Same |
| `client._register_new_version(...)` + `client._register_model_artifact(...)` + set LIVE | `client.create_model_version(..., uri=uri, state=LIVE)` | Merges version + artifact creation |
| `client._api.delete_registered_model(id)` | *(removed)* | Unused |

### How MLflow maps to the same interface

| Interface method | MLflow implementation |
|---|---|
| `create_registered_model(name, ...)` | `mlflow_client.create_registered_model(name, tags=custom_properties)` |
| `create_model_version(rm, version_name, uri, ...)` | `mlflow_client.create_model_version(rm.name, source=uri, tags=...)` |
| `get_registered_model(name=name)` | `mlflow_client.get_registered_model(name)` |
| `get_registered_model(model_id=id)` | `mlflow_client.search_registered_models(filter_string=...)` or raise (MLflow doesn't use IDs) |
| `get_model_version(rm_id, version_name)` | `mlflow_client.get_model_version(rm_name, version_name)` |
| `update_model_version(mv_id, state=PENDING)` | `mlflow_client.transition_model_version_stage(name, version, "Staging")` or use aliases |
| `update_model_version(mv_id, state=LIVE, uri=...)` | URI is immutable in MLflow (set at creation), state maps to stage transition |

### The `update_artifact` intent and URI mutability

The two registries have inverted flexibility:

| | Kubeflow | MLflow |
|---|---|---|
| Mutate artifact/version URI | Yes | No (`source` is immutable) |
| Delete a version | No | Yes (`delete_model_version`) |

The `update_artifact` intent relies on URI mutability: set state to PENDING, re-upload
the model, then update the existing artifact with the new URI and set state to LIVE.
This works naturally in Kubeflow but has no direct equivalent in MLflow.

**Option A: Create a new version, delete or archive the old one**

The MLflow backend could implement `update_model_version` by creating a new ModelVersion
with the new URI and archiving (or deleting) the old one. This is the idiomatic MLflow
approach — versions are immutable, you make new ones.

However, this has significant caveats:
- **Version number changes** — MLflow auto-increments version numbers. Anything
  referencing the old version number (deployment configs, CI pipelines, MLflow aliases
  like `@champion`) would break or go stale.
- **Creation timestamp resets** — the new version gets a fresh timestamp, losing the
  history of when the model was first registered.
- **Run linkage lost** — MLflow versions can link to a `run_id`. The new version
  wouldn't have the original run association unless explicitly copied.
- **Tags not carried over** — any tags set on the old version (manually or by other
  systems) would need to be explicitly copied. Easy to miss.
- **Audit trail noise** — what looks like a URI update in Kubeflow looks like a
  delete + create in MLflow, confusing external systems watching version events.
- **Archiving instead of deleting** preserves references but doesn't solve the
  version number change, which is the core issue.

**Option B: Don't support `update_artifact` for MLflow**

The MLflow backend raises a clear error when the `update_artifact` intent is used,
directing users to `create_version` instead. This is simple, honest, and avoids the
caveats above.

**Recommendation**: Option B. The `update_artifact` intent is inherently tied to URI
mutability, which is a Kubeflow-specific capability. Forcing this into MLflow's model
creates fragile workarounds with surprising side effects. Backends should declare which
intents they support, and the job should validate this at startup before doing any
expensive download/upload work.

### Other MLflow considerations

1. **MLflow uses names, not IDs** — `get_registered_model(model_id=...)` is awkward for
   MLflow. The Kubeflow backend would resolve IDs; the MLflow backend would need the
   caller to provide names instead, or store a name->ID mapping.

2. **Stages are deprecated in MLflow** — `ArtifactState` mapping to MLflow stages is
   fragile. Newer MLflow uses aliases instead. The MLflow backend could use tags or
   aliases for state tracking.

### Cost analysis

- **No additional DB queries**: `get_registered_model(model_id=..., name=...)` is still
  one lookup, just with flexible parameters.
- **`create_model_version` merging version + artifact**: The Kubeflow backend internally
  does two creates (version + artifact) plus a state update — same as today, just hidden
  behind one interface call. No extra queries.
- **`update_model_version` replacing fetch-mutate-save**: The Kubeflow backend internally
  still does get + upsert on the artifact, but the caller makes one call instead of three.
  Same DB cost, simpler interface.

## Summary

| | Current | Proposed |
|---|---|---|
| Methods | 9 | 5 |
| Data classes | 3 (RegisteredModel, ModelVersion, ModelArtifact) | 2 (RegisteredModel, ModelVersion) |
| MLflow compatibility | Poor (ModelArtifact doesn't exist in MLflow) | Good (ModelVersion-centric like MLflow) |
| Breaking for Kubeflow | N/A | None — Kubeflow backend wraps artifact ops internally |
