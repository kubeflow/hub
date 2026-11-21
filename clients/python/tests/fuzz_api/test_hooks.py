from pathlib import Path

import pytest
import schemathesis
from schemathesis.hooks import HookContext

from tests.fuzz_api.conftest import map_body, map_case


@pytest.fixture
def registry_schema(root: Path):
    return schemathesis.openapi.from_path(root / "api/openapi/model-registry-v1.yaml", config=schemathesis.Config())


@pytest.mark.parametrize("path", [
    "/api/model_registry/v1/registered_models",
    "/api/model_registry/v1/model_versions",
    "/api/model_registry/v1/registered_models/{registered_model_id}/versions",
    "/api/model_registry/v1/serving_environments",
])
def test_map_case_sanitizes_bodies_without_generation_hooks(registry_schema, path):
    operation = registry_schema[path]["POST"]
    case = operation.Case(
        media_type="application/json",
        body={
            "name": "coverage\x00name",
            "customProperties": {
                "x-schemathesis-additional": {
                    "metadataType": "MetadataProtoValue", "proto_value": "", "type": "",
                },
            },
            "unknownField": "extra",
        },
    )

    map_case(HookContext(operation=operation), case)

    assert case.body == {
        "name": "coveragename",
        "customProperties": {
            "x-schemathesis-additional": {
                "metadataType": "MetadataStringValue", "string_value": "placeholder",
            },
        },
    }


def test_map_case_sanitizes_queries_without_generation_hooks(registry_schema):
    operation = registry_schema["/api/model_registry/v1/registered_models"]["GET"]
    case = operation.Case(query={"nextPageToken": "invalid", "filterQuery": "invalid", "name": "a\x00b"})

    map_case(HookContext(operation=operation), case)

    assert case.query == {"name": "ab"}


@pytest.mark.parametrize(("path", "method", "field", "limit"), [
    ("/api/model_registry/v1/model_versions/{model_version_id}", "PATCH", "externalId", 255),
    ("/api/model_registry/v1/model_versions", "POST", "name", 218),
    ("/api/model_registry/v1/registered_models", "POST", "name", 255),
    ("/api/model_registry/v1/registered_models", "POST", "externalId", 255),
    ("/api/model_registry/v1/registered_models/{registered_model_id}/versions", "POST", "name", 218),
    ("/api/model_registry/v1/artifacts", "POST", "name", 218),
    ("/api/model_registry/v1/serving_environments", "POST", "name", 255),
])
@pytest.mark.parametrize("length", [218, 219, 255, 256, 65536])
def test_map_case_limits_database_identifiers(registry_schema, path, method, field, limit, length):
    operation = registry_schema[path][method]
    value = "界" * length
    description = "a" * 65536
    case = operation.Case(
        media_type="application/json",
        body={field: value, "description": description, "artifactType": "doc-artifact"},
    )

    map_case(HookContext(operation=operation), case)

    assert case.body[field] == value[:limit]
    assert case.body["description"] == description


@pytest.mark.parametrize(("value", "expected"), [
    pytest.param("a" * 65535, "a" * 65535, id="ascii-at-limit"),
    pytest.param("a" * 65536, "a" * 65535, id="ascii-over-limit"),
    pytest.param("界" * 21845, "界" * 21845, id="unicode-at-limit"),
    pytest.param("界" * 21846, "界" * 21845, id="unicode-over-limit"),
    pytest.param("a" + "界" * 21845, "a" + "界" * 21844, id="partial-unicode-character"),
])
def test_map_case_limits_artifact_uri_bytes(registry_schema, value, expected):
    operation = registry_schema["/api/model_registry/v1/artifacts"]["POST"]
    case = operation.Case(
        media_type="application/json",
        body={"artifactType": "doc-artifact", "uri": value},
    )

    map_case(HookContext(operation=operation), case)

    assert case.body["uri"] == expected


def test_map_case_uses_v1_model_artifact_path(registry_schema):
    operation = registry_schema["/api/model_registry/v1/model_artifacts/{model_artifact_id}"]["PATCH"]
    case = operation.Case(
        media_type="application/json", path_parameters={"model_artifact_id": "invalid"},
        body={"description": "updated", "source": "unknown field"},
    )

    map_case(HookContext(operation=operation), case)

    assert case.body == {"description": "updated"}
    assert case.path_parameters == {"model_artifact_id": "1"}


@pytest.mark.parametrize(("step", "expected"), [("invalid", 0), ("12", 12), (2**80, 2**63 - 1)])
def test_map_case_coerces_metric_update_step(registry_schema, monkeypatch, step, expected):
    monkeypatch.setattr("tests.fuzz_api.conftest._get_artifact_type", lambda case: "metric")
    operation = registry_schema["/api/model_registry/v1/artifacts/{id}"]["PATCH"]
    case = operation.Case(media_type="application/json", body={"artifactType": "metric", "step": step})

    map_case(HookContext(operation=operation), case)

    assert case.body == {"artifactType": "metric", "step": expected}


def test_map_body_preserves_catalog_uploads(root: Path):
    schema = schemathesis.openapi.from_path(root / "api/openapi/catalog-v1.yaml", config=schemathesis.Config())
    operation = schema["/api/model_catalog/v1/sources/preview"]["POST"]
    body = {"config": b"type: yaml\n", "catalogData": b"models: []\n"}

    assert map_body(HookContext(operation=operation), body) == body


def test_auth_headers_allow_multipart_content_type(root: Path, auth_headers):
    schema = schemathesis.openapi.from_path(root / "api/openapi/catalog-v1.yaml", config=schemathesis.Config())
    operation = schema["/api/model_catalog/v1/sources/preview"]["POST"]
    case = operation.Case(media_type="multipart/form-data", body={"config": "type: yaml\n"})

    request = case.as_transport_kwargs(base_url="http://localhost:8080", headers=auth_headers)

    assert request["headers"].get("Content-Type") != "application/json"
    assert request["files"]
