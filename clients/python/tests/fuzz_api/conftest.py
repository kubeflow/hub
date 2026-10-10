import contextlib
import logging
import os
import secrets
import subprocess
from collections.abc import Generator
from typing import Any

import pytest
import requests  # type: ignore[import-untyped,unused-ignore]
import schemathesis
from schemathesis import Case, Response
from schemathesis.generation.stateful.state_machine import APIStateMachine
from schemathesis.specs.openapi.schemas import OpenApiSchema

from tests.constants import DEFAULT_API_TIMEOUT, REGISTRY_URL

logger = logging.getLogger(__name__)

SINGLETON_GET_PATHS = {
    "/api/model_registry/v1/registered_model",
    "/api/model_registry/v1/model_version",
    "/api/model_registry/v1/inference_service",
    "/api/model_registry/v1/serving_environment",
    "/api/model_registry/v1/artifact",
    "/api/model_registry/v1/model_artifact",
}


_PATH_PROPERTIES: dict[str, set[str]] = {
    "/api/model_registry/v1/artifacts": {"artifactType", "customProperties", "description", "digest", "externalId", "modelFormatName", "modelFormatVersion", "modelSourceClass", "modelSourceGroup", "modelSourceId", "modelSourceKind", "modelSourceName", "name", "parameterType", "profile", "schema", "serviceAccountName", "source", "sourceType", "state", "step", "storageKey", "storagePath", "timestamp", "uri", "value"},
    "/api/model_registry/v1/artifacts/{id}": {"artifactType", "customProperties", "description", "digest", "externalId", "modelFormatName", "modelFormatVersion", "modelSourceClass", "modelSourceGroup", "modelSourceId", "modelSourceKind", "modelSourceName", "parameterType", "profile", "schema", "serviceAccountName", "source", "sourceType", "state", "step", "storageKey", "storagePath", "timestamp", "uri", "value"},
    "/api/model_registry/v1/inference_services": {"customProperties", "description", "desiredState", "externalId", "modelVersionId", "name", "registeredModelId", "runtime", "servingEnvironmentId"},
    "/api/model_registry/v1/inference_services/{inference_service_id}": {"customProperties", "description", "desiredState", "externalId", "modelVersionId", "runtime"},
    "/api/model_registry/v1/inference_services/{inference_service_id}/serves": {"customProperties", "description", "externalId", "lastKnownState", "modelVersionId", "name"},
    "/api/model_registry/v1/model_artifacts": {"artifactType", "customProperties", "description", "externalId", "modelFormatName", "modelFormatVersion", "modelSourceClass", "modelSourceGroup", "modelSourceId", "modelSourceKind", "modelSourceName", "name", "serviceAccountName", "state", "storageKey", "storagePath", "uri"},
    "/api/model_registry/v1/model_artifacts/{modelartifactId}": {"artifactType", "customProperties", "description", "externalId", "modelFormatName", "modelFormatVersion", "modelSourceClass", "modelSourceGroup", "modelSourceId", "modelSourceKind", "modelSourceName", "serviceAccountName", "state", "storageKey", "storagePath", "uri"},
    "/api/model_registry/v1/model_versions": {"author", "customProperties", "description", "externalId", "name", "registeredModelId", "state"},
    "/api/model_registry/v1/model_versions/{model_version_id}": {"author", "customProperties", "description", "externalId", "state"},
    "/api/model_registry/v1/model_versions/{model_version_id}/artifacts": {"artifactType", "customProperties", "description", "digest", "experimentId", "experimentRunId", "externalId", "modelFormatName", "modelFormatVersion", "modelSourceClass", "modelSourceGroup", "modelSourceId", "modelSourceKind", "modelSourceName", "name", "parameterType", "profile", "schema", "serviceAccountName", "source", "sourceType", "state", "step", "storageKey", "storagePath", "timestamp", "uri", "value"},
    "/api/model_registry/v1/registered_models": {"customProperties", "description", "externalId", "language", "libraryName", "license", "licenseLink", "logo", "maturity", "name", "owner", "provider", "readme", "state", "tasks"},
    "/api/model_registry/v1/registered_models/{registered_model_id}": {"customProperties", "description", "externalId", "language", "libraryName", "license", "licenseLink", "logo", "maturity", "owner", "provider", "readme", "state", "tasks"},
    "/api/model_registry/v1/registered_models/{registered_model_id}/versions": {"author", "customProperties", "description", "externalId", "name", "registeredModelId", "state"},
    "/api/model_registry/v1/serving_environments": {"customProperties", "description", "externalId", "name"},
    "/api/model_registry/v1/serving_environments/{serving_environment_id}": {"customProperties", "description", "externalId"},
    "/api/model_registry/v1/serving_environments/{serving_environment_id}/inference_services": {"customProperties", "description", "desiredState", "externalId", "modelVersionId", "name", "registeredModelId", "runtime", "servingEnvironmentId"},
}

_ALL_BODY_PROPERTIES = set().union(*_PATH_PROPERTIES.values())


_ARTIFACT_TYPES = ["model-artifact", "doc-artifact", "dataset-artifact", "metric", "parameter"]

_STRING_FIELDS = {
    "uri", "name", "description", "externalId", "artifactType", "runtime",
    "modelFormatName", "modelFormatVersion", "storageKey", "storagePath",
    "serviceAccountName", "owner", "registeredModelId", "servingEnvironmentId",
    "modelVersionId", "experimentId", "experimentRunId", "startTimeSinceEpoch",
    "endTimeSinceEpoch", "state", "status", "desiredState", "lastKnownState",
    "timestamp",
}

_NUMERIC_STRING_FIELDS = {
    "id", "registeredModelId", "servingEnvironmentId", "modelVersionId",
    "experimentId", "experimentRunId", "startTimeSinceEpoch",
    "endTimeSinceEpoch", "timestamp",
}


@schemathesis.hook
def map_body(context: Any, body: Any) -> Any:
    """Sanitize request bodies for characters that cause database/encoding errors.

    The Go server uses strict JSON decoding (DisallowUnknownFields), rejecting any
    property not defined in the struct. OpenAPI 3.0 allOf composition prevents using
    additionalProperties: false on base schemas, so we strip fuzz-generated extra
    properties here instead. We resolve allowed properties from the spec per-schema.
    """
    body = _sanitize_strings(body)
    if isinstance(body, dict):
        body = _sanitize_body_dict(context, body)
    return body


def _sanitize_body_dict(context: Any, body: dict) -> dict:
    """Filter, coerce, and fix fields in a fuzz-generated request body dict."""
    allowed = _resolve_allowed(context)
    body = {k: v for k, v in body.items() if k in allowed}
    _enforce_field_types(body)
    if "name" in body and body["name"] == "":
        body["name"] = f"fuzz-{secrets.randbelow(1000000)}"
    if "customProperties" in body:
        body["customProperties"] = _sanitize_custom_properties(body["customProperties"])
    if "artifactType" in body and body["artifactType"] not in _ARTIFACT_TYPES:
        body["artifactType"] = "doc-artifact"
    return body


def _enforce_field_types(body: dict) -> None:
    """Remove non-string values from string fields and coerce numeric string fields."""
    for field in _STRING_FIELDS:
        if field in body and not isinstance(body[field], str):
            del body[field]
    for field in _NUMERIC_STRING_FIELDS:
        if field in body and isinstance(body[field], str):
            try:
                int(body[field])
            except ValueError:
                body[field] = str(secrets.randbelow(999999999))


def _resolve_allowed(context: Any) -> set[str]:
    """Resolve allowed properties for the current operation from the spec."""
    try:
        op = context.operation
        if op is not None:
            path = op.path.value if hasattr(op.path, "value") else str(op.path)
            if path in _PATH_PROPERTIES:
                return _PATH_PROPERTIES[path]
    except Exception:
        logger.debug("Failed to resolve allowed properties from context", exc_info=True)
    return _ALL_BODY_PROPERTIES


@schemathesis.hook
def map_query(context: Any, query: dict[str, Any] | None) -> dict[str, Any] | None:
    """Sanitize fuzz-generated query parameters for cases the OpenAPI spec cannot express."""
    if query is None:
        return query

    query = _sanitize_strings(query)

    if "nextPageToken" in query:
        del query["nextPageToken"]

    if "filterQuery" in query:
        del query["filterQuery"]

    for param in ("name", "externalId"):
        if param in query and isinstance(query[param], str):
            query[param] = _strip_filter_unsafe(query[param])
            if not query[param]:
                del query[param]

    return query


@schemathesis.hook
def map_case(context: Any, case: Case) -> Case:
    """Fix parameter constraints the OpenAPI spec cannot express."""
    _fix_path_param_ids(case)
    _fix_body_id(case)
    _fix_artifact_body(case)
    if case.method and case.method.upper() != "GET":
        return case
    if case.path not in SINGLETON_GET_PATHS:
        return case
    _fix_singleton_get_query(case)
    return case


def _fix_artifact_body(case: Case) -> None:
    """Ensure artifact bodies have valid artifactType and required fields."""
    if not (case.path and _is_generic_artifact_path(case.path) and isinstance(case.body, dict)):
        return
    method = case.method.upper() if case.method else ""
    if method == "POST":
        if "artifactType" not in case.body or case.body["artifactType"] not in _ARTIFACT_TYPES:
            case.body["artifactType"] = secrets.choice(_ARTIFACT_TYPES)
        _ensure_artifact_required_fields(case.body)
    elif method == "PATCH":
        existing_type = _get_artifact_type(case)
        if existing_type:
            case.body["artifactType"] = existing_type


def _fix_singleton_get_query(case: Case) -> None:
    """Ensure singleton GET endpoints have a valid identifier query parameter."""
    if case.query is None:
        case.query = {}
    has_name = case.query.get("name")
    has_external_id = case.query.get("externalId")
    has_parent_id = case.query.get("parentResourceId")
    if not has_name and not has_external_id:
        case.query["externalId"] = "999999"
    elif has_name and not has_external_id and not has_parent_id:
        case.query["externalId"] = "999999"
        del case.query["name"]


def _fix_body_id(case: Case) -> None:
    """Sanitize or remove empty id from request body.

    Schemathesis's explicit example replay can bypass map_body, so the body
    may contain id: "" which the server parses as strconv.ParseInt("").
    """
    if not isinstance(case.body, dict):
        return
    val = case.body.get("id")
    if val is None:
        return
    if isinstance(val, str) and (not val or not _is_valid_id(val)):
        del case.body["id"]


_ID_PATH_PARAMS = {"id", "registered_model_id", "model_version_id",
                   "inference_service_id", "serving_environment_id", "servemodelId"}


def _fix_path_param_ids(case: Case) -> None:
    """Replace empty or non-numeric path parameter IDs with a valid fallback.

    Schemathesis sometimes generates empty strings for path parameters despite
    the pattern constraint. An empty ID causes 400 (bad request); a valid but
    non-existent numeric ID causes 404 which Schemathesis already accepts.
    """
    if not case.path_parameters:
        return
    for param in _ID_PATH_PARAMS:
        val = case.path_parameters.get(param)
        if val is not None and not _is_valid_id(val):
            case.path_parameters[param] = "1"


def _is_valid_id(val: Any) -> bool:
    """Check if a value is a non-empty numeric string."""
    if not isinstance(val, str) or not val:
        return False
    try:
        return int(val) > 0
    except ValueError:
        return False


_GENERIC_ARTIFACT_PATHS = {
    "/api/model_registry/v1/artifacts",
    "/api/model_registry/v1/artifacts/{id}",
    "/api/model_registry/v1/model_versions/{model_version_id}/artifacts",
}


def _is_generic_artifact_path(path: str) -> bool:
    """Check if path is a generic /artifacts endpoint that accepts any artifact type."""
    return path in _GENERIC_ARTIFACT_PATHS


def _get_artifact_type(case: Case) -> str | None:
    """GET the existing artifact to find its immutable artifactType."""
    try:
        url = f"{REGISTRY_URL}{case.formatted_path}"
        resp = requests.get(url, timeout=5)
        if resp.ok:
            return resp.json().get("artifactType")
    except Exception:
        logger.debug("Failed to fetch artifact type for %s", case.formatted_path, exc_info=True)
    return None


def _ensure_artifact_required_fields(body: dict) -> None:
    """Ensure the body has required fields and correct types for the chosen artifact type.

    The `value` field has different types per artifact: number for metric, string for
    parameter, and invalid for other types. The `step` field is int64 (JSON number)
    while `timestamp` is string — an inconsistency in the Go struct.
    """
    art_type = body.get("artifactType", "")
    if art_type == "metric":
        if "value" not in body or isinstance(body["value"], str):
            body["value"] = round(secrets.randbelow(10000) / 100.0, 2)
        body.setdefault("name", f"metric-{secrets.randbelow(100000)}")
        body["step"] = _clamp_int64(body.get("step", secrets.randbelow(1000)))
        body.setdefault("timestamp", str(secrets.randbelow(2000000000000)))
    elif art_type == "parameter":
        if "value" in body and not isinstance(body["value"], str):
            body["value"] = str(body["value"])
    else:
        body.pop("value", None)
        body.pop("step", None)
        body.pop("timestamp", None)
        body.pop("parameterType", None)


_INT64_MAX = 9223372036854775807
_INT64_MIN = -9223372036854775808


def _clamp_int64(val: Any) -> int:
    """Clamp a value to the int64 range."""
    try:
        n = int(val)
    except (TypeError, ValueError):
        return 0
    return max(_INT64_MIN, min(_INT64_MAX, n))


def _sanitize_custom_properties(props: Any) -> Any:
    """Sanitize customProperties keys and values for server compatibility.

    The server's EmbedMD converter supports Bool, Int, Double, String metadata
    types without issues. Proto and Struct types are replaced with StringValue:
    - MetadataProtoValue: no case in EmbedMD converter switch (server returns 400)
    - MetadataStructValue: server base64-decodes on write but doesn't re-encode on
      read, so any subsequent PATCH fails with "illegal base64 data" when the server
      tries to re-process the stored (decoded) value through the write converter
    """
    if not isinstance(props, dict):
        return props
    sanitized = {}
    for key, val in props.items():
        safe_key = _to_ascii(key)
        if not safe_key:
            safe_key = "prop"
        if isinstance(val, dict):
            meta_type = val.get("metadataType", "")
            if meta_type in ("MetadataStructValue", "MetadataProtoValue"):
                val = {"metadataType": "MetadataStringValue", "string_value": "placeholder"}
        sanitized[safe_key] = val
    return sanitized


def _sanitize_strings(data: Any) -> Any:
    """Recursively strip null bytes and surrogates from strings."""
    if isinstance(data, str):
        return data.replace("\x00", "").encode("utf-8", errors="ignore").decode("utf-8")
    if isinstance(data, dict):
        return {_sanitize_strings(k): _sanitize_strings(v) for k, v in data.items()}
    if isinstance(data, list):
        return [_sanitize_strings(item) for item in data]
    return data


def _to_ascii(s: str) -> str:
    """Keep only ASCII printable characters, stripping surrogates and non-ASCII."""
    return "".join(c for c in s if 0x20 <= ord(c) <= 0x7E)


def _strip_filter_unsafe(s: str) -> str:
    """Strip characters that break the server's internal filter query parser."""
    return s.replace("\\", "").replace("'", "").replace('"', "")


@pytest.fixture
def generated_schema(request: pytest.FixtureRequest, pytestconfig: pytest.Config,
                     verify_ssl: bool) -> OpenApiSchema:
    """Generate schema for the API based on the schema_file parameter"""
    schema_file = getattr(request, "param", "model-registry-v1.yaml")
    os.environ["API_HOST"] = REGISTRY_URL

    # Read and modify schemathesis.toml if verify_ssl is False
    toml_path = f"{pytestconfig.rootpath}/schemathesis.toml"
    config = schemathesis.config.SchemathesisConfig.from_path(toml_path)
    # tls-verify is by default true
    if verify_ssl is False:
        with open(toml_path) as f:
            toml_content = f.read()

        # Replace tls-verify = true with tls-verify = false
        modified_content = toml_content.replace("tls-verify = true", "tls-verify = false")

        # Write to temporary file
        import tempfile
        with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as temp_file:
            temp_file.write(modified_content)
            temp_toml_path = temp_file.name

        config = schemathesis.config.SchemathesisConfig.from_path(temp_toml_path)

        # Clean up temp file later
        os.unlink(temp_toml_path)
    print(f"Generating schema for {config}")
    schema = schemathesis.openapi.from_path(
        path=f"{pytestconfig.rootpath}/../../api/openapi/{schema_file}",
        config=config,
    )
    schema.config.output.sanitization.update(enabled=False)

    return schema


@pytest.fixture
def auth_headers(user_token: str) -> dict[str, str]:
    """Provides authorization headers for API requests."""
    return {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {user_token}",
    }


@pytest.fixture
def state_machine(generated_schema: OpenApiSchema, auth_headers: str, pytestconfig: pytest.Config,
                  verify_ssl: bool) -> APIStateMachine:
    BaseAPIWorkflow = generated_schema.as_state_machine()

    class APIWorkflow(BaseAPIWorkflow):  # type: ignore
        headers: dict[str, str]
        verify: bool

        def setup(self) -> None:
            print("Cleaning up database")
            root_path = pytestconfig.rootpath.parent.parent
            cleanup_script = root_path / "scripts" / "cleanup.sh"
            subprocess.run(  # noqa: S603
                [str(cleanup_script)],
                capture_output=True,
                check=True
            )
            self.headers = auth_headers  # type: ignore[assignment]
            self.verify = verify_ssl

        def before_call(self, case: Case) -> None:
            print(f"Checking: {case.method} {case.path}")

        def get_call_kwargs(self, case: Case) -> dict[str, Any]:
            return {"verify": self.verify, "headers": self.headers}

        def after_call(self, response: Response, case: Case) -> None:
            print(f"{case.method} {case.path} -> {response.status_code},")

    return APIWorkflow  # type: ignore[return-value,unused-ignore]


@pytest.fixture
def cleanup_artifacts(request: pytest.FixtureRequest, auth_headers: dict, verify_ssl: bool):
    """Cleanup artifacts created during the test."""
    created_ids = []

    def register(artifact_id):
        created_ids.append(artifact_id)

    yield register

    for artifact_id in created_ids:
        del_url = f"{REGISTRY_URL}/api/model_registry/v1/artifacts/{artifact_id}"
        try:
            requests.delete(del_url, headers=auth_headers, timeout=DEFAULT_API_TIMEOUT, verify=verify_ssl)
        except Exception as e:
            print(f"Failed to delete artifact {artifact_id}: {e}")


@pytest.fixture
def artifact_resource(verify_ssl: bool):
    """Create an artifact resource for the test."""

    @contextlib.contextmanager
    def _artifact_resource(auth_headers: dict, payload: dict) -> Generator[str, None, None]:
        create_endpoint = f"{REGISTRY_URL}/api/model_registry/v1/artifacts"
        resp = requests.post(create_endpoint, headers=auth_headers, json=payload, timeout=DEFAULT_API_TIMEOUT,
                             verify=verify_ssl)
        resp.raise_for_status()
        artifact_id = resp.json()["id"]
        try:
            yield artifact_id
        finally:
            del_url = f"{REGISTRY_URL}/api/model_registry/v1/artifacts/{artifact_id}"
            try:
                requests.delete(del_url, headers=auth_headers, timeout=DEFAULT_API_TIMEOUT, verify=verify_ssl)
            except Exception as e:
                print(f"Failed to delete artifact {artifact_id}: {e}")

    return _artifact_resource
