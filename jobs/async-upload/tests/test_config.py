import os
import random
import tempfile
import shutil
import base64
import json
from pathlib import Path
import pytest
from job.config import get_config

from job.models import S3StorageConfig, OCIStorageConfig, URISourceStorageConfig

MR_PREFIX = "MODEL_SYNC"
MR_SOURCE_PREFIX = "MODEL_SYNC_SOURCE"
MR_DEST_PREFIX = "MODEL_SYNC_DESTINATION"


@pytest.fixture
def source_s3_env_vars(monkeypatch):
    """Fixture to set up s3-specific source environment variables"""

    vars = {
        "type": "s3",
        "aws_access_key_id": "source_key_env",
        "aws_secret_access_key": "source_secret_env",
        "aws_region": "us-east-1",
        "aws_bucket": "source-bucket-env",
        "aws_key": "source-key-env",
    }

    for key, value in vars.items():
        monkeypatch.setenv(f"MODEL_SYNC_SOURCE_{key.upper()}", value)

    yield vars


@pytest.fixture
def destination_oci_env_vars(monkeypatch):
    """Fixture to set up oci-specific destination environment variables"""
    vars = {
        "type": "oci",
        "oci_uri": "quay.io/example/oci",
        "oci_registry": "quay.io",
        "oci_username": "oci_username_env",
        "oci_password": "oci_password_env",
    }

    # Set up test environment variables
    for key, value in vars.items():
        monkeypatch.setenv(f"MODEL_SYNC_DESTINATION_{key.upper()}", value)

    yield vars

@pytest.fixture
def create_model_intent_env_vars(monkeypatch):
    """Fixture to set up environment variables for testing"""
    vars = {
        "registry_server_address": "https://registry.example.com",
    }
    for key, value in vars.items():
        monkeypatch.setenv(f"MODEL_SYNC_{key.upper()}", value)

    yield vars

@pytest.fixture
def create_version_intent_env_vars(monkeypatch):
    """Fixture to set up environment variables for testing"""
    vars = {
        "model_id": "1234",
        "registry_server_address": "https://registry.example.com",
    }
    for key, value in vars.items():
        monkeypatch.setenv(f"MODEL_SYNC_{key.upper()}", value)

    yield vars

@pytest.fixture
def update_artifact_intent_env_vars(monkeypatch):
    """Fixture to set up environment variables for testing"""
    vars = {
        "model_artifact_id": "1234",
        "registry_server_address": "https://registry.example.com",
    }
    for key, value in vars.items():
        monkeypatch.setenv(f"MODEL_SYNC_{key.upper()}", value)

    yield vars

@pytest.fixture
def s3_credentials_folder():
    """
    Fixture to create temporary config files for testing
    """
    # Create source config file
    with tempfile.TemporaryDirectory() as temp_dir:
        # Create the credentials files and store their contents
        credentials = {
            "access_key_id": f"file_key_{random.randint(1000, 9999)}",
            "secret_access_key": f"file_secret_{random.randint(1000, 9999)}",
            "region": f"eu-west-1_{random.randint(1000, 9999)}",
            "bucket": f"file-bucket_{random.randint(1000, 9999)}",
            "endpoint": f"file-endpoint_{random.randint(1000, 9999)}",
        }

        # Write the credentials to files
        with open(os.path.join(temp_dir, "AWS_ACCESS_KEY_ID"), "w") as f:
            f.write(credentials["access_key_id"])
        with open(os.path.join(temp_dir, "AWS_SECRET_ACCESS_KEY"), "w") as f:
            f.write(credentials["secret_access_key"])
        with open(os.path.join(temp_dir, "AWS_REGION"), "w") as f:
            f.write(credentials["region"])
        with open(os.path.join(temp_dir, "AWS_S3_BUCKET"), "w") as f:
            f.write(credentials["bucket"])
        with open(os.path.join(temp_dir, "AWS_S3_ENDPOINT"), "w") as f:
            f.write(credentials["endpoint"])

        yield Path(temp_dir), credentials

        # Clean up temporary folder
        shutil.rmtree(temp_dir)


@pytest.fixture
def uri_credentials_folder(tmp_path):
    """
    Fixture to create temporary URI credentials files for testing
    """
    # Create the URI credential file and store its content
    uri_value = f"hf://test/model/{random.randint(1000, 9999)}"

    # Write the URI to file
    uri_file = tmp_path / "URI"
    uri_file.write_text(uri_value)

    return tmp_path, {"uri": uri_value}


@pytest.fixture
def create_oci_credentials_file(tmp_path):
    """Factory fixture to create OCI credentials file (.dockerconfigjson) from a docker config dict"""
    def _create_file(dockerconfigjson):
        config_file = tmp_path / ".dockerconfigjson"
        config_file.write_text(json.dumps(dockerconfigjson))
        return config_file
    return _create_file


def test_s3_file_to_oci_env_config(
    s3_credentials_folder, destination_oci_env_vars, update_artifact_intent_env_vars
):
    """Tests a configuration where the source is S3, using a credentials path, to a destination with OCI env vars"""
    folder_location, expected_credentials = s3_credentials_folder

    # The same folder/files are used for both source and destination
    config = get_config(
        [
            "--source-type",
            "s3",
            "--source-s3-credentials-path",
            str(folder_location),
            "--source-aws-key",
            "my-key", # see samples/sample_job_s3_to_oci.yaml, as 'key' is not provided in the Secret (#1256)
            "--registry-server-address",
            "https://registry.example.com",
        ]
    )

    # Source credentials were not set, so they are None
    assert isinstance(config.source, S3StorageConfig)
    assert config.source.access_key_id == expected_credentials["access_key_id"]
    assert config.source.secret_access_key == expected_credentials["secret_access_key"]
    assert config.source.region == expected_credentials["region"]
    assert config.source.bucket == expected_credentials["bucket"]
    assert config.source.endpoint == expected_credentials["endpoint"]
    assert config.source.key == "my-key" # see samples/sample_job_s3_to_oci.yaml, as 'key' is not provided in the Secret (#1256)

    assert isinstance(config.destination, OCIStorageConfig)
    assert config.destination.uri == destination_oci_env_vars["oci_uri"]
    assert config.destination.username == destination_oci_env_vars["oci_username"]
    assert config.destination.password == destination_oci_env_vars["oci_password"]


def test_env_based_s3_to_oci_config(
    update_artifact_intent_env_vars, source_s3_env_vars, destination_oci_env_vars
):
    """Test configuration using environment variables"""
    config = get_config([])

    assert isinstance(config.source, S3StorageConfig)
    assert config.source.access_key_id == source_s3_env_vars["aws_access_key_id"]
    assert (
        config.source.secret_access_key
        == source_s3_env_vars["aws_secret_access_key"]
    )
    assert config.source.region == source_s3_env_vars["aws_region"]
    assert config.source.bucket == source_s3_env_vars["aws_bucket"]

    assert isinstance(config.destination, OCIStorageConfig)
    assert config.destination.uri == destination_oci_env_vars["oci_uri"]
    assert config.destination.username == destination_oci_env_vars["oci_username"]
    assert config.destination.password == destination_oci_env_vars["oci_password"]

    assert config.model.intent.artifact_id == update_artifact_intent_env_vars["model_artifact_id"]
    assert config.registry.server_address == update_artifact_intent_env_vars["registry_server_address"]


def test_params_based_config():
    """Test configuration using parameters"""
    config = get_config(
        [
            "--source-type",
            "oci",
            "--source-oci-uri",
            "quay.io/example/params",
            "--source-oci-registry",
            "quay.io",
            "--destination-type",
            "s3",
            "--destination-aws-bucket",
            "destination-bucket-params",
            "--destination-aws-key",
            "destination-key-params",
            "--destination-aws-region",
            "eu-central-1",
            "--destination-aws-access-key-id",
            "destination_key_params",
            "--destination-aws-secret-access-key",
            "destination_secret_params",
            "--model-upload-intent",
            "update_artifact",
            "--model-artifact-id",
            "5678",
            "--registry-server-address",
            "https://registry.example.com",
        ]
    )

    assert isinstance(config.source, OCIStorageConfig)
    assert config.source.uri == "quay.io/example/params"
    assert isinstance(config.destination, S3StorageConfig)
    assert config.destination.access_key_id == "destination_key_params"
    assert config.destination.secret_access_key == "destination_secret_params"
    assert config.destination.region == "eu-central-1"
    assert config.destination.bucket == "destination-bucket-params"


def test_params_will_override_env_config(
    update_artifact_intent_env_vars, source_s3_env_vars, destination_oci_env_vars
):
    """Test a configuration in which ENV vars are set, but override params are provided to the CLI"""

    override_vars = {
        "aws_bucket": f"source-bucket-{random.randint(1000, 9999)}",
        "aws_region": f"eu-central-{random.randint(1000, 9999)}",
        "aws_access_key_id": f"source-key-{random.randint(1000, 9999)}",
        "aws_secret_access_key": f"source-secret-{random.randint(1000, 9999)}",
    }
    config = get_config(
        [
            "--source-type",
            "s3",
            "--source-aws-bucket",
            override_vars["aws_bucket"],
            "--source-aws-region",
            override_vars["aws_region"],
            "--source-aws-access-key-id",
            override_vars["aws_access_key_id"],
            "--source-aws-secret-access-key",
            override_vars["aws_secret_access_key"],
            "--model-upload-intent",
            "update_artifact",
            "--model-artifact-id",
            "5678",
            "--registry-server-address",
            "https://registry.example.com",
        ]
    )

    assert isinstance(config.source, S3StorageConfig)
    assert config.source.access_key_id == override_vars["aws_access_key_id"]
    assert config.source.secret_access_key == override_vars["aws_secret_access_key"]
    assert config.source.region == override_vars["aws_region"]
    assert config.source.bucket == override_vars["aws_bucket"]


def test_uri_file_to_oci_env_config(uri_credentials_folder, destination_oci_env_vars, update_artifact_intent_env_vars):
    """Tests a configuration where the source is URI, using a credentials path, to a destination with OCI env vars"""
    folder_location, expected_credentials = uri_credentials_folder

    config = get_config(
        [
            "--source-type",
            "uri",
            "--source-uri-credentials-path",
            str(folder_location),
            "--registry-server-address",
            "https://registry.example.com",
        ]
    )

    # Source credentials were loaded from file
    assert isinstance(config.source, URISourceStorageConfig)
    assert config.source.uri == expected_credentials["uri"]
    assert config.source.credentials_path == str(folder_location)

    assert isinstance(config.destination, OCIStorageConfig)
    assert config.destination.uri == destination_oci_env_vars["oci_uri"]
    assert config.destination.username == destination_oci_env_vars["oci_username"]
    assert config.destination.password == destination_oci_env_vars["oci_password"]


def test_uri_params_to_oci_config(update_artifact_intent_env_vars, destination_oci_env_vars):
    """Test URI source configuration using CLI parameters"""
    uri_value = "hf://test/model/params"

    config = get_config(
        [
            "--source-type",
            "uri",
            "--source-uri",
            uri_value,
            "--registry-server-address",
            "https://registry.example.com",
        ]
    )

    assert isinstance(config.source, URISourceStorageConfig)
    assert config.source.uri == uri_value
    assert config.source.credentials_path is None

    assert isinstance(config.destination, OCIStorageConfig)
    assert config.destination.uri == destination_oci_env_vars["oci_uri"]


def test_uri_credentials_override_params(uri_credentials_folder, destination_oci_env_vars, update_artifact_intent_env_vars):
    """Test that URI credentials from file override CLI parameters"""
    folder_location, expected_credentials = uri_credentials_folder
    cli_uri = "hf://cli/model/override"

    config = get_config(
        [
            "--source-type",
            "uri",
            "--source-uri",
            cli_uri,
            "--source-uri-credentials-path",
            str(folder_location),
            "--registry-server-address",
            "https://registry.example.com",
        ]
    )

    # Credentials from file should override CLI parameter
    assert isinstance(config.source, URISourceStorageConfig)
    assert config.source.uri == expected_credentials["uri"]  # From file, not CLI
    assert config.source.credentials_path == str(folder_location)


def test_uri_credentials_missing_folder_error(update_artifact_intent_env_vars, destination_oci_env_vars):
    """Test that missing credentials folder raises appropriate error"""
    with pytest.raises(FileNotFoundError, match="credentials folder not found"):
        get_config(
            [
                "--source-type",
                "uri",
                "--source-uri-credentials-path",
                "/non/existent/path",
                "--registry-server-address",
                "https://registry.example.com",
            ]
        )


def test_uri_credentials_missing_uri_file_error(update_artifact_intent_env_vars, destination_oci_env_vars, tmp_path):
    """Test that missing URI file in credentials folder raises appropriate error"""
    # Create empty directory without URI file
    with pytest.raises(FileNotFoundError, match="URI credential file not found"):
        get_config(
            [
                "--source-type",
                "uri",
                "--source-uri-credentials-path",
                str(tmp_path),
                "--registry-server-address",
                "https://registry.example.com",
            ]
        )


def test_uri_credentials_env_var_support(uri_credentials_folder, update_artifact_intent_env_vars, destination_oci_env_vars, tmp_path, monkeypatch):
    """Test that URI credentials path can be set via environment variable"""
    # Create URI credential file
    folder_location, expected_credentials = uri_credentials_folder

    with monkeypatch.context() as mp:
        # Set environment variable
        mp.setenv("MODEL_SYNC_SOURCE_URI_CREDENTIALS_PATH", str(folder_location))

        config = get_config(
            [
                "--source-type",
                "uri",
                "--registry-server-address",
                "https://registry.example.com",
            ]
        )

    # Credentials should be loaded from environment variable path
    assert isinstance(config.source, URISourceStorageConfig)
    assert config.source.uri == expected_credentials["uri"]
    assert config.source.credentials_path == str(tmp_path)


@pytest.mark.parametrize("dockerconfigjson,expected", [
    pytest.param(
        {
            "auths": {
                "registry.example.com": {
                    "auth": base64.b64encode(b"testuser:testpass123").decode(),
                    "email": "test@example.com"
                }
            }
        },
        {
            "username": "testuser",
            "password": "testpass123",
            "email": "test@example.com",
            "registry": "registry.example.com",
        },
        id="base64-encoded"
    ),
    pytest.param(
        {
            "auths": {
                "registry.example.com": {
                    "auth": "plainuser:plainpass456",
                    "email": "plain@example.com"
                }
            }
        },
        {
            "username": "plainuser",
            "password": "plainpass456",
            "email": "plain@example.com",
            "registry": "registry.example.com",
        },
        id="plaintext"
    ),
    pytest.param(
        {
            "auths": {
                "registry.example.com": {
                    "auth": "",
                    "email": "user@example.com"
                }
            }
        },
        {
            "username": None,
            "password": None,
            "email": "user@example.com",
            "registry": "registry.example.com",
        },
        id="empty-auth"
    ),
])
def test_oci_credentials_file_loading(
    dockerconfigjson,
    expected,
    create_oci_credentials_file,
    monkeypatch
):
    """
    Test OCI credentials loaded from .dockerconfigjson file for both source and destination.

    Test cases:
    - base64-encoded: Docker spec compliant (base64-encoded "username:password")
    - plaintext: Backwards compatibility (plain "username:password")
    - empty-auth: Edge case (empty string)
    """
    config_file = create_oci_credentials_file(dockerconfigjson)

    # Set OCI source env vars
    monkeypatch.setenv("MODEL_SYNC_SOURCE_TYPE", "oci")
    monkeypatch.setenv("MODEL_SYNC_SOURCE_OCI_URI", f"{expected['registry']}/source/model")
    monkeypatch.setenv("MODEL_SYNC_SOURCE_OCI_REGISTRY", expected["registry"])

    # Set OCI destination env vars
    monkeypatch.setenv("MODEL_SYNC_DESTINATION_TYPE", "oci")
    monkeypatch.setenv("MODEL_SYNC_DESTINATION_OCI_URI", f"{expected['registry']}/dest/model")
    monkeypatch.setenv("MODEL_SYNC_DESTINATION_OCI_REGISTRY", expected["registry"])

    monkeypatch.setenv("MODEL_SYNC_REGISTRY_SERVER_ADDRESS", "https://registry.example.com")

    config = get_config([
        "--source-oci-credentials-path", str(config_file),
        "--destination-oci-credentials-path", str(config_file),
        "--model-artifact-id", "123"
    ])

    # Verify source OCI credentials
    assert isinstance(config.source, OCIStorageConfig)
    assert config.source.username == expected["username"]
    assert config.source.password == expected["password"]
    assert config.source.email == expected["email"]

    # Verify destination OCI credentials
    assert isinstance(config.destination, OCIStorageConfig)
    assert config.destination.username == expected["username"]
    assert config.destination.password == expected["password"]
    assert config.destination.email == expected["email"]


def test_registry_custom_ca_uses_existing_file(
    update_artifact_intent_env_vars, source_s3_env_vars, destination_oci_env_vars, tmp_path
):
    custom_ca = tmp_path / "ca-bundle.crt"
    custom_ca.write_text("test ca bundle")

    config = get_config([
        "--registry-custom-ca", str(custom_ca),
    ])

    assert config.registry.custom_ca == str(custom_ca)


def test_registry_custom_ca_missing_file_falls_back_to_system_defaults(
    update_artifact_intent_env_vars, source_s3_env_vars, destination_oci_env_vars, tmp_path
):
    missing_custom_ca = tmp_path / "missing-ca-bundle.crt"

    config = get_config([
        "--registry-custom-ca", str(missing_custom_ca),
    ])

    assert config.registry.custom_ca is None


