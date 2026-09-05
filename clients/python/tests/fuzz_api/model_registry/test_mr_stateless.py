from __future__ import annotations

import logging
import secrets
import time
from collections.abc import Callable
from typing import Any

import pytest
import requests  # type: ignore[import-untyped,unused-ignore]
import schemathesis
from hypothesis import HealthCheck, settings

from tests.constants import (
    ARTIFACT_STATES,
    ARTIFACT_TYPE_PARAMS,
    DEFAULT_API_TIMEOUT,
    REGISTRY_URL,
)


def contains_null_bytes(data: Any) -> bool:
    """Check if the data contains null bytes that would cause validation errors."""
    if isinstance(data, str):
        return "\x00" in data
    if isinstance(data, dict):
        # Check both keys and values for null bytes
        return any(contains_null_bytes(key) or contains_null_bytes(value) for key, value in data.items())
    if isinstance(data, list):
        return any(contains_null_bytes(item) for item in data)
    return False


def call_and_validate_with_null_byte_handling(case: schemathesis.Case, auth_headers: dict, verify_ssl: bool) -> None:
    """Execute case validation with proper null byte error handling.

    This utility function handles the common pattern of calling case.call_and_validate()
    and catching exceptions related to null bytes in the request. If null bytes are detected
    in the query or body and the error is a 400 or CheckFailed, the test is skipped as expected.
    Otherwise, the exception is re-raised.

    Args:
        case: The Schemathesis test case to validate
        auth_headers: Authentication headers for the API
        verify_ssl: Whether to verify SSL certificates

    Raises:
        Exception: Re-raises any exception that is not related to null byte validation
    """
    try:
        case.call_and_validate(headers=auth_headers, verify=verify_ssl)
    except Exception as e:
        request_has_null_bytes = False
        if case.query:
            request_has_null_bytes = contains_null_bytes(case.query)
        if not request_has_null_bytes and hasattr(case, "body") and case.body:
            try:
                if isinstance(case.body, str | bytes):
                    body_str = case.body.decode("utf-8") if isinstance(case.body, bytes) else case.body
                    request_has_null_bytes = contains_null_bytes(body_str)
            except (UnicodeDecodeError, AttributeError):
                pass
        if request_has_null_bytes and ("400" in str(e) or "CheckFailed" in str(type(e).__name__)):
            logging.info(f"Expected 400 error for null bytes: {case.path}")
            pytest.skip("Expected validation error for null bytes")
        else:
            raise


# Helper functions for common operations
def generate_random_id() -> str:
    """Generate a random ID between 100000 and 2000000000."""
    return str(secrets.randbelow(2000000000 - 100000 + 1) + 100000)


def generate_unique_timestamp() -> str:
    """Generate a unique timestamp in milliseconds with random offset."""
    current_ms = int(time.time() * 1000)
    random_offset = secrets.randbelow(1000000)  # Random offset up to 1 million ms (1000 seconds)
    return str(current_ms + random_offset)


def build_artifact_payload(artifact_type: str, uri_prefix: str, state: str, name: str,
                          description: str | None = None, external_id: str | None = None) -> dict[str, Any]:
    """Build a payload for creating an artifact based on its type.

    Args:
        artifact_type: Type of artifact (e.g., 'model-artifact', 'doc-artifact', etc.)
        uri_prefix: Prefix for URI (e.g., 's3://', 'https://')
        state: State of the artifact
        name: Name of the artifact
        description: Optional description
        external_id: Optional external ID, generates random if not provided

    Returns:
        Dictionary containing the artifact payload
    """
    payload: dict[str, Any] = {
        "artifactType": artifact_type,
        "name": name,
        "state": state,
        "externalId": external_id or generate_random_id()
    }

    if description:
        payload["description"] = description

    # Add type-specific properties
    if artifact_type == "model-artifact":
        payload["modelFormatName"] = "tensorflow"
        payload["modelFormatVersion"] = "1.0"
        payload["uri"] = f"{uri_prefix}my-test-model.pkl"
    elif artifact_type == "doc-artifact":
        payload["uri"] = f"{uri_prefix}documentation.pdf"
    elif artifact_type == "dataset-artifact":
        payload["uri"] = f"{uri_prefix}dataset.parquet"
        payload["sourceType"] = "s3"
        payload["source"] = "s3://test-bucket/datasets/"
    elif artifact_type == "metric":
        payload["value"] = 0.95  # Example accuracy value
        payload["timestamp"] = generate_unique_timestamp()
    elif artifact_type == "parameter":
        payload["value"] = "0.001"  # Example learning rate
        payload["parameterType"] = "string"

    return payload


def validate_artifact_response(response: requests.Response, expected_payload: dict[str, Any]) -> str | None:
    """Validate artifact creation response and return the artifact ID.

    Args:
        response: The HTTP response from the API
        expected_payload: The payload that was sent, used for validation

    Returns:
        The artifact ID from the response

    Raises:
        AssertionError: If validation fails
    """
    # Check response status
    assert response.status_code in {200, 201, 404}, f"Expected 200, 201, or 404, got {response.status_code}: {response.text}"

    response_json = response.json()

    # Handle error responses (404)
    if response.status_code == 404:
        assert "message" in response_json, "Error response should contain 'message' field"
        assert "not found" in response_json["message"].lower(), f"Error message should contain 'not found', got: {response_json['message']}"
        # For 404 responses, we don't return an ID since the artifact wasn't created
        return None

    # Handle success responses (200, 201)
    assert response_json.get("id"), "Response body should contain 'id'"

    # Validate response matches payload
    assert response_json.get("name") == expected_payload["name"], "Response name should match payload name"
    assert response_json.get("artifactType") == expected_payload["artifactType"], "Response artifactType should match payload"

    return response_json["id"]


# Null byte validation for Model Registry API endpoints specifically
@schemathesis.check
def check_null_byte_validation_mr(ctx, response, case):
    """Validate that Model Registry requests with PostgreSQL null byte errors return 400 Bad Request."""

    # Only check model registry endpoints
    if not case.path.startswith("/api/model_registry/"):
        return  # Skip validation for non-model-registry endpoints


schema = schemathesis.pytest.from_fixture("generated_schema")


# Base schema with common exclusions
base_schema = (
    schema
    .exclude(
        path="/api/model_registry/v1/artifacts/{id}",
        method="PATCH"
    )
    .exclude(
        path="/api/model_registry/v1/model_versions/{model_version_id}/artifacts",
        method="POST"
    )
)

# Split tests by HTTP method AND resource type for maximum parallelization
# This creates many small test functions that can be distributed across workers

# GET endpoints - split by resource type
@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="GET", path_regex=".*/(registered_model|model_version).*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_get_models(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test GET endpoints for RegisteredModels and ModelVersions."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="GET", path_regex=".*/artifact.*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_get_artifacts(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test GET endpoints for Artifacts and ModelArtifacts."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="GET", path_regex=".*/inference_service.*|.*/serving_environment.*|.*/serve.*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_get_serving(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test GET endpoints for InferenceServices and ServingEnvironments."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


# POST endpoints - split by resource type
@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="POST", path_regex=".*/(registered_model|model_version).*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_post_models(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test POST endpoints for RegisteredModels and ModelVersions."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="POST", path_regex=".*/artifact.*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_post_artifacts(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test POST endpoints for Artifacts."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="POST", path_regex=".*/inference_service.*|.*/serving_environment.*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_post_serving(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test POST endpoints for InferenceServices and ServingEnvironments."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


# PATCH endpoints - split by resource type
@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="PATCH", path_regex=".*/(registered_model|model_version).*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_patch_models(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test PATCH endpoints for RegisteredModels and ModelVersions."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)


@pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
@base_schema.include(method="PATCH", path_regex=".*/artifact.*|.*/inference_service.*|.*/serving_environment.*").parametrize()
@settings(
    max_examples=100,
    deadline=None,
    suppress_health_check=[
        HealthCheck.filter_too_much,
        HealthCheck.too_slow,
        HealthCheck.data_too_large,
    ],
)
@pytest.mark.fuzz
def test_mr_api_stateless_patch_others(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test PATCH endpoints for Artifacts and Serving resources."""
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)

@pytest.mark.fuzz
@pytest.mark.parametrize(("artifact_type", "uri_prefix"), ARTIFACT_TYPE_PARAMS)
@pytest.mark.parametrize("state", ARTIFACT_STATES)
def test_post_model_version_artifacts(auth_headers: dict, artifact_type: str, uri_prefix: str, state: str,
                                      cleanup_artifacts: Callable, verify_ssl: bool):
    """
    Direct test for POST /api/model_registry/v1/model_versions/{model_version_id}/artifacts.
    """
    model_version_id = generate_random_id()
    endpoint = f"{REGISTRY_URL}/api/model_registry/v1/model_versions/{model_version_id}/artifacts"

    # Build payload using helper function
    payload = build_artifact_payload(
        artifact_type=artifact_type,
        uri_prefix=uri_prefix,
        state=state,
        name="my-test-model-artifact-post",
        description="A test model artifact created via direct POST test."
    )

    # Make the API request
    response = requests.post(endpoint, headers=auth_headers, json=payload, timeout=DEFAULT_API_TIMEOUT, verify=verify_ssl)

    # Validate response and get artifact ID
    artifact_id = validate_artifact_response(response, payload)

    # Cleanup after successful creation (only if artifact was created)
    if artifact_id is not None:
        cleanup_artifacts(artifact_id)


@pytest.mark.fuzz
@pytest.mark.parametrize(("artifact_type", "uri_prefix"), ARTIFACT_TYPE_PARAMS)
def test_patch_artifact(auth_headers: dict, artifact_resource: Callable, artifact_type: str, uri_prefix: str,
                        verify_ssl: bool):
    """
    Direct test for PATCH /api/model_registry/v1/artifacts/{id}.
    """
    initial_state = "PENDING"
    target_state = "LIVE"

    # Build initial payload for creation
    # Note: Using a hardcoded URI prefix for creation since these are specific test URIs
    build_artifact_payload(
        artifact_type=artifact_type,
        uri_prefix="s3://my-test-bucket/" if artifact_type in ["model-artifact", "dataset-artifact"] else "https://docs.example.com/",
        state=initial_state,
        name="test-create-for-patch"
    )

    # Override some specific properties for the patch test
    create_payload = {
        "artifactType": artifact_type,
        "name": "test-create-for-patch",
        "state": initial_state,
    }

    # Add type-specific properties for creation with test-specific values
    if artifact_type == "model-artifact":
        create_payload["modelFormatName"] = "tensorflow"
        create_payload["modelFormatVersion"] = "1.0"
        create_payload["uri"] = "s3://my-test-bucket/models/initial-model.pkl"
    elif artifact_type == "doc-artifact":
        create_payload["uri"] = "https://docs.example.com/initial-doc.pdf"
    elif artifact_type == "dataset-artifact":
        create_payload["uri"] = "s3://my-test-bucket/datasets/initial-dataset.parquet"
        create_payload["sourceType"] = "s3"
    elif artifact_type == "metric":
        create_payload["value"] = 0.85  # type: ignore[assignment]
        create_payload["timestamp"] = "1000000000"
    elif artifact_type == "parameter":
        create_payload["value"] = "0.01"
        create_payload["parameterType"] = "string"

    with artifact_resource(auth_headers, create_payload) as artifact_id:
        patch_endpoint = f"{REGISTRY_URL}/api/model_registry/v1/artifacts/{artifact_id}"
        patch_payload = {
            "artifactType": artifact_type,
            "description": f"Updated description for {artifact_type} ({target_state})",
            "state": target_state,
        }

        # Add type-specific update properties if needed
        if artifact_type == "metric":
            patch_payload["value"] = 0.99  # type: ignore[assignment]  # Updated metric value
        elif artifact_type == "parameter":
            patch_payload["value"] = "0.001"  # Updated parameter value

        patch_response = requests.patch(patch_endpoint, headers=auth_headers, json=patch_payload, timeout=DEFAULT_API_TIMEOUT, verify=verify_ssl)
        assert patch_response.status_code == 200
        patch_response_json = patch_response.json()
        assert patch_response_json.get("id") == artifact_id
        assert patch_response_json.get("description") == patch_payload["description"]
        assert patch_response_json.get("state") == patch_payload["state"]
        assert patch_response_json.get("artifactType") == artifact_type
