from dataclasses import asdict
import pytest
from unittest.mock import Mock, patch
from model_registry.utils import S3Params, OCIParams
from job.upload import _get_upload_params, _validate_credentials_path, perform_upload
from job.models import (
    AsyncUploadConfig,
    S3StorageConfig,
    OCIStorageConfig,
    ModelConfig,
    StorageConfig,
    RegistryConfig,
    UpdateArtifactIntent
)

class TestGetUploadParams:
    """Test cases for _get_upload_params function"""

    @patch("job.upload.utils._get_skopeo_backend")
    def test_get_upload_params_oci_passes_pull_args_tls_disabled(self, mock_get_skopeo_backend):
        """Test that pull_args include --src-tls-verify=false when TLS verification is disabled"""
        mock_get_skopeo_backend.return_value = Mock()

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username="test-user",
                password="test-password",
                base_image="foo-bar:latest",
                enable_tls_verify=False,
                credentials_path=None
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        _get_upload_params(config)

        mock_get_skopeo_backend.assert_called_once()
        call_kwargs = mock_get_skopeo_backend.call_args
        pull_args = call_kwargs.kwargs.get("pull_args") or call_kwargs[1].get("pull_args")
        push_args = call_kwargs.kwargs.get("push_args") or call_kwargs[1].get("push_args")
        assert "--src-tls-verify=false" in pull_args
        assert "--dest-tls-verify=false" in push_args

    @patch("job.upload._validate_credentials_path", side_effect=lambda p: p)
    @patch("job.upload.utils._get_skopeo_backend")
    def test_get_upload_params_oci_passes_pull_args_with_authfile(self, mock_get_skopeo_backend, _mock_validate):
        """Test that pull_args include --authfile when credentials_path is set"""
        mock_get_skopeo_backend.return_value = Mock()

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username="test-user",
                password="test-password",
                base_image="foo-bar:latest",
                enable_tls_verify=True,
                credentials_path="/tmp/test-creds"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        _get_upload_params(config)

        mock_get_skopeo_backend.assert_called_once()
        call_kwargs = mock_get_skopeo_backend.call_args
        pull_args = call_kwargs.kwargs.get("pull_args") or call_kwargs[1].get("pull_args")
        push_args = call_kwargs.kwargs.get("push_args") or call_kwargs[1].get("push_args")
        assert "--authfile" in pull_args
        assert "/tmp/test-creds" in pull_args
        assert "--authfile" in push_args
        assert "/tmp/test-creds" in push_args

    @patch("job.upload._validate_credentials_path", side_effect=lambda p: p)
    @patch("job.upload.utils._get_skopeo_backend")
    def test_get_upload_params_oci_passes_pull_args_tls_and_authfile(self, mock_get_skopeo_backend, _mock_validate):
        """Test that pull_args include both TLS and authfile flags when both are configured"""
        mock_get_skopeo_backend.return_value = Mock()

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username="test-user",
                password="test-password",
                base_image="foo-bar:latest",
                enable_tls_verify=False,
                credentials_path="/tmp/test-creds"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        _get_upload_params(config)

        mock_get_skopeo_backend.assert_called_once()
        call_kwargs = mock_get_skopeo_backend.call_args
        pull_args = call_kwargs.kwargs.get("pull_args") or call_kwargs[1].get("pull_args")
        push_args = call_kwargs.kwargs.get("push_args") or call_kwargs[1].get("push_args")
        assert "--src-tls-verify=false" in pull_args
        assert "--authfile" in pull_args
        assert "/tmp/test-creds" in pull_args
        assert "--dest-tls-verify=false" in push_args
        assert "--authfile" in push_args
        assert "/tmp/test-creds" in push_args

    @patch("job.upload.utils._get_skopeo_backend")
    def test_get_upload_params_oci_empty_pull_args_when_tls_enabled_no_creds(self, mock_get_skopeo_backend):
        """Test that pull_args is empty when TLS is enabled and no credentials_path"""
        mock_get_skopeo_backend.return_value = Mock()

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username="test-user",
                password="test-password",
                base_image="foo-bar:latest",
                enable_tls_verify=True,
                credentials_path=None
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        _get_upload_params(config)

        mock_get_skopeo_backend.assert_called_once()
        call_kwargs = mock_get_skopeo_backend.call_args
        pull_args = call_kwargs.kwargs.get("pull_args") or call_kwargs[1].get("pull_args")
        push_args = call_kwargs.kwargs.get("push_args") or call_kwargs[1].get("push_args")
        assert pull_args == []
        assert push_args == []

    def test_get_upload_params_s3_config(self):
        """Test _get_upload_params with S3 configuration returns S3Params"""
        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=S3StorageConfig(
                bucket="test-bucket",
                key="test-key",
                endpoint="https://s3.amazonaws.com",
                access_key_id="test-access-key",
                secret_access_key="test-secret-key",
                region="us-east-1"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="test-artifact")
            ),
            storage=StorageConfig(path="/tmp/test"),
            registry=RegistryConfig(server_address="test-server")
        )

        result = _get_upload_params(config)

        assert isinstance(result, S3Params)
        assert result.bucket_name == "test-bucket"
        assert result.s3_prefix == "test-key"
        assert result.endpoint_url == "https://s3.amazonaws.com"
        assert result.access_key_id == "test-access-key"
        assert result.secret_access_key == "test-secret-key"
        assert result.region == "us-east-1"

    @patch("job.upload._validate_credentials_path", side_effect=lambda p: p)
    def test_get_upload_params_oci_config(self, _mock_validate):
        """Test _get_upload_params with OCI configuration returns OCIParams"""
        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username="test-user",
                password="test-password",
                base_image="foo-bar:latest",
                enable_tls_verify=False,
                credentials_path="/tmp/test-creds"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        result = _get_upload_params(config)

        assert isinstance(result, OCIParams)
        assert result.base_image == "foo-bar:latest"
        assert result.oci_ref == "quay.io/example/test:latest"
        assert result.dest_dir == "/tmp/test-model"
        assert result.oci_username is None
        assert result.oci_password is None

    def test_get_upload_params_oci_preserves_inline_credentials_without_credentials_path(self):
        """Test inline OCI credentials are kept when no authfile is configured"""
        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username="test-user",
                password="test-password",
                base_image="foo-bar:latest",
                enable_tls_verify=True,
                credentials_path=None
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        result = _get_upload_params(config)

        assert isinstance(result, OCIParams)
        assert result.oci_username == "test-user"
        assert result.oci_password == "test-password"

    def test_get_upload_params_unsupported_type(self):
        """Test _get_upload_params with unsupported destination type raises ValueError"""
        # Create a mock config with an unsupported destination type
        config = Mock(spec=AsyncUploadConfig)
        config.destination = Mock()
        config.destination.__class__.__name__ = "UnsupportedStorageConfig"

        with pytest.raises(ValueError, match="Unsupported destination type"):
            _get_upload_params(config)

    def test_get_upload_params_oci_with_none_values(self):
        """Test _get_upload_params with OCI config containing None values"""
        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/test:latest",
                registry="quay.io",
                username=None,
                password=None,
                base_image="foo-bar:latest",
                enable_tls_verify=False,
                credentials_path=None
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        result = _get_upload_params(config)

        assert isinstance(result, OCIParams)
        assert result.base_image == "foo-bar:latest"
        assert result.oci_ref == "quay.io/example/test:latest"
        assert result.dest_dir == "/tmp/test-model"
        assert result.oci_username is None
        assert result.oci_password is None


class TestPerformUpload:
    """Test cases for perform_upload function"""

    @patch("job.upload._validate_credentials_path", side_effect=lambda p: p)
    @patch("job.upload.save_to_oci_registry")
    def test_perform_upload_oci(
        self, mock_save_to_oci_registry, _mock_validate
    ):
        """Test perform_upload with OCI destination"""

        mock_save_to_oci_registry.return_value = 'quay.io/example/oci/abc:def'

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/oci",
                registry="quay.io",
                username="oci_user",
                password="oci_pass",
                base_image="foo-bar:latest",
                enable_tls_verify=False,
                credentials_path="/tmp/test-creds"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        # Act
        result_uri = perform_upload(config)

        # - And the returned URI is forwarded
        assert result_uri == "quay.io/example/oci/abc:def"
        assert mock_save_to_oci_registry.call_args.kwargs["oci_username"] is None
        assert mock_save_to_oci_registry.call_args.kwargs["oci_password"] is None

    @patch("job.upload._get_upload_params")
    def test_perform_upload_propagates_exceptions_from_get_upload_params(
        self, mock_get_upload_params
    ):
        """Test perform_upload propagates exceptions from _get_upload_params"""
        # Setup
        mock_client = Mock()
        mock_get_upload_params.side_effect = ValueError("Invalid config")

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=S3StorageConfig(
                bucket="test-bucket",
                key="test-key",
                access_key_id="test-access-key",
                secret_access_key="test-secret-key",
                region="us-east-1"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="test-artifact")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        # Execute and verify exception is propagated
        with pytest.raises(ValueError, match="Invalid config"):
            perform_upload(config)

        # Verify client method was not called
        mock_client.upload_artifact_and_register_model.assert_not_called()

    @patch("job.upload._validate_credentials_path", side_effect=lambda p: p)
    @patch("job.upload.save_to_oci_registry")
    def test_perform_upload_propagates_exceptions_from_client(
        self, mock_save_to_oci_registry, _mock_validate
    ):
        """Test perform_upload propagates exceptions from client method"""
        # Setup
        mock_save_to_oci_registry.side_effect = Exception(
            "Upload failed"
        )

        config = AsyncUploadConfig(
            source=S3StorageConfig(
                bucket="source-bucket",
                key="source-key",
                access_key_id="source-key-id",
                secret_access_key="source-secret",
                region="us-west-1"
            ),
            destination=OCIStorageConfig(
                uri="quay.io/example/oci",
                registry="quay.io",
                username="oci_user",
                password="oci_pass",
                base_image="foo-bar:latest",
                enable_tls_verify=False,
                credentials_path="/tmp/test-creds"
            ),
            model=ModelConfig(
                intent=UpdateArtifactIntent(artifact_id="123")
            ),
            storage=StorageConfig(path="/tmp/test-model"),
            registry=RegistryConfig(server_address="test-server")
        )

        # Execute and verify exception is propagated
        with pytest.raises(Exception, match="Upload failed"):
            perform_upload(config)

        # Verify client method was called
        mock_save_to_oci_registry.assert_called_once()


class TestValidateCredentialsPath:
    """Test cases for _validate_credentials_path function"""

    def test_valid_absolute_regular_file(self, tmp_path):
        """Test that a valid absolute path to a regular file passes validation"""
        creds_file = tmp_path / "auth.json"
        creds_file.write_text("{}")
        result = _validate_credentials_path(str(creds_file))
        assert result == str(creds_file)

    def test_rejects_tilde_expanded_path(self, tmp_path, monkeypatch):
        """Test that a home-relative path is rejected unless already absolute"""
        home_dir = tmp_path / "home"
        home_dir.mkdir()
        creds_file = home_dir / "auth.json"
        creds_file.write_text("{}")
        monkeypatch.setenv("HOME", str(home_dir))
        with pytest.raises(ValueError, match="must be an absolute path"):
            _validate_credentials_path("~/auth.json")

    def test_accepts_symlink_to_regular_file(self, tmp_path):
        """Test that a symlink to a regular file is accepted (Kubernetes Secret mounts use symlinks)"""
        real_file = tmp_path / "real_auth.json"
        real_file.write_text("{}")
        link_file = tmp_path / "link_auth.json"
        link_file.symlink_to(real_file)
        result = _validate_credentials_path(str(link_file))
        assert result == str(real_file)

    def test_rejects_nonexistent_path(self, tmp_path):
        """Test that a nonexistent path is rejected"""
        nonexistent = tmp_path / "does_not_exist.json"
        with pytest.raises(ValueError, match="does not exist or cannot be resolved"):
            _validate_credentials_path(str(nonexistent))

    def test_rejects_directory(self, tmp_path):
        """Test that a directory path is rejected"""
        with pytest.raises(ValueError, match="must be a regular file"):
            _validate_credentials_path(str(tmp_path))

    def test_rejects_relative_path_with_traversal(self):
        """Test that a relative path with directory traversal is rejected"""
        with pytest.raises(ValueError, match="must be an absolute path"):
            _validate_credentials_path("../etc/passwd")
