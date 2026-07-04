import logging
import os
import tempfile
from itertools import islice
from typing import Any
from unittest.mock import MagicMock

import pytest
import requests  # type: ignore[import-untyped,unused-ignore]

from model_registry import ModelRegistry, utils
from model_registry.exceptions import StoreError
from model_registry.types import ModelArtifact
from model_registry.types.artifacts import DocArtifact


def test_secure_client(monkeypatch):
    monkeypatch.delenv("CERT", raising=False)
    monkeypatch.delenv("KF_PIPELINES_SA_TOKEN_PATH", raising=False)
    with pytest.raises(StoreError) as e:
        ModelRegistry("anything", author="test_author")

    assert "user token" in str(e.value).lower()


def test_http_server_address_defaults_to_port_8080(mock_get_registered_models):
    client = ModelRegistry(server_address="http://localhost", author="test_author")

    assert client._api.config.host == "http://localhost:8080"
    assert client._api.config.verify_ssl is False


def test_https_server_address_defaults_to_port_443(mock_get_registered_models):
    test_token = "test-token"  # noqa: S105
    client = ModelRegistry(
        server_address="https://localhost",
        author="test_author",
        user_token=test_token,
    )

    assert client._api.config.host == "https://localhost:443"


def test_hint_server_address_port_http_url_uses_embedded_port(mock_get_registered_models, caplog):
    with caplog.at_level(logging.WARNING):
        client = ModelRegistry(server_address="http://example.com:8080", author="test")

    assert client._api.config.host == "http://example.com:8080"
    assert client._api.config.verify_ssl is False
    assert len(caplog.records) == 0


def test_hint_server_address_port_https_url_uses_embedded_port(mock_get_registered_models, caplog):
    test_token = "test-token"  # noqa: S105
    with caplog.at_level(logging.WARNING):
        client = ModelRegistry(
            server_address="https://example.com:8080",
            author="test",
            user_token=test_token,
        )

    assert client._api.config.host == "https://example.com:8080"
    assert client._api.config.access_token == test_token
    assert len(caplog.records) == 1
    assert "Server address protocol is https://, but port is not 443 or ending with 443" in caplog.records[0].message


@pytest.mark.e2e
async def test_register_new(
    client: ModelRegistry,
    register_model_with_version,
):
    """As a MLOps engineer I would like to store Model name"""
    name = "test_model"
    version = "1.0.0"
    rm, _ = register_model_with_version(
        name,
        version,
        uri="https://acme.org/something",
    )
    assert rm.name == name  # check the Model name

    mr_api = client._api
    mv = await mr_api.get_model_version_by_params(rm.id, version)
    assert mv
    assert mv.id
    assert mv.name == version
    assert mv.registered_model_id == rm.id

    ma = await mr_api.get_model_artifact_by_params(name, mv.id)
    assert ma
    assert ma.uri == "https://acme.org/something"


@pytest.mark.e2e
async def test_register_new_using_s3_uri_builder(
    client: ModelRegistry,
    register_model_with_version,
):
    name = "test_model"
    version = "1.0.0"
    uri = utils.s3_uri_from("storage/path", "my-bucket", endpoint="my-endpoint", region="my-region")
    rm, _ = register_model_with_version(
        name,
        version,
        uri=uri,
    )
    assert rm.id is not None

    mr_api = client._api
    mv = await mr_api.get_model_version_by_params(rm.id, version)
    assert mv
    assert mv.id
    ma = await mr_api.get_model_artifact_by_params(name, mv.id)
    assert ma
    assert ma.uri == uri


@pytest.mark.e2e
def test_page_through_zero_models(client: ModelRegistry):
    """Test that we can page through zero models (i.e. a Model Registry just created)"""
    # leave with no models in the MR server
    for _ in client.get_registered_models():
        pytest.fail("should never enter here, there are no models in the MR server")


@pytest.mark.e2e
def test_page_through_one_models(client: ModelRegistry):
    """Complementary of test_page_through_zero_models, check a simple pagination with 1 model on the Model Registry server"""
    client.register_model("my-model", "some://uri", version="v1", model_format_name="vLLM", model_format_version="v1")
    for registered_model in client.get_registered_models():
        assert registered_model.name == "my-model"  # there is only 1 specific model in the MR server.


@pytest.mark.e2e
def test_register_existing_version(client: ModelRegistry):
    params = {
        "name": "test_model",
        "uri": "s3",
        "model_format_name": "test_format",
        "model_format_version": "test_version",
        "version": "1.0.0",
    }
    client.register_model(**params, metadata=None)

    with pytest.raises(StoreError):
        client.register_model(**params, metadata=None)


@pytest.mark.e2e
def test_register_version_long_name(client: ModelRegistry):
    """ModelVersion.name can generally account for up to 250chars, assuming up to 10K RegisteredModels.
    This is because ModelVersion being a MLMD.Context owned entity, its name is prefixed with `RegisteredModel.id:` in the backend
    to preserve uniqueness for MLMD schema constraints
    """
    lorem = "Lorem ipsum dolor sit amet, consectetuer adipiscing elit. Aenean commodo ligula eget dolor. Aenean massa. Cum sociis natoque penatibus et magnis dis parturient montes, nascetur ridiculus mus. Donec quam felis, ultricies nec, pellentesque eu, pretium."
    assert len(lorem) == 250

    client.register_model(
        name="test_model",
        uri="https://acme.org/something",
        model_format_name="test_format_name",
        model_format_version="test_format_version",
        version=lorem,
    )
    ma = client.get_model_artifact(name="test_model", version=lorem)
    assert ma.uri == "https://acme.org/something"  # type: ignore[union-attr]
    assert ma.model_format_name == "test_format_name"  # type: ignore[union-attr]

    with pytest.raises(Exception):  # noqa the focus of this test is the failure case, not to fix on the exception being raised
        client.register_model(
            name="test_model",
            uri="https://acme.org/something",
            model_format_name="test_format_name",
            model_format_version="test_format_version",
            version=lorem + "12345",
        )  # version of 255 chars is above limit because does not account for owned entity prefix, ie `1:...`


@pytest.mark.e2e
async def test_update_models(client: ModelRegistry, register_model_with_version):
    name = "test_model"
    version = "1.0.0"
    rm, mv = register_model_with_version(name, version)
    ma = client.get_model_artifact(name, version)
    assert ma

    new_description = "updated description"
    rm.description = new_description
    mv.description = new_description
    ma.description = new_description
    assert client.update(rm).description == new_description
    assert client.update(mv).description == new_description
    assert client.update(ma).description == new_description


@pytest.mark.e2e
async def test_update_logical_model_with_labels(
    client: ModelRegistry,
    register_model_with_version,
):
    """As a MLOps engineer I would like to store some labels

    A custom property of type string, with empty string value, shall be considered a Label; this is also semantically compatible for properties having empty string values in general.
    """
    name = "test_model"
    version = "1.0.0"
    rm, mv = register_model_with_version(name, version)
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id

    rm_labels = {
        "my-label1": "",
        "my-label2": "",
    }
    rm.custom_properties = rm_labels
    client.update(rm)

    mv_labels = {
        "my-label3": "",
        "my-label4": "",
    }
    mv.custom_properties = mv_labels
    client.update(mv)

    ma_labels = {
        "my-label5": "",
        "my-label6": "",
    }
    ma.custom_properties = ma_labels
    client.update(ma)

    rm = client.get_registered_model(name)
    assert rm
    assert rm.custom_properties == rm_labels
    mv = client.get_model_version(name, version)
    assert mv
    assert mv.custom_properties == mv_labels
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.custom_properties == ma_labels


@pytest.mark.e2e
async def test_patch_model_artifacts_artifact_type(
    client: ModelRegistry,
    request_headers: dict[str, str],
    verify_ssl: bool,
    register_model_with_version,
):
    """Patching ModelArtifact requires `artifactType` value which was previously not required

    reported with https://issues.redhat.com/browse/RHOAIENG-15326
    """
    name = "test_model"
    version = "1.0.0"
    register_model_with_version(name, version)
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id

    payload = {"modelFormatName": "foo"}
    from .conftest import REGISTRY_HOST, REGISTRY_PORT

    response = requests.patch(
        url=f"{REGISTRY_HOST}:{REGISTRY_PORT}/api/model_registry/v1alpha3/model_artifacts/{ma.id}",
        json=payload,
        timeout=10,
        headers=request_headers,
        verify=verify_ssl,
    )
    assert response.status_code == 200
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id
    assert ma.model_format_name == "foo"


@pytest.mark.e2e
async def test_update_preserves_model_info(
    client: ModelRegistry,
    register_model_with_version,
):
    name = "test_model"
    version = "1.0.0"
    uri = "s3"
    model_fmt_name = "test_format"
    model_fmt_version = "test_version"
    rm, _ = register_model_with_version(
        name,
        version,
        uri=uri,
        model_format_name=model_fmt_name,
        model_format_version=model_fmt_version,
    )

    mr_api = client._api
    mv = await mr_api.get_model_version_by_params(rm.id, version)
    assert mv
    assert mv.id
    ma = await mr_api.get_model_artifact_by_params(name, mv.id)
    assert ma

    new_description = "updated description"
    ma = ModelArtifact(id=ma.id, uri=uri, description=new_description)

    updated_ma = client.update(ma)
    assert updated_ma.description == new_description
    assert updated_ma.uri == uri
    assert updated_ma.id == ma.id
    assert updated_ma.model_format_name == model_fmt_name
    assert updated_ma.model_format_version == model_fmt_version


@pytest.mark.e2e
async def test_update_existing_model_artifact(
    client: ModelRegistry,
    register_model_with_version,
):
    """Updating uri (or other properties) by re-using and call to update

    reported via slack
    """
    name = "test_model"
    version = "1.0.0"
    register_model_with_version(name, version)
    ma = client.get_model_artifact(name, version)
    assert ma
    assert ma.id

    something_else = "https://something.else/model.onnx"
    ma.uri = something_else
    response = client.update(ma)
    assert response
    assert response.uri == something_else

    ma = client.get_model_artifact(name, version)
    assert ma.uri == something_else  # type: ignore[union-attr]


@pytest.mark.e2e
async def test_get(
    client: ModelRegistry,
    register_model_with_version,
):
    name = "test_model"
    version = "1.0.0"
    metadata = {"a": 1, "b": "2"}

    rm, _ = register_model_with_version(
        name,
        version,
        metadata=metadata,
    )

    assert rm.id
    assert (_rm := client.get_registered_model(name))
    assert rm.id == _rm.id

    mr_api = client._api
    assert (mv := await mr_api.get_model_version_by_params(rm.id, version))
    assert mv.id
    assert (ma := await mr_api.get_model_artifact_by_params(name, mv.id))

    assert (_mv := client.get_model_version(name, version))
    assert mv.id == _mv.id
    assert mv.custom_properties == metadata
    assert (_ma := client.get_model_artifact(name, version))
    assert ma.id == _ma.id


@pytest.mark.e2e
def test_get_registered_models(client: ModelRegistry):
    models = 21

    for name in [f"test_model{i}" for i in range(models)]:
        client.register_model(
            name,
            "s3",
            model_format_name="test_format",
            model_format_version="test_version",
            version="1.0.0",
        )

    rm_iter = client.get_registered_models().page_size(10)
    i = 0
    prev_tok = None
    changes = 0
    with pytest.raises(StopIteration):  # noqa: PT012
        while i < 50 and next(rm_iter):
            if rm_iter.options.next_page_token != prev_tok:
                print(f"Token changed from {prev_tok} to {rm_iter.options.next_page_token} at {i}")
                prev_tok = rm_iter.options.next_page_token
                changes += 1
            i += 1

    assert changes == 3
    assert i == models


@pytest.mark.e2e
def test_get_registered_models_order_by(client: ModelRegistry):
    models = 5

    rms = []
    for name in [f"test_model{i}" for i in range(models)]:
        rms.append(
            client.register_model(
                name,
                "s3",
                model_format_name="test_format",
                model_format_version="test_version",
                version="1.0.0",
            )
        )

    # id ordering should match creation order
    i = 0
    for rm, by_id in zip(
        rms,
        client.get_registered_models().order_by_id(), strict=False,
    ):
        assert rm.id == by_id.id
        i += 1

    assert i == models

    # and obviously, creation ordering should match creation ordering
    i = 0
    for rm, by_creation in zip(
        rms,
        client.get_registered_models().order_by_creation_time(), strict=False,
    ):
        assert rm.id == by_creation.id
        i += 1

    assert i == models

    # update order should match creation ordering by default
    i = 0
    for rm, by_update in zip(
        rms,
        client.get_registered_models().order_by_update_time(), strict=False,
    ):
        assert rm.id == by_update.id
        i += 1

    assert i == models

    # now update the models in reverse order
    for rm in reversed(rms):
        rm.description = "updated"
        client.update(rm)

    # and they should match in reverse
    i = 0
    for rm, by_update in zip(
        reversed(rms),
        client.get_registered_models().order_by_update_time(), strict=False,
    ):
        assert rm.id == by_update.id
        i += 1

    assert i == models

    # or if descending is explicitly set
    i = 0
    for rm, by_update in zip(
        rms,
        client.get_registered_models().order_by_update_time().descending(), strict=False,
    ):
        assert rm.id == by_update.id
        i += 1

    assert i == models


@pytest.mark.e2e
def test_get_registered_models_and_reset(client: ModelRegistry):
    model_count = 6
    page = model_count // 2

    for name in [f"test_model{i}" for i in range(model_count)]:
        client.register_model(
            name,
            "s3",
            model_format_name="test_format",
            model_format_version="test_version",
            version="1.0.0",
        )

    rm_iter = client.get_registered_models().page_size(model_count - 1)
    models = []
    for rm in islice(rm_iter, page):
        models.append(rm)
    assert len(models) == page
    rm_iter.restart()
    complete = list(rm_iter)
    assert len(complete) == model_count
    assert complete[:page] == models


@pytest.mark.e2e
def test_get_model_versions(client: ModelRegistry):
    name = "test_model"
    models = 21

    for v in [f"1.0.{i}" for i in range(models)]:
        client.register_model(
            name,
            "s3",
            model_format_name="test_format",
            model_format_version="test_version",
            version=v,
        )

    mv_iter = client.get_model_versions(name).page_size(10)
    i = 0
    prev_tok = None
    changes = 0
    with pytest.raises(StopIteration):  # noqa: PT012
        while i < 50 and next(mv_iter):
            if mv_iter.options.next_page_token != prev_tok:
                print(f"Token changed from {prev_tok} to {mv_iter.options.next_page_token} at {i}")
                prev_tok = mv_iter.options.next_page_token
                changes += 1
            i += 1

    assert changes == 3
    assert i == models


@pytest.mark.e2e
def test_get_model_versions_order_by(client: ModelRegistry):
    name = "test_model"
    models = 5
    mvs = []
    for v in [f"1.0.{i}" for i in range(models)]:
        client.register_model(
            name,
            "s3",
            model_format_name="test_format",
            model_format_version="test_version",
            version=v,
        )
        mvs.append(client.get_model_version(name, v))

    i = 0
    for mv, by_id in zip(
        mvs,
        client.get_model_versions(name).order_by_id(), strict=False,
    ):
        assert mv.id == by_id.id  # type: ignore[union-attr]
        i += 1

    assert i == models

    i = 0
    for mv, by_creation in zip(
        mvs,
        client.get_model_versions(name).order_by_creation_time(), strict=False,
    ):
        assert mv.id == by_creation.id  # type: ignore[union-attr]
        i += 1

    assert i == models

    i = 0
    for mv, by_update in zip(
        mvs,
        client.get_model_versions(name).order_by_update_time(), strict=False,
    ):
        assert mv.id == by_update.id  # type: ignore[union-attr]
        i += 1

    assert i == models

    for mv in reversed(mvs):
        mv.description = "updated"  # type: ignore[union-attr]
        client.update(mv)  # type: ignore[type-var]

    i = 0
    for mv, by_update in zip(
        reversed(mvs),
        client.get_model_versions(name).order_by_update_time(), strict=False,
    ):
        assert mv.id == by_update.id  # type: ignore[union-attr]
        i += 1

    assert i == models

    i = 0
    for mv, by_update in zip(
        mvs,
        client.get_model_versions(name).order_by_update_time().descending(), strict=False,
    ):
        assert mv.id == by_update.id  # type: ignore[union-attr]
        i += 1

    assert i == models


@pytest.mark.e2e
def test_get_model_versions_and_reset(client: ModelRegistry):
    name = "test_model"

    model_count = 6
    page = model_count // 2

    for v in [f"1.0.{i}" for i in range(model_count)]:
        client.register_model(
            name,
            "s3",
            model_format_name="test_format",
            model_format_version="test_version",
            version=v,
        )

    mv_iter = client.get_model_versions(name).page_size(model_count - 1)
    models = []
    for rm in islice(mv_iter, page):
        models.append(rm)
    assert len(models) == page
    mv_iter.restart()
    complete = list(mv_iter)
    assert len(complete) == model_count
    assert complete[:page] == models


@pytest.mark.e2e
def test_hf_import(client: ModelRegistry):
    pytest.importorskip("huggingface_hub")
    name = "openai-community/gpt2"
    version = "1.2.3"
    author = "test author"

    assert client.register_hf_model(
        name,
        "onnx/decoder_model.onnx",
        author=author,
        version=version,
        model_format_name="test format",
        model_format_version="test version",
    )
    assert (mv := client.get_model_version(name, version))
    assert mv.author == author
    assert mv.custom_properties
    assert mv.custom_properties["model_author"] == author
    assert mv.custom_properties["model_origin"] == "huggingface_hub"
    assert (
        mv.custom_properties["source_uri"]
        == "https://huggingface.co/openai-community/gpt2/resolve/main/onnx/decoder_model.onnx"
    )
    assert mv.custom_properties["repo"] == name
    assert client.get_model_artifact(name, version)


@pytest.mark.e2e
def test_hf_import_default_env(client: ModelRegistry):
    """Test setting environment variables, hence triggering defaults, does _not_ interfere with HF metadata"""
    pytest.importorskip("huggingface_hub")
    name = "openai-community/gpt2"
    version = "1.2.3"
    author = "test author"
    env_values = {
        "AWS_S3_ENDPOINT": "value1",
        "AWS_S3_BUCKET": "value2",
        "AWS_DEFAULT_REGION": "value3",
    }
    for k, v in env_values.items():
        os.environ[k] = v

    assert client.register_hf_model(
        name,
        "onnx/decoder_model.onnx",
        author=author,
        version=version,
        model_format_name="test format",
        model_format_version="test version",
    )
    assert (mv := client.get_model_version(name, version))
    assert mv.custom_properties
    assert mv.custom_properties["model_author"] == author
    assert mv.custom_properties["model_origin"] == "huggingface_hub"
    assert (
        mv.custom_properties["source_uri"]
        == "https://huggingface.co/openai-community/gpt2/resolve/main/onnx/decoder_model.onnx"
    )
    assert mv.custom_properties["repo"] == name
    assert client.get_model_artifact(name, version)

    for k in env_values:
        os.environ.pop(k)


@pytest.mark.e2e
def test_singular_store_in_s3(get_model_file, patch_s3_env, client: ModelRegistry):
    pytest.importorskip("boto3")

    # So we have an import locally, since we are directly using it
    import boto3

    assert get_model_file is not None

    s3_endpoint = os.getenv("AWS_S3_ENDPOINT")
    access_id = os.getenv("AWS_ACCESS_KEY_ID")
    secret_key = os.getenv("AWS_SECRET_ACCESS_KEY")
    default_region = os.getenv("AWS_DEFAULT_REGION")
    bucket = os.getenv("AWS_S3_BUCKET")

    # Make sure MonkeyPatch env vars are set
    assert s3_endpoint is not None
    assert access_id is not None
    assert secret_key is not None
    assert default_region is not None
    assert bucket is not None

    model_name = get_model_file.split("/")[-1]
    prefix = "models"
    uri = client.save_to_s3(path=get_model_file, bucket_name=bucket, s3_prefix=prefix)

    s3 = boto3.client(
        "s3",
        endpoint_url=s3_endpoint,
        aws_access_key_id=access_id,
        aws_secret_access_key=secret_key,
        region_name=default_region,
    )

    # Manually check that the object is indeed here
    objects = s3.list_objects_v2(Bucket="default")["Contents"]
    objects_by_name = [obj["Key"] for obj in objects]
    model_name_pfx = os.path.join(prefix, model_name)
    s3_link = utils.s3_uri_from(bucket=bucket, path=prefix, endpoint=s3_endpoint, region=default_region)

    assert isinstance(uri, str)
    assert uri == s3_link
    assert model_name_pfx in objects_by_name

    # Test file not exists
    with pytest.raises(ValueError, match="Please ensure path is correct.") as e:
        client.save_to_s3(path=f"{get_model_file}x", s3_prefix=prefix, bucket_name=bucket)
    assert "please ensure path is correct" in str(e.value).lower()


@pytest.mark.e2e
def test_recursive_store_in_s3(get_temp_dir_with_models, patch_s3_env, client: ModelRegistry):
    pytest.importorskip("boto3")

    # So we have an import locally, since we are directly using it
    import boto3

    model_dir, files = get_temp_dir_with_models
    assert model_dir is not None
    assert isinstance(files, list)
    assert len(files) == 3

    s3_endpoint = os.getenv("AWS_S3_ENDPOINT")
    access_id = os.getenv("AWS_ACCESS_KEY_ID")
    secret_key = os.getenv("AWS_SECRET_ACCESS_KEY")
    default_region = os.getenv("AWS_DEFAULT_REGION")
    bucket = os.getenv("AWS_S3_BUCKET")

    # Make sure MonkeyPatch env vars are set
    assert s3_endpoint is not None
    assert access_id is not None
    assert secret_key is not None
    assert default_region is not None
    assert bucket is not None

    prefix = "models2"
    uri = client.save_to_s3(path=model_dir, bucket_name=bucket, s3_prefix=prefix)

    s3 = boto3.client(
        "s3",
        endpoint_url=s3_endpoint,
        aws_access_key_id=access_id,
        aws_secret_access_key=secret_key,
        region_name=default_region,
    )

    # Manually check that the object is indeed here
    objects = s3.list_objects_v2(Bucket="default")["Contents"]
    objects_by_name = [obj["Key"] for obj in objects]
    formatted_paths = [os.path.join(prefix, os.path.basename(path)) for path in files]
    s3_uri = utils.s3_uri_from(bucket=bucket, path=prefix, endpoint=s3_endpoint, region=default_region)

    assert isinstance(uri, str)
    assert uri == s3_uri
    for path in formatted_paths:
        assert path in objects_by_name

    # Test incorrect folder
    with pytest.raises(ValueError, match="Please ensure path is correct.") as e:
        client.save_to_s3(path=f"{model_dir}x", s3_prefix=prefix, bucket_name=bucket)
    assert "please ensure path is correct" in str(e.value).lower()


@pytest.mark.e2e
def test_nested_recursive_store_in_s3(get_temp_dir_with_nested_models, patch_s3_env, client: ModelRegistry):
    pytest.importorskip("boto3")

    # So we have an import locally, since we are directly using it
    import boto3

    model_dir, files = get_temp_dir_with_nested_models
    assert model_dir is not None
    assert isinstance(files, list)
    assert len(files) == 3

    s3_endpoint = os.getenv("AWS_S3_ENDPOINT")
    access_id = os.getenv("AWS_ACCESS_KEY_ID")
    secret_key = os.getenv("AWS_SECRET_ACCESS_KEY")
    default_region = os.getenv("AWS_DEFAULT_REGION")
    bucket = os.getenv("AWS_S3_BUCKET")

    # Make sure MonkeyPatch env vars are set
    assert s3_endpoint is not None
    assert access_id is not None
    assert secret_key is not None
    assert default_region is not None
    assert bucket is not None

    prefix = "models3"
    uri = client.save_to_s3(path=model_dir, s3_prefix=prefix, bucket_name=bucket)

    s3 = boto3.client(
        "s3",
        endpoint_url=s3_endpoint,
        aws_access_key_id=access_id,
        aws_secret_access_key=secret_key,
        region_name=default_region,
    )

    # Manually check that the object is indeed here
    objects = s3.list_objects_v2(Bucket="default")["Contents"]
    objects_by_name = [obj["Key"] for obj in objects]
    s3_uri = utils.s3_uri_from(bucket=bucket, path=prefix, endpoint=s3_endpoint, region=default_region)
    # this is creating a list of all the file names + their immediate parent folder only
    formatted_paths = [
        os.path.join(prefix, os.path.basename(os.path.dirname(path)), os.path.basename(path)) for path in files
    ]

    assert isinstance(uri, str)
    assert uri == s3_uri
    for path in formatted_paths:
        assert path in objects_by_name

    # Test incorrect folder
    with pytest.raises(ValueError, match="Please ensure path is correct.") as e:
        client.save_to_s3(path=f"{model_dir}x", s3_prefix=prefix, bucket_name=bucket)
    assert "please ensure path is correct" in str(e.value).lower()


@pytest.mark.e2e
def test_custom_async_runner_with_ray(
    client_attrs: dict[str, Any],
    client: ModelRegistry,
    user_token: str,
    monkeypatch,  # type: ignore[valid-type, unused-ignore]
):
    """Test Ray integration with uvloop event loop policy"""
    import asyncio

    ray = pytest.importorskip("ray")
    import uvloop

    def run_test_with_uvloop():
        # Set up uvloop policy in this thread
        asyncio.set_event_loop_policy(uvloop.EventLoopPolicy())
        loop = uvloop.new_event_loop()
        asyncio.set_event_loop(loop)

        try:
            # Start the loop and verify we're actually using uvloop
            async def verify_uvloop():
                current_loop = asyncio.get_running_loop()
                assert isinstance(current_loop, uvloop.Loop), f"Expected uvloop.Loop, got {type(current_loop)}"

            loop.run_until_complete(verify_uvloop())

            # Mock nest_asyncio2.apply to prevent conflicts with uvloop
            monkeypatch.setattr("nest_asyncio2.apply", lambda *args, **kwargs: "patched")
            # Import here to avoid the nest_asyncio2.apply() call during module loading
            from tests.extras.async_task_runner import AsyncTaskRunner

            @ray.remote
            def test_with_ray():
                atr = AsyncTaskRunner()
                # we have to construct a client from scratch due to serialization issues from Ray
                client = ModelRegistry(
                    server_address=client_attrs["host"],
                    port=client_attrs["port"],
                    author=client_attrs["author"],
                    is_secure=client_attrs["ssl"],
                    async_runner=atr.run,
                    user_token=user_token,
                )
                client.register_model(
                    name="test_model",
                    uri="https://acme.org/something",
                    version="v1",
                    model_format_version="random",
                    model_format_name="onnx",
                )
                ma = client.get_model_artifact(name="test_model", version="v1")
                assert ma.uri == "https://acme.org/something"  # type: ignore[union-attr]
                assert ma.model_format_name == "onnx"  # type: ignore[union-attr]

            # Run the Ray test - ray.get is synchronous
            ray.get(test_with_ray.remote())

        finally:
            if not loop.is_closed():
                loop.close()

    # Run the test - ray.get is synchronous and doesn't need the event loop
    run_test_with_uvloop()


@pytest.mark.e2e
def test_upload_artifact_and_register_model_with_default_oci(
    client: ModelRegistry,
    get_temp_dir_with_models,
) -> None:
    # olot is required to run this test
    pytest.importorskip("olot")
    name = "oci-test/defaults"
    version = "0.0.1"
    oci_ref = "localhost:5001/foo/bar:latest"

    model_dir, _ = get_temp_dir_with_models
    upload_params = utils.OCIParams(
        "quay.io/mmortari/hello-world-wait:latest",
        oci_ref,
        custom_oci_backend=utils._get_skopeo_backend(push_args=["--dest-tls-verify=false"]),
    )

    assert client.upload_artifact_and_register_model(
        name,
        model_files_path=model_dir,
        author="Tester McTesterson",
        version=version,
        model_format_name="test format",
        model_format_version="test version",
        upload_params=upload_params,
    )

    assert (ma := client.get_model_artifact(name, version))
    assert ma.uri == f"oci://{oci_ref}"

    # Assert fail on duplicate
    with pytest.raises(StoreError, match="already exists"):
        client.upload_artifact_and_register_model(
            name,
            model_files_path=model_dir,
            version=version,
            model_format_name="test format",
            model_format_version="test version",
            upload_params=upload_params,
        )


@pytest.mark.e2e
def test_upload_artifact_and_register_model_with_default_s3(
    client: ModelRegistry,
    patch_s3_env,
    get_temp_dir_with_models,
) -> None:
    name = "s3-test"
    version = "0.0.1"

    s3_prefix = f"my-model-{version}"
    model_dir, _ = get_temp_dir_with_models

    bucket, s3_endpoint, access_key_id, secret_access_key, region = patch_s3_env

    upload_params = utils.S3Params(
        bucket,
        s3_prefix,
        s3_endpoint,
        access_key_id,
        secret_access_key,
        region,
    )

    assert client.upload_artifact_and_register_model(
        name,
        model_files_path=model_dir,
        author="Tester McTesterson",
        version=version,
        model_format_name="test format",
        model_format_version="test version",
        upload_params=upload_params,
    )

    assert (ma := client.get_model_artifact(name, version))
    assert ma.uri == f"s3://{bucket}/{s3_prefix}?endpoint={s3_endpoint}&defaultRegion={region}"


@pytest.mark.e2e
def test_upload_artifact_and_register_model_missing_upload_params(client):
    with pytest.raises(ValueError, match='Param "upload_params" is required to perform an upload') as e:
        client.upload_artifact_and_register_model(
            "a name",
            model_files_path="/doesnt/matter",
            author="Tester McTesterson",
            version="v0.0.1",
            model_format_name="test format",
            model_format_version="test version",
            upload_params=None,
        )
    assert 'Param "upload_params" is required to perform an upload. Please ensure the value provided is valid' in str(
        e.value
    )


@pytest.mark.e2e
async def test_register_model_with_owner(client):
    model_params = {
        "name": "test_model",
        "uri": "s3",
        "model_format_name": "test_format",
        "model_format_version": "test_version",
        "version": "1.0.0",
        "owner": "test owner",
    }
    rm = client.register_model(
        **model_params,
    )
    assert rm.id
    assert rm.owner == model_params["owner"]
    assert (_get_rm := client.get_registered_model(name=model_params["name"]))
    assert _get_rm.owner == model_params["owner"]


@pytest.mark.e2e
async def test_register_model_with_s3_data_connection(client: ModelRegistry):
    """As a MLOps engineer I want to track a Model from an S3 bucket Data Connection"""
    data_connection_name = "aws-connection-my-data-connection"
    s3_bucket = "my-bucket"
    s3_path = "my-path"
    s3_endpoint = "https://minio-api.acme.org"
    s3_region = "us-east-1"

    # Create the S3 URI using the utility function
    uri = utils.s3_uri_from(path=s3_path, bucket=s3_bucket, endpoint=s3_endpoint, region=s3_region)

    model_params = {
        "name": "test_model",
        "uri": uri,
        "model_format_name": "onnx",
        "model_format_version": "1",
        "version": "v1.0",
        "version_description": "The Model",  # This will be set on the model version
        "storage_key": data_connection_name,
        "storage_path": s3_path,
    }

    # Register the model with S3 connection details
    rm = client.register_model(**model_params)  # type: ignore[arg-type]
    assert rm.id

    # Get and verify the registered model
    rm_by_name = client.get_registered_model(model_params["name"])
    assert rm_by_name.id == rm.id  # type: ignore[union-attr]

    # Get and verify the model version
    mv = client.get_model_version(model_params["name"], model_params["version"])
    assert mv.description == "The Model"  # type: ignore[union-attr]
    assert mv.name == "v1.0"  # type: ignore[union-attr]

    # Get and verify the model artifact
    ma = client.get_model_artifact(model_params["name"], model_params["version"])
    assert ma.uri == uri  # type: ignore[union-attr]
    assert ma.model_format_name == "onnx"  # type: ignore[union-attr]
    assert ma.model_format_version == "1"  # type: ignore[union-attr]
    assert ma.storage_key == data_connection_name  # type: ignore[union-attr]
    assert ma.storage_path == s3_path  # type: ignore[union-attr]


@pytest.mark.e2e
def test_upload_large_model_file(get_large_model_dir, patch_s3_env, client: ModelRegistry):
    """Test uploading and registering a large model file (300-500MB)."""
    pytest.importorskip("boto3")

    # Verify the large model file exists and has correct size
    model_file = os.path.join(get_large_model_dir, "large_model.onnx")
    file_size = os.path.getsize(model_file)
    assert 300 * 1024 * 1024 <= file_size <= 500 * 1024 * 1024, f"File size {file_size} bytes is not in expected range"

    version = "1.0.0"
    prefix = "large_models"
    bucket, s3_endpoint, access_key_id, secret_access_key, region = patch_s3_env

    client.upload_artifact_and_register_model(
        "large_test_model",
        model_files_path=get_large_model_dir,
        author="Tester McTesterson",
        version=version,
        model_format_name="test_format",
        model_format_version="test_version",
        upload_params=utils.S3Params(
            bucket,
            prefix,
            s3_endpoint,
            access_key_id,
            secret_access_key,
            region,
            multipart_threshold=1024 * 1024,  # 1MB
            multipart_chunksize=1024 * 1024,  # 1MB
            max_pool_connections=10,
        ),
    )

    # Verify the model was registered correctly
    mv = client.get_model_artifact(name="large_test_model", version=version)
    assert mv
    assert mv.name == "large_test_model"


@pytest.mark.e2e
async def test_as_mlops_engineer_i_would_like_to_update_a_description_of_the_model(
    client: ModelRegistry,
):
    """As a MLOps engineer I would like to update a description of the model"""
    name = "test_model"
    version = "1.0.0"
    rm = client.register_model(
        name,
        "https://acme.org/something",
        model_format_name="test_format",
        model_format_version="test_version",
        version=version,
        owner="me",
        version_description="Lorem ipsum dolor sit amet",
    )
    assert rm.id

    rm.description = "New description"
    rm = client.update(rm)
    assert rm.description == "New description"
    assert rm.owner == "me"


@pytest.mark.e2e
async def test_as_mlops_engineer_i_would_like_to_store_a_description_of_the_model(
    client: ModelRegistry,
):
    """As a MLOps engineer I would like to store a description of the model
    Note: on Creation, the Description belongs to the Model Version; we could improve the logic to maintain it for the Registered Model if it's not already existing
    """
    name = "test_model"
    version = "1.0.0"
    rm = client.register_model(
        name,
        "https://acme.org/something",
        model_format_name="test_format",
        model_format_version="test_version",
        version=version,
        version_description="consectetur adipiscing elit",
    )
    assert rm.id

    mr_api = client._api
    mv = await mr_api.get_model_version_by_params(rm.id, version)
    assert mv
    assert mv.id
    assert mv.description == "consectetur adipiscing elit"
    ma = await mr_api.get_model_artifact_by_params(name, mv.id)
    assert ma

    rm.description = "Lorem ipsum dolor sit amet"
    assert client.update(rm).description == "Lorem ipsum dolor sit amet"
    mv.description = "consectetur adipiscing elit2"
    assert client.update(mv).description == "consectetur adipiscing elit2"
    ma.description = "sed do eiusmod tempor incididunt"
    assert client.update(ma).description == "sed do eiusmod tempor incididunt"


@pytest.mark.e2e
async def test_as_mlops_engineer_i_would_like_to_store_a_longer_documentation_for_the_model(
    client: ModelRegistry,
    register_model_with_version,
):
    """As a MLOps engineer I would like to store a longer documentation for the model"""
    name = "test_model"
    version = "1.0.0"
    _, mv = register_model_with_version(
        name,
        version,
        uri="https://acme.org/something",
        version_description="consectetur adipiscing elit",
    )

    mr_api = client._api
    da = await mr_api.upsert_model_version_artifact(DocArtifact(uri="https://README.md"), mv.id)
    assert da
    assert da.uri == "https://README.md"


@pytest.fixture
def mock_get_registered_models(monkeypatch):
    """Mock the get_registered_models method to avoid server calls."""
    mock_get_registered_models = MagicMock()
    mock_get_registered_models.return_value.page_size.return_value._next_page.return_value = None
    monkeypatch.setattr(ModelRegistry, "get_registered_models", mock_get_registered_models)
    return mock_get_registered_models


def test_user_token_from_envvar(monkeypatch, mock_get_registered_models):
    """Test for user not providing explicitly user_token,
    reading user token from environment variable."""
    test_token = "test-token-from-envvar"  # noqa: S105

    with tempfile.NamedTemporaryFile(mode="w", delete=False) as token_file:
        token_file.write(test_token)
        token_file_path = token_file.name

    monkeypatch.setenv("KF_PIPELINES_SA_TOKEN_PATH", token_file_path)
    client = ModelRegistry(
        server_address="http://localhost:8080",
        author="test_author",
        is_secure=False,
        # user_token=None -> ... Let it read from Env var
    )
    assert client is not None
    assert client._api.config.access_token == test_token

    os.unlink(token_file_path)


def test_user_token_from_k8s_file(monkeypatch, mock_get_registered_models):
    """Test for user not providing explicitly user_token,
    reading user token from Kubernetes service account token file."""
    test_token = "test-token-from-k8s-file"  # noqa: S105

    monkeypatch.delenv("KF_PIPELINES_SA_TOKEN_PATH", raising=False)
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as k8s_token_file:
        k8s_token_file.write(test_token)
        k8s_token_file_path = k8s_token_file.name
    monkeypatch.setattr("model_registry._client.DEFAULT_K8S_SA_TOKEN_PATH", k8s_token_file_path)
    client = ModelRegistry(
        server_address="http://localhost:8080",
        author="test_author",
        is_secure=False,
        # user_token=None -> ... Let it read from K8s file
    )
    assert client is not None
    assert client._api.config.access_token == test_token
    os.unlink(k8s_token_file_path)


def test_user_token_envvar_priority_over_k8s(monkeypatch, mock_get_registered_models):
    """Test for user not providing explicitly user_token,
    reading user token from environment variable,
    taking precedence over K8s file for Service Account token."""
    env_token = "test-token-from-envvar"  # noqa: S105
    k8s_token = "test-token-from-k8s-file"  # noqa: S105

    with tempfile.NamedTemporaryFile(mode="w", delete=False) as env_token_file:
        env_token_file.write(env_token)
        env_token_path = env_token_file.name
    monkeypatch.setenv("KF_PIPELINES_SA_TOKEN_PATH", env_token_path)
    with tempfile.NamedTemporaryFile(mode="w", delete=False) as k8s_token_file:
        k8s_token_file.write(k8s_token)
        k8s_token_file_path = k8s_token_file.name
    monkeypatch.setattr("model_registry._client.DEFAULT_K8S_SA_TOKEN_PATH", k8s_token_file_path)
    client = ModelRegistry(
        server_address="http://localhost:8080",
        author="test_author",
        is_secure=False,
        # user_token=None -> ... Let it read from Env var (and not from K8s file, given that Env var is set)
    )
    assert client is not None
    assert client._api.config.access_token == env_token
    os.unlink(env_token_path)
    os.unlink(k8s_token_file_path)


def test_user_token_missing_warning(monkeypatch, mock_get_registered_models):
    """Test for user not providing explicitly user_token,
    after trying to read from envvar and K8s file for Service Account token but both are missing,
    it will emit a warning."""
    monkeypatch.delenv("KF_PIPELINES_SA_TOKEN_PATH", raising=False)
    mock_warn = MagicMock()
    monkeypatch.setattr("model_registry._client.warn", mock_warn)
    client = ModelRegistry(
        server_address="http://localhost:8080",
        author="test_author",
        is_secure=False,
        # user_token=None -> ... Let it try to read, but missing Env var and K8s file, it will emit a warning
    )
    assert client is not None
    mock_warn.assert_called_with("User access token is missing", stacklevel=2)


def test_hint_server_address_port_https_with_standard_port_no_warning(mock_get_registered_models, caplog):
    """Test cases for the hint_server_address_port method via ModelRegistry constructor.
    Test that no warning is issued when using HTTPS with port 443."""
    with caplog.at_level(logging.WARNING):
        ModelRegistry(server_address="https://example.com", port=443, author="test", user_token="test")  # noqa: S106

    assert len(caplog.records) == 0


def test_hint_server_address_port_https_with_port_ending_443_no_warning(mock_get_registered_models, caplog):
    """Test cases for the hint_server_address_port method via ModelRegistry constructor.
    Test that no warning is issued when using HTTPS with port ending in 443."""
    with caplog.at_level(logging.WARNING):
        ModelRegistry(server_address="https://example.com", port=8443, author="test", user_token="test")  # noqa: S106

    assert len(caplog.records) == 0


def test_hint_server_address_port_https_with_non_443_port_warning(mock_get_registered_models, caplog):
    """Test cases for the hint_server_address_port method via ModelRegistry constructor.
    Test that a warning is issued when using HTTPS with non-443 port."""
    with caplog.at_level(logging.WARNING):
        ModelRegistry(server_address="https://example.com", port=8080, author="test", user_token="test")  # noqa: S106

    assert len(caplog.records) == 1
    assert "Server address protocol is https://, but port is not 443 or ending with 443" in caplog.records[0].message


def test_hint_server_address_port_http_with_standard_port_no_warning(mock_get_registered_models, caplog):
    """Test cases for the hint_server_address_port method via ModelRegistry constructor.
    Test that no warning is issued when using HTTP with port 80."""
    with caplog.at_level(logging.WARNING):
        ModelRegistry(server_address="http://example.com", port=80, author="test", is_secure=False)

    assert len(caplog.records) == 0


def test_hint_server_address_port_http_with_port_ending_80_no_warning(mock_get_registered_models, caplog):
    """Test cases for the hint_server_address_port method via ModelRegistry constructor.
    Test that no warning is issued when using HTTP with port ending in 80."""
    with caplog.at_level(logging.WARNING):
        ModelRegistry(server_address="http://example.com", port=8080, author="test", is_secure=False)

    assert len(caplog.records) == 0


def test_hint_server_address_port_http_with_non_80_port_warning(mock_get_registered_models, caplog):
    """Test cases for the hint_server_address_port method via ModelRegistry constructor.
    Test that a warning is issued when using HTTP with non-80 port."""
    with caplog.at_level(logging.WARNING):
        ModelRegistry(server_address="http://example.com", port=8443, author="test", is_secure=False)

    assert len(caplog.records) == 1
    assert "Server address protocol is http://, but port is not 80 or ending with 80" in caplog.records[0].message
