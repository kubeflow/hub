from unittest.mock import Mock, patch

import pytest
from model_registry.utils import BackendDefinition

from job.models import OCIStorageConfig
from job.upload import _base_image_backend


@pytest.mark.parametrize("source", ["job-default", "bff-default", "deployment-config", "explicit"])
def test_pull_failure_identifies_reference_and_source(source, caplog):
    error = RuntimeError("registry connection refused")
    original_pull = Mock(side_effect=error)
    push = Mock()
    backend = BackendDefinition(is_available=lambda: True, pull=original_pull, push=push)
    config = OCIStorageConfig(
        uri="destination.example.com/model:1", registry="destination.example.com",
        base_image="mirror.example.com/base:1", base_image_source=source,
    )
    with patch("job.upload.utils._get_skopeo_backend", return_value=backend):
        wrapped = _base_image_backend(config, [], [])
    with caplog.at_level("INFO"), pytest.raises(RuntimeError) as caught:
        wrapped.pull(config.base_image, "/tmp/model")
    assert config.base_image in str(caught.value)
    assert source in str(caught.value)
    assert caught.value.__cause__ is error
    assert "registry connection refused" in str(caught.value)
    assert ("No base-image override" in str(caught.value)) == (source in ("job-default", "bff-default"))
    assert config.base_image in caplog.text
    assert wrapped.push is push
    push.assert_not_called()


def test_successful_pull_preserves_arguments_and_result():
    original_pull = Mock(return_value="pulled")
    backend = BackendDefinition(is_available=lambda: True, pull=original_pull, push=Mock())
    config = OCIStorageConfig(uri="destination.example.com/model:1", registry="destination.example.com")
    with patch("job.upload.utils._get_skopeo_backend", return_value=backend):
        wrapped = _base_image_backend(config, [], [])
    assert wrapped.pull(config.base_image, "/tmp/model", authfile="/tmp/auth") == "pulled"
    original_pull.assert_called_once_with(config.base_image, "/tmp/model", authfile="/tmp/auth")
