import io
import mimetypes
import os
import pytest
import tarfile
import zipfile
from pathlib import Path
from unittest.mock import Mock, call, patch
from job.download import download_from_s3, unpack_archive_file
from job.config import get_config
from job.mr_client import validate_and_get_model_registry_client
from job.models import S3StorageConfig

DUMMY_FILE_DATA = {
    "file1.txt": b"test file 1",
    "dir/file2.txt": b"abc",
    "file3.log": b"123"
}


@pytest.fixture
def minimal_update_artifact_env_source_dest_vars():
    original_env = dict(os.environ)

    # Destination variables
    dest_vars = {
        "type": "oci",
        "oci_uri": "quay.io/example/oci",
        "oci_registry": "quay.io",
        "oci_username": "oci_username_env",
        "oci_password": "oci_password_env",
    }

    # Source variables - using the correct format from existing tests
    source_vars = {
        "type": "s3",
        "aws_bucket": "test-bucket",
        "aws_key": "test-key",
        "aws_access_key_id": "test-access-key-id",
        "aws_secret_access_key": "test-secret-access-key",
        "aws_endpoint": "http://localhost:9000",
    }

    # Set up test environment variables
    for key, value in dest_vars.items():
        os.environ[f"MODEL_SYNC_DESTINATION_{key.upper()}"] = value
    for key, value in source_vars.items():
        os.environ[f"MODEL_SYNC_SOURCE_{key.upper()}"] = value

    # Model and registry variables
    model_vars = {
        "model_upload_intent": "update_artifact",
        "model_artifact_id": "123",
        "registry_server_address": "http://localhost",
        "registry_port": "8080",
        "registry_author": "author",
        "storage_path": "/tmp/model-sync",
    }

    for key, value in model_vars.items():
        os.environ[f"MODEL_SYNC_{key.upper()}"] = value

    yield model_vars

    # Restore original environment
    os.environ.clear()
    os.environ.update(original_env)


@pytest.fixture
def dummy_archive(request, tmp_path):
    match request.param:
        case "tar":
            return create_dummy_tar(tmp_path / "dummy.tar", "w")
        case "tar.gz":
            return create_dummy_tar(tmp_path / "dummy.tar.gz", "w:gz")
        case "zip":
            return create_dummy_zip(tmp_path / "dummy.zip", "w")
        case _:
            raise ValueError(f"Unsupported archive type: {request.param}")


def create_dummy_tar(path, mode):
    with tarfile.open(path, mode) as f:
        for filename, content in DUMMY_FILE_DATA.items():
            file_data = io.BytesIO(content)
            tarinfo = tarfile.TarInfo(name=filename)
            tarinfo.size = len(content)
            f.addfile(tarinfo, fileobj=file_data)
    return path


def create_dummy_zip(path, mode):
    with zipfile.ZipFile(path, mode) as f:
        for filename, content in DUMMY_FILE_DATA.items():
            f.writestr(filename, content)
    return path


@pytest.mark.parametrize("dummy_archive", ["tar", "tar.gz", "zip"], indirect=True)
def test_unpack_archive_file(dummy_archive, tmp_path):
    dest_dir = tmp_path / "unpacked_archive"
    mimetype = mimetypes.guess_type(dummy_archive)[0]
    unpack_archive_file(dummy_archive, mimetype, dest_dir)

    result = {}
    for dirpath, _, filenames in os.walk(dest_dir):
        for filename in filenames:
            filepath = Path(dirpath) / filename
            key = filepath.relative_to(dest_dir).as_posix()
            contents = filepath.read_bytes()
            result[key] = contents
    assert result == DUMMY_FILE_DATA


@pytest.mark.parametrize("source_key", ["test-key", "test-key/"])
def test_download_from_s3(minimal_update_artifact_env_source_dest_vars, source_key):
    """Test download_from_s3 pages through a directory prefix."""

    os.environ["MODEL_SYNC_SOURCE_AWS_KEY"] = source_key

    # load config from your fixture
    config = get_config([])

    # sanity-check config
    assert isinstance(config.source, S3StorageConfig)
    assert config.source.bucket == "test-bucket"
    assert config.source.key == source_key

    # use whatever path came back in config
    storage_path = config.storage.path
    assert storage_path == "/tmp/model-sync"

    # mock out ModelRegistry so validate_and_get_model_registry_client returns a dummy client
    with patch("job.mr_client.ModelRegistry") as mock_registry_class:
        mock_registry_class.return_value = Mock()
        client = validate_and_get_model_registry_client(config.registry)

    # now patch _connect_to_s3 and os.makedirs
    with patch("job.download._connect_to_s3") as mock_connect, \
         patch("os.makedirs") as mock_makedirs:

        # prepare our fake s3 client + transfer config
        mock_s3 = Mock()
        mock_transfer_cfg = Mock()
        mock_connect.return_value = (mock_s3, mock_transfer_cfg)

        # set up a paginator that yields a single page with two entries
        fake_page = {
            "Contents": [
                {"Key": "test-key/file1.txt"},
                {"Key": "test-key/dir/"},                # should be skipped
                {"Key": "test-key/dir/file2.bin"},
            ]
        }
        mock_paginator = Mock()
        mock_paginator.paginate.return_value = [fake_page]
        mock_s3.get_paginator.return_value = mock_paginator

        # call under test
        download_from_s3(config.source, config.storage.path)

        # ensure _connect_to_s3 got all args including multipart settings
        mock_connect.assert_called_once_with(
            "http://localhost:9000",
            "test-access-key-id",
            "test-secret-access-key",
            None,  # region
            multipart_threshold=1024 * 1024,
            multipart_chunksize=1024 * 1024,
            max_pool_connections=10,
        )

        # ensure we asked for the right paginator and paginated correctly
        mock_s3.get_paginator.assert_called_once_with("list_objects_v2")
        mock_paginator.paginate.assert_called_once_with(
            Bucket="test-bucket",
            Prefix="test-key/",
        )

        # build expected download calls using the real storage_path
        expected = [
            call(
                "test-bucket",
                "test-key/file1.txt",
                os.path.join(storage_path, "file1.txt"),
            ),
            call(
                "test-bucket",
                "test-key/dir/file2.bin",
                os.path.join(storage_path, "dir", "file2.bin"),
            ),
        ]
        mock_s3.download_file.assert_has_calls(expected, any_order=False)

        # directories should be created for each file
        mock_makedirs.assert_any_call(
            os.path.dirname(os.path.join(storage_path, "file1.txt")),
            exist_ok=True
        )
        mock_makedirs.assert_any_call(
            os.path.dirname(os.path.join(storage_path, "dir", "file2.bin")),
            exist_ok=True
        )


def test_download_from_s3_with_region(minimal_update_artifact_env_source_dest_vars):
    """Test download_from_s3 function with region specified"""

    # Set region in environment
    os.environ["MODEL_SYNC_SOURCE_AWS_REGION"] = "us-west-2"

    config = get_config([])

    # Create mock ModelRegistry client
    with patch("job.mr_client.ModelRegistry") as mock_registry_class:
        mock_client = Mock()
        mock_registry_class.return_value = mock_client
        client = validate_and_get_model_registry_client(config.registry)

    # Mock the S3 client and _connect_to_s3 function
    with patch("job.download._connect_to_s3") as mock_connect, \
         patch("os.makedirs"):  # silence dir creation
        mock_s3_client = Mock()
        mock_transfer_config = Mock()
        mock_connect.return_value = (mock_s3_client, mock_transfer_config)

        # set up a paginator that yields a single page with two entries
        fake_page = {
            "Contents": [
                {"Key": "test-key/file1.txt"},
                {"Key": "test-key/dir/"},                # should be skipped
                {"Key": "test-key/dir/file2.bin"},
            ]
        }
        mock_paginator = Mock()
        mock_paginator.paginate.return_value = [fake_page]
        mock_s3_client.get_paginator.return_value = mock_paginator

        # Call the function under test
        download_from_s3(config.source, config.storage.path)

        # Verify _connect_to_s3 was called with correct parameters including region
        mock_connect.assert_called_once_with(
            "http://localhost:9000",
            "test-access-key-id",
            "test-secret-access-key",
            "us-west-2",
            multipart_threshold=1024 * 1024,
            multipart_chunksize=1024 * 1024,
            max_pool_connections=10,
        )


def test_download_from_s3_connection_error(minimal_update_artifact_env_source_dest_vars):
    """Test download_from_s3 function when S3 connection fails"""

    config = get_config([])

    # Create mock ModelRegistry client
    with patch("job.mr_client.ModelRegistry") as mock_registry_class:
        mock_client = Mock()
        mock_registry_class.return_value = mock_client
        client = validate_and_get_model_registry_client(config.registry)

        # Mock _connect_to_s3 to raise an exception
        with patch("job.download._connect_to_s3") as mock_connect:
            mock_connect.side_effect = Exception("Connection failed")

            # Test that the exception is propagated
            with pytest.raises(Exception, match="Connection failed"):
                download_from_s3(config.source, config.storage.path)


def test_download_from_s3_download_error(minimal_update_artifact_env_source_dest_vars):
    """Test download_from_s3 function when file download fails"""

    config = get_config([])

    # Create mock ModelRegistry client
    with patch("job.mr_client.ModelRegistry") as mock_registry_class:
        mock_client = Mock()
        mock_registry_class.return_value = mock_client
        client = validate_and_get_model_registry_client(config.registry)

    # Mock the S3 client, paginator, and _connect_to_s3 function
    with patch("job.download._connect_to_s3") as mock_connect, \
         patch("os.makedirs"):  # silence dir creation
        mock_s3_client = Mock()
        mock_transfer_config = Mock()
        mock_connect.return_value = (mock_s3_client, mock_transfer_config)

        # Stub out pagination so we get one file to download
        fake_page = {
            "Contents": [
                {"Key": "test-key/failing-file.txt"},
            ]
        }
        mock_paginator = Mock()
        mock_paginator.paginate.return_value = [fake_page]
        mock_s3_client.get_paginator.return_value = mock_paginator

        # Have download_file raise
        mock_s3_client.download_file.side_effect = Exception("Download failed")

        # Now the loop will hit download_file and propagate our exception
        with pytest.raises(Exception, match="Download failed"):
            download_from_s3(config.source, config.storage.path)
