# Evaluation benchmark data contract

This document proposes the `evaluations.ndjson` record contract consumed by the
AI Hub model-catalog service for evaluation insights. The file remains in each
model directory beside `metadata.json`. Each line represents one benchmark
result from one evaluation run.

## Proposed record

```json
{"id":"result-7f1f","model_id":"RedHatAI/example-model","run_id":"run-2026-09-17-001","evaluation":"General language evaluation","category":"general-llm","benchmark":"mmlu","description":"Measures multitask language understanding.","result":0.72,"result_metric":"accuracy","created_at":1789646400000,"updated_at":1789646460000}
```

Each NDJSON line can be validated with this JSON Schema:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "urn:kubeflow:catalog:evaluation-result:v1",
  "title": "Catalog evaluation result",
  "type": "object",
  "required": [
    "id",
    "model_id",
    "run_id",
    "evaluation",
    "category",
    "benchmark",
    "description",
    "result",
    "result_metric",
    "created_at"
  ],
  "properties": {
    "id": { "type": "string", "minLength": 1 },
    "model_id": { "type": "string", "minLength": 1 },
    "run_id": { "type": "string", "minLength": 1 },
    "evaluation": { "type": "string", "minLength": 1 },
    "category": {
      "type": "string",
      "pattern": "^[a-z0-9]+(?:-[a-z0-9]+)*$"
    },
    "benchmark": { "type": "string", "minLength": 1 },
    "description": { "type": "string", "minLength": 1 },
    "result": { "type": "number" },
    "result_metric": { "type": "string", "minLength": 1 },
    "created_at": { "type": "integer", "minimum": 0 },
    "updated_at": { "type": "integer", "minimum": 0 }
  },
  "additionalProperties": true
}
```

## Fields

| Field | Type | Required | Contract |
| --- | --- | --- | --- |
| `id` | string | Yes | Globally unique, stable identifier for this benchmark result. |
| `model_id` | string | Yes | Exact match for the sibling `metadata.json` `id`; includes the provider/namespace and model name. |
| `run_id` | string | Yes | Stable identifier shared by all benchmark results produced by one evaluation run. |
| `evaluation` | string | Yes | Human-readable evaluation or collection name shown in the UI. |
| `category` | string | Yes | Stable, lowercase kebab-case category identifier, such as `general-llm`, `safety-testing`, or `software-engineering`. |
| `benchmark` | string | Yes | Stable benchmark identifier, such as `mmlu` or `ifeval`. |
| `description` | string | Yes | Human-readable explanation of what the evaluation or benchmark measures. |
| `result` | number | Yes | Finite numeric result. Producers must not encode the number as a string. |
| `result_metric` | string | Yes | Unit or metric needed to interpret `result`, such as `accuracy`, `accuracy-percent`, or `pass-rate`. |
| `created_at` | integer | Yes | Evaluation result timestamp in Unix epoch milliseconds. |
| `updated_at` | integer | No | Last update timestamp in Unix epoch milliseconds. |

Producers may append optional scalar properties. Suggested optional properties
include `provider_id`, `benchmark_url`, `higher_is_better`, and
`execution_context`. New required properties need a coordinated contract change.

## Catalog artifact identity

The producer's `id` identifies the benchmark result and is served unchanged as
`customProperties.id`. The same result may be ingested for separate catalog
models, such as the same model listed in two sources.
Performance records that use the supported `config_id` fallback also expose that
resolved producer ID in `customProperties.id`, retaining `config_id` as well.

For evaluation, performance, and security records, the catalog generates both
the artifact `name` and `externalId` using
`catalog-metrics:v1:<modelID>:<metricsType>:<sha256(producerID)>`. The digest is
the full lowercase hexadecimal SHA-256 hash of the original ID's UTF-8 bytes.
These identities distinguish catalog models and metrics types, fit the database
key length, and preserve case-sensitive producer IDs. Accuracy aggregates retain
their existing `accuracy-metrics-model-<modelID>` identity.

Consumers should correlate producer data using `customProperties.id`; filters
on `externalId` use the scoped value returned by the API. Deleting and recreating
a catalog model can change its artifact identities because its database ID can
change.

This changes the returned `name` and `externalId` values for record-based metrics,
without changing the API schema or producer record format. Existing artifacts
retain their legacy values until the normal model reload deletes and recreates
them. During that transition, ingestion recognizes legacy records only for their
owning catalog model and metrics type. No database migration or reset is needed.
If a legacy producer ID was deliberately chosen to equal a new scoped identity,
ingestion reports the database conflict rather than silently dropping the new
result. A normal model reload replaces the legacy identity before retrying.

## Producer migration

For current Model Validation records:

- Rename `score` to `result`.
- Rename `score_metric` to `result_metric`.
- Add `run_id`, `evaluation`, `category`, and `description`.
- Preserve `id`, `model_id`, `benchmark`, `execution_context`, `created_at`,
  and `updated_at`.

During rollout, producers may temporarily emit both the old and new score field
names. The catalog accepts `result` for the existing aggregate accuracy view as
well as for detailed evaluation results.

## Validation and correlation rules

- `model_id` must exactly equal the sibling `metadata.json` `id`. Directory
  placement alone is not sufficient correlation.
- `id` identifies one benchmark result; `run_id` groups results belonging to the
  same evaluation execution.
- A file may contain multiple records for the same benchmark when they have
  different `id` values. This is how run history is retained.
- Producers must retain historical records needed by the UI rather than writing
  only the latest score.
- Duplicate `id` values, missing required fields, non-numeric results, invalid
  timestamps, and records whose `model_id` does not match `metadata.json` are
  invalid.
- When present, `updated_at` must not precede `created_at`.
- Category identifiers are API/filter values. Presentation labels and badge
  colors remain UI concerns.

## Compatibility

Legacy records containing only `benchmark` and `score` continue to feed the
existing aggregate `accuracy-metrics` artifact. The model-catalog service emits
record-level `evaluation-metrics` artifacts only for records satisfying this
contract. This permits the BenchmarkDataImage producer and consumer to roll out
independently without breaking existing accuracy consumers.

The model-catalog API represents `created_at` and `updated_at` as the artifact's
standard timestamps. All other record fields are returned as typed custom
properties on a `CatalogMetricsArtifact` whose `metricsType` is
`evaluation-metrics`.
