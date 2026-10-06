import io
import mimetypes
import os
import pytest
import tarfile
import zipfile
from pathlib import Path
from unittest.mock import Mock, call, patch
from job.download import download_from_http, download_from_s3, unpack_archive_file
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
    assert unpack_archive_file(dummy_archive, mimetype, dest_dir) == dest_dir
    assert not dummy_archive.exists()

    result = {}
    for dirpath, _, filenames in os.walk(dest_dir):
        for filename in filenames:
            filepath = Path(dirpath) / filename
            key = filepath.relative_to(dest_dir).as_posix()
            contents = filepath.read_bytes()
            result[key] = contents
    assert result == DUMMY_FILE_DATA


@pytest.fixture
def permissive_tar_defaults(monkeypatch):
    # Explicit filtering must work even when runtime defaults trust the archive.
    monkeypatch.setattr(
        tarfile.TarFile, "extraction_filter", staticmethod(tarfile.fully_trusted_filter)
    )


def create_test_archive(tmp_path, archive_type, entries):
    path = tmp_path / f"archive.{archive_type}"
    if archive_type == "zip":
        with zipfile.ZipFile(path, "w") as archive:
            for name, content in entries:
                archive.writestr(name, content)
    else:
        with (
            tarfile.open(path, "w:gz") if archive_type == "tar.gz" else tarfile.open(path, "w")
        ) as archive:
            for member, content in entries:
                if isinstance(member, str):
                    member = tarfile.TarInfo(member)
                member.size = len(content)
                archive.addfile(member, io.BytesIO(content))
    return path


def create_http_response(archive: Path) -> Mock:
    response = Mock()
    response.headers = {"Content-Type": mimetypes.guess_type(archive)[0]}
    response.raw = io.BytesIO(archive.read_bytes())
    response.__enter__ = Mock(return_value=response)
    response.__exit__ = Mock(return_value=False)
    return response


@pytest.mark.parametrize("archive_type", ["tar", "tar.gz"])
@pytest.mark.parametrize("name", ["../outside.txt", "../unpacked-sibling/outside.txt"])
def test_unpack_rejects_tar_traversal(tmp_path, permissive_tar_defaults, archive_type, name):
    dest_dir = tmp_path / "unpacked"
    outside = (dest_dir / name).resolve()
    outside.parent.mkdir(parents=True, exist_ok=True)
    outside.write_bytes(b"original")
    archive = create_test_archive(tmp_path, archive_type, [(name, b"overwrite")])

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, "application/x-tar", dest_dir)

    assert isinstance(error.value.__cause__, tarfile.FilterError)
    assert outside.read_bytes() == b"original"
    assert archive.exists()


@pytest.mark.parametrize("name", [
    "../outside.txt",
    "nested/../../outside.txt",
    "nested/../model.bin",
    r"..\outside.txt",
    r"nested\..\model.bin",
    "/absolute.txt",
    r"\absolute.txt",
    "C:/outside.txt",
    r"C:\outside.txt",
    "C:outside.txt",
    r"\\server\share\outside.txt",
])
def test_unpack_rejects_suspicious_zip_names_before_extraction(tmp_path, name):
    dest_dir = tmp_path / "unpacked"
    outside = tmp_path / "outside.txt"
    outside.write_bytes(b"original")
    archive = create_test_archive(tmp_path, "zip", [
        ("valid/model.bin", b"valid"),
        (name, b"overwrite"),
    ])

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, "application/zip", dest_dir)

    assert error.value.__cause__ is not None
    assert outside.read_bytes() == b"original"
    assert not (dest_dir / "valid/model.bin").exists()
    assert archive.exists()


@pytest.mark.parametrize("archive_type", ["tar", "zip"])
@pytest.mark.parametrize("symlink_path", ["nested", "nested/model.bin"])
def test_unpack_rejects_existing_symlink_escape(
    tmp_path, permissive_tar_defaults, archive_type, symlink_path
):
    dest_dir = tmp_path / "unpacked"
    outside_dir = tmp_path / "unpacked-sibling"
    outside_dir.mkdir()
    outside = outside_dir / "model.bin"
    outside.write_bytes(b"original")
    link = dest_dir / symlink_path
    link.parent.mkdir(parents=True)
    link.symlink_to(outside_dir if symlink_path == "nested" else outside)
    archive = create_test_archive(tmp_path, archive_type, [("nested/model.bin", b"overwrite")])
    mimetype = mimetypes.guess_type(archive)[0]
    assert mimetype is not None

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$"):
        unpack_archive_file(archive, mimetype, dest_dir)

    assert outside.read_bytes() == b"original"
    assert link.is_symlink()
    assert archive.exists()


@pytest.mark.parametrize("link_type", [tarfile.SYMTYPE, tarfile.LNKTYPE])
@pytest.mark.parametrize("absolute", [False, True])
def test_unpack_rejects_tar_link_escape(tmp_path, permissive_tar_defaults, link_type: bytes, absolute):
    dest_dir = tmp_path / "unpacked"
    outside = tmp_path / "outside.txt"
    outside.write_bytes(b"original")
    link = tarfile.TarInfo("link")
    link.type = link_type
    link.linkname = str(outside) if absolute else "../outside.txt"
    archive = create_test_archive(tmp_path, "tar", [(link, b"")])

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, "application/x-tar", dest_dir)

    assert isinstance(error.value.__cause__, tarfile.FilterError)
    assert not (dest_dir / "link").is_symlink()
    assert not (dest_dir / "link").exists()
    assert outside.read_bytes() == b"original"
    assert archive.exists()


@pytest.mark.parametrize("archive_type", ["tar", "tar.gz"])
def test_unpack_rejects_traversal_through_archive_created_symlink(
    tmp_path, permissive_tar_defaults, archive_type
):
    dest_dir = tmp_path / "working/unpacked"
    outside = tmp_path / "outside.txt"
    outside.write_bytes(b"original")
    directory = tarfile.TarInfo("target")
    directory.type = tarfile.DIRTYPE
    link = tarfile.TarInfo("nested/link")
    link.type = tarfile.SYMTYPE
    link.linkname = "../target"
    archive = create_test_archive(tmp_path, archive_type, [
        (directory, b""),
        (link, b""),
        ("nested/link/../../../outside.txt", b"overwrite"),
    ])

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, "application/x-tar", dest_dir)

    assert isinstance(error.value.__cause__, tarfile.FilterError)
    assert (dest_dir / "nested/link").is_symlink()
    assert (dest_dir / "nested/link").resolve() == dest_dir / "target"
    assert outside.read_bytes() == b"original"
    assert archive.exists()


@pytest.mark.parametrize("member_type", [tarfile.FIFOTYPE, tarfile.CHRTYPE, tarfile.BLKTYPE])
def test_unpack_rejects_tar_special_files(tmp_path, permissive_tar_defaults, member_type: bytes):
    member = tarfile.TarInfo("special")
    member.type = member_type
    archive = create_test_archive(tmp_path, "tar", [(member, b"")])
    dest_dir = tmp_path / "unpacked"

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, "application/x-tar", dest_dir)

    assert isinstance(error.value.__cause__, tarfile.FilterError)
    assert not (dest_dir / "special").exists()
    assert archive.exists()


@pytest.mark.parametrize("link_type", [tarfile.SYMTYPE, tarfile.LNKTYPE])
def test_unpack_preserves_safe_tar_links(tmp_path, permissive_tar_defaults, link_type: bytes):
    link = tarfile.TarInfo("nested/link")
    link.type = link_type
    link.linkname = "model.bin" if link_type == tarfile.SYMTYPE else "nested/model.bin"
    archive = create_test_archive(tmp_path, "tar", [
        ("nested/model.bin", b"model"),
        (link, b""),
    ])
    dest_dir = tmp_path / "unpacked"

    assert unpack_archive_file(archive, "application/x-tar", dest_dir) == dest_dir

    assert (dest_dir / "nested/link").read_bytes() == b"model"
    assert (dest_dir / "nested/link").samefile(dest_dir / "nested/model.bin")
    assert not archive.exists()


@pytest.mark.parametrize("archive_type", ["tar", "tar.gz", "zip"])
def test_unpack_empty_archive(tmp_path, archive_type):
    archive = create_test_archive(tmp_path, archive_type, [])
    dest_dir = tmp_path / "unpacked"
    mimetype = mimetypes.guess_type(archive)[0]
    assert mimetype is not None

    assert unpack_archive_file(archive, mimetype, dest_dir) == dest_dir
    assert not archive.exists()


@pytest.mark.parametrize("archive_type", ["tar", "zip"])
def test_unpack_preserves_explicit_empty_directories(tmp_path, archive_type):
    directories = ["./", "empty/"]
    if archive_type == "tar":
        directories = [tarfile.TarInfo(name) for name in directories]
        for member in directories:
            member.type = tarfile.DIRTYPE
    archive = create_test_archive(tmp_path, archive_type, [(member, b"") for member in directories])
    dest_dir = tmp_path / "unpacked"
    mimetype = mimetypes.guess_type(archive)[0]
    assert mimetype is not None

    assert unpack_archive_file(archive, mimetype, dest_dir) == dest_dir
    assert (dest_dir / "empty").is_dir()
    assert not archive.exists()


@pytest.mark.parametrize("archive_type", ["tar", "zip"])
def test_unpack_accepts_bytes_destination(tmp_path, archive_type):
    archive = create_test_archive(tmp_path, archive_type, [("model.bin", b"model")])
    dest_dir = tmp_path / "unpacked"
    dest_bytes = os.fsencode(dest_dir)
    mimetype = mimetypes.guess_type(archive)[0]
    assert mimetype is not None

    assert unpack_archive_file(archive, mimetype, dest_bytes) == dest_bytes
    assert (dest_dir / "model.bin").read_bytes() == b"model"
    assert not archive.exists()


@pytest.mark.parametrize("archive_type", ["tar", "zip"])
def test_unpack_accepts_relative_destination_and_existing_safe_symlink(tmp_path, monkeypatch, archive_type):
    dest_dir = tmp_path / "unpacked"
    target_dir = dest_dir / "target"
    target_dir.mkdir(parents=True)
    (dest_dir / "nested").symlink_to(target_dir)
    archive = create_test_archive(tmp_path, archive_type, [("./nested/model..bin", b"model")])
    mimetype = mimetypes.guess_type(archive)[0]
    assert mimetype is not None
    monkeypatch.chdir(tmp_path)

    assert unpack_archive_file(archive.name, mimetype, "unpacked") == "unpacked"

    assert (target_dir / "model..bin").read_bytes() == b"model"
    assert not archive.exists()


def test_unpack_zip_rejects_late_symlink_escape(tmp_path, monkeypatch):
    dest_dir = tmp_path / "unpacked"
    outside_dir = tmp_path / "outside"
    outside_dir.mkdir()
    outside = outside_dir / "model.bin"
    outside.write_bytes(b"original")
    archive = create_test_archive(tmp_path, "zip", [
        ("first.bin", b"first"),
        ("nested/model.bin", b"overwrite"),
    ])
    original_extract = zipfile.ZipFile.extract

    def extract_and_create_link(self, member, path=None, pwd=None):
        result = original_extract(self, member, path, pwd)
        if member.filename == "first.bin":
            (dest_dir / "nested").symlink_to(outside_dir)
        return result

    monkeypatch.setattr(zipfile.ZipFile, "extract", extract_and_create_link)

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$"):
        unpack_archive_file(archive, "application/zip", dest_dir)

    assert (dest_dir / "first.bin").read_bytes() == b"first"
    assert outside.read_bytes() == b"original"
    assert archive.exists()


@pytest.mark.parametrize("mimetype", ["application/x-tar", "application/zip"])
def test_unpack_corrupt_archive_retains_input(tmp_path, mimetype):
    archive = tmp_path / "archive"
    archive.write_bytes(b"not an archive")

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, mimetype, tmp_path / "unpacked")

    assert isinstance(error.value.__cause__, (tarfile.TarError, zipfile.BadZipFile))
    assert archive.read_bytes() == b"not an archive"


def test_unpack_tar_fails_closed_without_data_filter(tmp_path, monkeypatch, permissive_tar_defaults):
    archive = create_test_archive(tmp_path, "tar", [("model.bin", b"model")])
    dest_dir = tmp_path / "unpacked"
    monkeypatch.delattr(tarfile, "data_filter")

    with pytest.raises(RuntimeError, match="^Failed to unpack archive file$") as error:
        unpack_archive_file(archive, "application/x-tar", dest_dir)

    assert error.value.__cause__ is not None
    assert not (dest_dir / "model.bin").exists()
    assert archive.exists()


@pytest.mark.parametrize("archive_type", ["tar.gz", "zip"])
def test_http_download_rejects_malicious_archive(tmp_path, permissive_tar_defaults, archive_type):
    archive = create_test_archive(tmp_path, archive_type, [("../outside.txt", b"overwrite")])
    outside = tmp_path / "outside.txt"
    outside.write_bytes(b"original")
    dest_dir = tmp_path / "downloaded"
    response = create_http_response(archive)
    uri = f"https://example.test/{archive.name}"
    destination = str(dest_dir)

    with patch("job.download.requests.get", return_value=response) as get:
        with pytest.raises(RuntimeError, match="^Failed to unpack archive file$"):
            download_from_http(uri, destination)

    get.assert_called_once_with(uri, stream=True)
    response.raise_for_status.assert_called_once_with()
    assert outside.read_bytes() == b"original"
    assert (dest_dir / archive.name).exists()


@pytest.mark.parametrize("archive_type", ["tar.gz", "zip"])
def test_http_download_extracts_safe_archive(tmp_path, archive_type):
    archive = create_test_archive(tmp_path, archive_type, [
        ("nested/model.bin", b"model"),
        ("config.json", b"{}"),
    ])
    dest_dir = tmp_path / "downloaded"
    response = create_http_response(archive)

    with patch("job.download.requests.get", return_value=response) as get:
        assert download_from_http(
            f"https://example.test/{archive.name}", str(dest_dir)
        ) == str(dest_dir)

    get.assert_called_once_with(f"https://example.test/{archive.name}", stream=True)
    response.raise_for_status.assert_called_once_with()
    assert (dest_dir / "nested/model.bin").read_bytes() == b"model"
    assert (dest_dir / "config.json").read_bytes() == b"{}"
    assert not (dest_dir / archive.name).exists()


def test_download_from_s3(minimal_update_artifact_env_source_dest_vars):
    """Test download_from_s3 now that it pages through prefixes."""

    # load config from your fixture
    config = get_config([])

    # sanity-check config
    assert isinstance(config.source, S3StorageConfig)
    assert config.source.bucket == "test-bucket"
    assert config.source.key == "test-key"

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
            Prefix="test-key",
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
