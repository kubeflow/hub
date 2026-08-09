import logging
import mimetypes
import os
import shutil
import tarfile
from typing import Union
import zipfile
from urllib.parse import urlparse

import requests
from model_registry.utils import _connect_to_s3

from .models import AsyncUploadConfig, OCIStorageConfig, S3StorageConfig, URISourceConfig

logger = logging.getLogger(__name__)

PathType = Union[str, bytes, os.PathLike]

HF_URI_PREFIX = "hf://"
HTTP_URI_PREFIXES = ("http://", "https://")
HEADERS_SUFFIX = "-headers"
ZIP_CONTENT_TYPES = (
    "application/x-zip-compressed",
    "application/zip",
    "application/zip-compressed",
)
TAR_CONTENT_TYPES = (
    "application/x-tar",
    "application/x-gtar",
    "application/x-gzip",
    "application/gzip",
)
REGULAR_FILE_CONTENT_TYPES = ("application/octet-stream",)


def download_from_s3(config: S3StorageConfig, storage_path: str):
    logger.debug("🔍 Downloading model from S3...")
    logger.debug("🔍 Source config: %s", config)
    s3_client, _ = _connect_to_s3(
        config.endpoint,
        config.access_key_id,
        config.secret_access_key,
        config.region,
        multipart_threshold=1024 * 1024,
        multipart_chunksize=1024 * 1024,
        max_pool_connections=10,
    )

    bucket_name = config.bucket
    prefix = config.key if config.key.endswith("/") else f"{config.key}/"

    # TODO: It might make sense to check if the provided key points to a single file first before assuming the directory needs to be traversed

    paginator = s3_client.get_paginator("list_objects_v2")
    for page in paginator.paginate(Bucket=bucket_name, Prefix=prefix):
        for obj in page.get("Contents", []):
            key = obj["Key"]
            if key.endswith("/"):
                continue
            relative = os.path.relpath(key, prefix)
            local_path = os.path.join(storage_path, relative)
            os.makedirs(os.path.dirname(local_path), exist_ok=True)
            logger.info(f"⏳ Downloading s3://{bucket_name}/{key} → {local_path}")
            s3_client.download_file(bucket_name, key, local_path)
            logger.info(f"☑️ Downloaded s3://{bucket_name}/{key} → {local_path}")
    logger.debug("✅ Model files downloaded from S3")


def download_from_hf(uri: str, dest_dir: str) -> str:
    """
    adapted from kserve:
    https://github.com/kserve/kserve/blob/4edbb36c520c2e880842229bfc56b7f11d766822/python/storage/kserve_storage/kserve_storage.py#L292-L322
    """
    from huggingface_hub import snapshot_download

    if not uri.startswith(HF_URI_PREFIX):
        raise ValueError(f"Expected URI beginning with {HF_URI_PREFIX}")

    components = uri[len(HF_URI_PREFIX) :].split("/")
    if len(components) != 2:
        raise ValueError(
            "URI must contain exactly one '/' separating the repo and model name"
        )

    repo, model_id = components
    if not repo:
        raise ValueError("Repository name cannot be empty")

    model_name, _, hash_value = model_id.partition(":")
    if not model_name:
        raise ValueError("Model name cannot be empty")

    revision = hash_value if hash_value else None
    return snapshot_download(
        repo_id=f"{repo}/{model_name}", revision=revision, local_dir=dest_dir
    )


def download_from_http(uri: str, dest_dir: str) -> str:
    """
    adapted from kserve:
    https://github.com/kserve/kserve/blob/4edbb36c520c2e880842229bfc56b7f11d766822/python/storage/kserve_storage/kserve_storage.py#L698-L771
    """
    url = urlparse(uri)
    filename = os.path.basename(url.path)
    # Determine if the symbol '?' exists in the path
    if mimetypes.guess_type(url.path)[0] is None and url.query != "":
        mimetype, encoding = mimetypes.guess_type(url.query)
    else:
        mimetype, encoding = mimetypes.guess_type(url.path)

    if filename == "":
        raise ValueError(f"No filename contained in URI: {uri}")

    is_archive = True
    match mimetype:
        case "application/zip":
            valid_content_types = ZIP_CONTENT_TYPES
        case "application/x-tar":
            valid_content_types = TAR_CONTENT_TYPES
        case _:
            valid_content_types = REGULAR_FILE_CONTENT_TYPES
            is_archive = False

    # Use body content workflow to defer body download until response.raw is called
    # https://requests.readthedocs.io/en/latest/user/advanced/#body-content-workflow
    with requests.get(uri, stream=True) as response:
        response.raise_for_status()

        if not response.headers.get("Content-Type", "").startswith(valid_content_types):
            content_types_str = ", ".join(valid_content_types)
            logger.warning(
                f"URI {uri} appears to have MIME type {mimetype} but did not respond with any of following for 'Content-Type': {content_types_str}"
            )

        dest_path = os.path.join(dest_dir, filename)
        os.makedirs(dest_dir, exist_ok=True)
        with open(dest_path, "wb") as dest_file:
            shutil.copyfileobj(response.raw, dest_file)

    if is_archive:
        dest_dir = unpack_archive_file(dest_path, mimetype, dest_dir)
    return dest_dir


def unpack_archive_file(file_path: PathType, mimetype: str, dest_dir: PathType) -> str:
    """
    adapted from kserve:
    https://github.com/kserve/kserve/blob/4edbb36c520c2e880842229bfc56b7f11d766822/python/storage/kserve_storage/kserve_storage.py#L773-L792
    """
    logger.info("Unpacking archive: %s", file_path)
    try:
        if mimetype == "application/x-tar":
            archive = tarfile.open(file_path, "r", encoding="utf-8")
        else:
            archive = zipfile.ZipFile(file_path, "r")
        with archive:
            archive.extractall(dest_dir)
    except (tarfile.TarError, zipfile.BadZipfile) as e:
        raise RuntimeError("Failed to unpack archive file") from e
    os.remove(file_path)
    return dest_dir


def perform_download(config: AsyncUploadConfig):
    logger.info("📥 Downloading model from source...")
    # Download the model from the defined source
    if isinstance(config.source, S3StorageConfig):
        download_from_s3(config.source, config.storage.path)
    elif isinstance(config.source, URISourceConfig):
        uri = config.source.uri
        if uri.startswith(HTTP_URI_PREFIXES):
            download_from_http(uri, config.storage.path)
        elif uri.startswith(HF_URI_PREFIX):
            download_from_hf(uri, config.storage.path)
        else:
            raise ValueError(f"Unsupported URI format: {uri}")
    elif isinstance(config.source, OCIStorageConfig):
        # TODO: Implement the OCI download logic here
        raise ValueError("OCI source is not supported yet")
    else:
        raise ValueError(f"Unsupported source type: {type(config.source)}")
    logger.info("✅ Model downloaded from source")
