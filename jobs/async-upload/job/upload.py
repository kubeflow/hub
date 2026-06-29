from dataclasses import asdict, fields
import logging
from pathlib import Path
from typing import Any, Dict
from model_registry import utils
from model_registry.utils import OCIParams, S3Params, save_to_oci_registry

from .models import AsyncUploadConfig, DestinationConfig, OCIStorageConfig, S3StorageConfig

logger = logging.getLogger(__name__)


def _validate_credentials_path(path: str) -> str:
    """Validate that a credentials path is safe to pass to skopeo --authfile.

    Args:
        path: The credentials file path to validate.

    Returns:
        The resolved absolute path string.

    Raises:
        ValueError: If the path is not absolute, does not exist,
            or is not a regular file.
    """
    p = Path(path)
    if not p.is_absolute():
        raise ValueError(f"credentials_path must be an absolute path, got: {path}")
    try:
        resolved = p.resolve(strict=True)
    except OSError as e:
        raise ValueError(f"credentials_path does not exist or cannot be resolved: {path}") from e
    if not resolved.is_file():
        raise ValueError(f"credentials_path must be a regular file, got: {path}")
    return str(resolved)


def _get_upload_params(config: AsyncUploadConfig) -> S3Params | OCIParams:
    """
    Returns the upload params for the destination type

    Args:
        config: Configuration dictionary
    """
    destination_config = config.destination
    logger.debug("🔍 Getting upload params for destination type: %s", destination_config)
    if isinstance(config.destination, S3StorageConfig):
        return S3Params(
            bucket_name=config.destination.bucket,
            s3_prefix=config.destination.key,
            endpoint_url=config.destination.endpoint,
            access_key_id=config.destination.access_key_id,
            secret_access_key=config.destination.secret_access_key,
            region=config.destination.region,
        )
    elif isinstance(destination_config, OCIStorageConfig):
        push_args = []
        pull_args = []
        username = destination_config.username
        password = destination_config.password
        # Note: These are all skopeo args, see: https://github.com/containers/skopeo/blob/main/docs/skopeo-copy.1.md
        if not destination_config.enable_tls_verify:
            push_args.append("--dest-tls-verify=false")
            pull_args.append("--src-tls-verify=false")
        if destination_config.credentials_path:
            validated_path = _validate_credentials_path(destination_config.credentials_path)
            push_args.append("--authfile")
            push_args.append(validated_path)
            pull_args.append("--authfile")
            pull_args.append(validated_path)
            # Avoid synthesizing a second authfile in save_to_oci_registry().
            username = None
            password = None

        return OCIParams(
            base_image=destination_config.base_image,
            oci_ref=destination_config.uri,
            dest_dir=config.storage.path,
            oci_username=username,
            oci_password=password,
            # Same as the default backend, but with additional args included
            custom_oci_backend=utils._get_skopeo_backend(
                pull_args=pull_args,
                push_args=push_args
            ),
        )
    else:
        raise ValueError(f"Unsupported destination type")


def perform_upload(config: AsyncUploadConfig) -> str:
    """
    Performs the upload of the model to the destination with KServe Modelcars compatibility

    Returns:
        The URI of the uploaded model
    """
    model_files_path = config.storage.path

    upload_params = _get_upload_params(config)
    logger.debug("🔍 Upload params: %s", upload_params)

    logger.info("📤 Uploading model to destination...")
    if isinstance(upload_params, S3Params):
        raise ValueError("S3 upload destination is not supported")
    elif isinstance(upload_params, OCIParams):
        uri = save_to_oci_registry(
            **{field.name: getattr(upload_params, field.name) for field in fields(upload_params)},
            model_files_path=model_files_path
        )
    else:
        raise ValueError("Unsupported destination type")

    logger.info("✅ Model uploaded to destination: %s", uri)
    return uri
