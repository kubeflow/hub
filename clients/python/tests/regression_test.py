import pytest
import requests  # type: ignore[import-untyped,unused-ignore]

from model_registry import ModelRegistry
from model_registry.types.artifacts import ModelArtifact

from .conftest import REGISTRY_HOST, REGISTRY_PORT


@pytest.mark.e2e
def test_create_tagged_version(register_model_with_version):
    """Test regression for creating tagged versions.

    Reported on: https://github.com/kubeflow/hub/issues/255
    """
    name = "test_model"
    version = "model:latest"
    register_model_with_version(name, version)


@pytest.mark.e2e
def test_get_model_without_user_token(
    setup_env_user_token,
    client: ModelRegistry,
    register_model_with_version,
):
    """Test regression for using client methods without an user_token in the init arguments.

    Reported on: https://github.com/kubeflow/hub/issues/340
    """
    assert setup_env_user_token != ""
    name = "test_model"
    version = "1.0.0"
    metadata = {"a": 1, "b": "2"}
    rm, _ = register_model_with_version(
        name,
        version,
        metadata=metadata,
    )
    assert (_rm := client.get_registered_model(name))
    assert rm.id == _rm.id


@pytest.mark.e2e
def test_get_few_registered_models(client: ModelRegistry):
    """Test regression for paging without next page token.

    Reported on: https://github.com/kubeflow/hub/issues/348
    """
    models = 9

    for name in [f"test_model{i}" for i in range(models)]:
        client.register_model(
            name,
            "s3",
            model_format_name="test_format",
            model_format_version="test_version",
            version="1.0.0",
        )

    i = 0
    for rm in client.get_registered_models():
        print(f"found {rm}")
        i += 1
        assert i < models + 1

    assert i == models


@pytest.mark.e2e
async def test_create_standalone_model_artifact(client: ModelRegistry, register_model_with_version):
    """Test regression for creating standalone model artifact.

    Reported on: https://github.com/kubeflow/hub/issues/231"""
    ma = ModelArtifact(uri="s3")
    async with client._api.get_client() as api:
        new_raw_ma = await api.create_model_artifact(ma.create())
        new_ma = ModelArtifact.from_basemodel(new_raw_ma)
        assert new_ma.id
        assert new_ma.uri == "s3"

    _, mv = register_model_with_version(
        "test_model",
        "1.0.0",
        model_format_name="x",
        model_format_version="y",
    )
    mv_ma = await client._api.upsert_model_version_artifact(new_ma, mv.id)
    assert mv_ma.id == new_ma.id


@pytest.mark.e2e
async def test_patch_model_artifacts_artifact_type(
    client: ModelRegistry,
    request_headers: dict[str, str],
    verify_ssl: bool,
    register_model_with_version,
):
    """Patching Artifacts makes the model registry server panic.

    reported with https://issues.redhat.com/browse/RHOAIENG-16932
    """
    name = "test_model"
    version = "1.0.0"
    register_model_with_version(name, version)
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id

    payload = { "modelFormatName": "foo", "artifactType": "model-artifact" }
    response = requests.patch(url=f"{REGISTRY_HOST}:{REGISTRY_PORT}/api/model_registry/v1/artifacts/{ma.id}",
                              json=payload, timeout=10,
                              headers=request_headers, verify=verify_ssl)
    assert response.status_code == 200
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id
    assert ma.model_format_name == "foo"


@pytest.mark.e2e
async def test_patch_artifact_cannot_change_artifact_type(
    client: ModelRegistry,
    request_headers: dict[str, str],
    verify_ssl: bool,
    register_model_with_version,
):
    """Changing artifact type via PATCH should return 400 error, not crash server.

    Regression test: attempting to change artifactType from model-artifact to doc-artifact
    previously caused the server to panic with a 503 error instead of returning a meaningful
    error message.
    """
    name = "test_model_artifact_type_immutable"
    version = "1.0.0"
    register_model_with_version(name, version)
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id

    # Attempt to change artifact type from model-artifact to doc-artifact
    # This should fail with a 400 Bad Request, not crash the server (503)
    payload = {"artifactType": "doc-artifact"}
    response = requests.patch(url=f"{REGISTRY_HOST}:{REGISTRY_PORT}/api/model_registry/v1/artifacts/{ma.id}",
                              json=payload, timeout=10,
                              headers=request_headers, verify=verify_ssl)

    # Server should return 400 Bad Request, not 503 (which would indicate a crash)
    assert response.status_code == 400, (
        f"Expected 400 Bad Request when changing artifact type, got {response.status_code}. "
        f"Response: {response.text}"
    )

    # Verify the error message mentions the type mismatch
    error_response = response.json()
    assert "message" in error_response
    assert "model" in error_response["message"].lower() or "doc" in error_response["message"].lower(), (
        f"Expected error message to mention artifact types, got: {error_response['message']}"
    )


@pytest.mark.e2e
async def test_as_mlops_engineer_i_would_like_to_store_a_malformed_registered_model_i_get_a_structured_error_message(
        request_headers: dict[str, str], verify_ssl: bool):
    """As a MLOps engineer if I try to store a malformed RegisteredModel I get a structured error message
    """
    payload = { "name": "test_model", "ext_id": 123 }
    response = requests.post(url=f"{REGISTRY_HOST}:{REGISTRY_PORT}/api/model_registry/v1/registered_models",
                             json=payload, timeout=10, headers=request_headers, verify=verify_ssl
                             )
    assert response.status_code == 400
    assert response.json() == {
        "code": "Bad Request",
        "message": 'json: unknown field "ext_id"',
    }


@pytest.mark.e2e
async def test_as_mlops_engineer_i_would_like_to_store_a_malformed_model_version_i_get_a_structured_error_message(
        request_headers: dict[str, str], verify_ssl: bool, register_model_with_version):
    """As a MLOps engineer if I try to store a malformed ModelVersion I get a structured error message
    """
    name = "test_model"
    version = "1.0.0"
    rm, _ = register_model_with_version(
        name,
        version,
        uri="https://acme.org/something",
    )

    payload = { "registeredModelId": rm.id }
    response = requests.post(url=f"{REGISTRY_HOST}:{REGISTRY_PORT}/api/model_registry/v1/model_versions",
                             json=payload, timeout=10, headers=request_headers, verify=verify_ssl)
    assert response.status_code == 422
    assert response.json() == {
        "code": "Bad Request",
        "message": "required field 'name' is zero value.",
    }
