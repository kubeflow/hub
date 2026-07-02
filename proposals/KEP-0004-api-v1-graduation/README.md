# KEP-0004: Graduate Model Registry and Catalog APIs to v1

## Summary

Graduate the Model Registry API from `v1alpha3` to `v1` and the Catalog API from `v1alpha1` to `v1`,
incorporating several breaking changes that address accumulated inconsistencies. The alpha versions
will remain available during a deprecation period to give users time to migrate.

## Motivation

The Model Registry and Catalog APIs have been in alpha long enough to identify problems that breaking changes alone can fix.

Three categories of problems need fixing before v1:

**Parameter casing is inconsistent.** Model Registry path parameters use smashed lowercase
(`registeredmodelId`, `modelversionId`, `inferenceserviceId`). Catalog path parameters use
snake\_case (`server_id`, `source_id`). URL paths use snake\_case in both APIs. Query parameters
and body fields use camelCase in both APIs. The path parameter inconsistency is the most
egregious: `/registered_models/{registeredmodelId}` reads as if someone forgot to add separators.

**orderBy behavior differs between API endpoints.** A long-standing
inconsistency causes some endpoints to return `400 Bad Request` when asked to
sort on an invalid field while others sort by the default field. The `400`
response is correct; v1 enforces it consistently.

**Experiment tracking has no meaningful adoption.** The feature is extensive,
but has no UI or evidence of meaningful adoption. Carrying it to v1 means
maintaining it for the lifetime of v1.

### Goals

- Ship v1 APIs that are internally consistent and consistent with each other
- Remove experiment tracking before v1 commits to supporting it long-term
- Provide a clear, time-bounded deprecation period for existing users

### Non-Goals

- Adding new endpoints or capabilities (v1 is a cleanup release)
- Changing URL path segment casing (stays snake\_case)
- Redesigning pagination or filtering
- Changing database schemas

## Proposal

### 1. Fix parameter casing

Adopt a single convention: **snake\_case for path parameters, camelCase for query parameters and
body fields.**

This matches what each layer already does for most parameters. The path parameter names in the
model-registry OpenAPI spec are the primary offenders; fixing them is the main practical change.
The remaining Catalog body/response outliers are `source_id` (in `Agent`, `CatalogModel`, and `MCPServer`) and `license_link` (in `MCPServer`).

See [Appendix A](#appendix-a-full-parameter-casing-changes) for the complete list of renames.

### 2. Specify consistent orderBy error behavior

Sorting on an unsupported field must return `400 Bad Request`. Some endpoints currently fall back to the default sort order, which masks client bugs and makes behavior unpredictable. The `400` response is correct; enforce it everywhere.

### 3. Remove experiment tracking

Experiment endpoints are not included in v1. No deprecation period within v1 — the alpha API
continues serving them during the alpha deprecation window. Users who depend on this functionality
should migrate to MLflow, Weights & Biases, or Kubeflow Pipelines, all of which offer mature
experiment tracking.

The `experimentId` and `experimentRunId` fields are also removed from `Artifact` schemas.

See [Appendix B](#appendix-b-experiment-tracking-removal-scope) for the full scope.

### 4. New v1 API paths

New paths:
- `/api/model_registry/v1/`
- `/api/model_catalog/v1/`
- `/api/mcp_catalog/v1/`

The v1 specs are new source files in `api/openapi/src/` alongside the alpha specs. Alpha specs
remain until removed at the end of the deprecation period.

The Python library and BFF will switch to the `v1` endpoints as soon as they are available, providing a seamless transition for most users.

### 5. Deprecation timeline

Both alpha versions are served alongside v1 for nine months. During this
period, alpha responses include `Deprecation` and `Sunset` HTTP headers per
[RFC 8594](https://www.rfc-editor.org/rfc/rfc8594), and the server logs a
warning on every alpha endpoint call. After nine months, alpha endpoints return
`410 Gone` and the alpha-specific code is removed. The `410 Gone` endpoints
will persist indefinitely.

## Implementation History

- 2026-05-11: KEP created

---

## Appendix A: Full Parameter Casing Changes

### Model Registry path parameters

| Path | Alpha parameter | v1 parameter |
|------|----------------|--------------|
| `/registered_models/{id}` | `registeredmodelId` | `registered_model_id` |
| `/registered_models/{id}/versions` | `registeredmodelId` | `registered_model_id` |
| `/model_versions/{id}` | `modelversionId` | `model_version_id` |
| `/model_versions/{id}/artifacts` | `modelversionId` | `model_version_id` |
| `/model_artifacts/{id}` | `modelartifactId` | `model_artifact_id` |
| `/inference_services/{id}` | `inferenceserviceId` | `inference_service_id` |
| `/inference_services/{id}/model` | `inferenceserviceId` | `inference_service_id` |
| `/inference_services/{id}/serves` | `inferenceserviceId` | `inference_service_id` |
| `/inference_services/{id}/version` | `inferenceserviceId` | `inference_service_id` |
| `/serving_environments/{id}` | `servingenvironmentId` | `serving_environment_id` |
| `/serving_environments/{id}/inference_services` | `servingenvironmentId` | `serving_environment_id` |

Experiment endpoints also currently use `experimentId` and `experimentrunId`, but those endpoints are removed from v1.

### Catalog path parameters

No changes needed. All catalog path parameters already use snake\_case (`server_id`, `source_id`,
`tool_name`, `model_name`).

### Body/response field changes

| API | Alpha field | v1 field |
|-----|------------|---------|
| Catalog | `source_id` (in `Agent`, `CatalogModel`, and `MCPServer` response bodies) | `sourceId` |
| Catalog | `license_link` (in `MCPServer` response bodies) | `licenseLink` |

All other body and response fields are already camelCase in both APIs.

---

## Appendix B: Experiment Tracking Removal Scope

### Endpoints removed from v1 spec (16 total)

- `GET /experiment` — find by params
- `GET /experiments`, `POST /experiments` — list, create
- `GET /experiments/{experiment_id}`, `PATCH /experiments/{experiment_id}` — get, update
- `GET /experiments/{experiment_id}/experiment_runs`, `POST /experiments/{experiment_id}/experiment_runs`
- `GET /experiment_run` — find by params
- `GET /experiment_runs`, `POST /experiment_runs` — list, create
- `GET /experiment_runs/{experiment_run_id}`, `PATCH /experiment_runs/{experiment_run_id}`
- `POST /experiment_runs/{experiment_run_id}/artifacts`, `GET /experiment_runs/{experiment_run_id}/artifacts`
- `GET /experiment_runs/metric_history`, `GET /experiment_runs/{experiment_run_id}/metric_history`

### Schemas removed (11 total)

`Experiment`, `ExperimentCreate`, `ExperimentUpdate`, `ExperimentList`, `ExperimentState`,
`ExperimentRun`, `ExperimentRunCreate`, `ExperimentRunUpdate`, `ExperimentRunList`,
`ExperimentRunState`, `ExperimentRunStatus`

### Fields removed from Artifact schemas

`experimentId`, `experimentRunId`

### Implementation scope (removed at alpha deletion)

- Go server: `internal/core/experiment*.go`, `internal/db/service/experiment*.go`,
  `internal/db/models/experiment*.go`, all experiment converters — approximately 3,900 lines
- Public Go API: 12 experiment interface methods in `pkg/api/api.go`
- Python client: `types/experiments.py`, `_experiments.py`, experiment methods from `core.py`
  and `_client.py`, generated experiment OpenAPI models

Database tables for experiments (stored in the MLMD schema as Context and Execution rows)
are not dropped automatically. A migration to clean them up will be provided as an optional step.
