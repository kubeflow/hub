"""Utilities for the model registry."""

from __future__ import annotations

import base64
import json
import os
import shutil
import tempfile
from collections.abc import Callable, Generator
from contextlib import contextmanager, suppress
from dataclasses import asdict, dataclass
from pathlib import Path
from subprocess import CalledProcessError, CompletedProcess
from typing import TYPE_CHECKING, Generic, Literal, Protocol, TextIO, TypeVar, cast

from typing_extensions import overload

from ._utils import required_args
from .exceptions import MissingMetadata, StoreError

# Generic return types
T = TypeVar("T")
TPull = TypeVar("TPull", covariant=True)
TPush = TypeVar("TPush", covariant=True)

# If we want to forward reference
if TYPE_CHECKING:
    from boto3.s3.transfer import TransferConfig
    from botocore.client import BaseClient


@overload
def s3_uri_from(
    path: str,
) -> str: ...


@overload
def s3_uri_from(
    path: str,
    bucket: str,
) -> str: ...


@overload
def s3_uri_from(
    path: str,
    bucket: str,
    *,
    endpoint: str,
    region: str,
) -> str: ...


@required_args(
    (),
    (  # pre-configured env
        "bucket",
    ),
    (  # custom env or non-default bucket
        "bucket",
        "endpoint",
        "region",
    ),
)
def s3_uri_from(
    path: str,
    bucket: str | None = None,
    *,
    endpoint: str | None = None,
    region: str | None = None,
) -> str:
    """Build an S3 URI.

    This helper function builds an S3 URI from a path and a bucket name, assuming you have a configured environment
    with a default bucket, endpoint, and region set.
    If you don't, you must provide all three optional arguments.
    That is also the case for custom environments, where the default bucket is not set, or if you want to use a
    different bucket.

    Args:
        path: Storage path.
        bucket: Name of the S3 bucket. Defaults to AWS_S3_BUCKET.
        endpoint: Endpoint of the S3 bucket. Defaults to AWS_S3_ENDPOINT.
        region: Region of the S3 bucket. Defaults to AWS_DEFAULT_REGION.

    Returns:
        S3 URI.
    """
    default_bucket = os.environ.get("AWS_S3_BUCKET")
    if not bucket:
        if not default_bucket:
            msg = "Custom environment requires all arguments"
            raise MissingMetadata(msg)
        bucket = default_bucket
    elif (not default_bucket or default_bucket != bucket) and not endpoint:
        msg = "endpoint must be provided for non-default bucket"
        raise MissingMetadata(msg)

    endpoint = endpoint or os.getenv("AWS_S3_ENDPOINT")
    region = region or os.getenv("AWS_DEFAULT_REGION")
    if not endpoint:
        msg = "Missing environment variable: `bucket` is required"
        raise MissingMetadata(msg)

    # https://alexwlchan.net/2020/s3-keys-are-not-file-paths/ nor do they resolve to valid URls
    # FIXME: is this safe?
    if not region:
        return f"s3://{bucket}/{path}?endpoint={endpoint}"
    return f"s3://{bucket}/{path}?endpoint={endpoint}&defaultRegion={region}"


class PullFn(Protocol[TPull]):
    """Pull function definition."""

    def __call__(self, base: str, dest: Path, **kwargs) -> TPull: ...  # noqa: D102


class PushFn(Protocol[TPush]):
    """Push function definition."""

    def __call__(self, src: Path, oci_ref: str, **kwargs) -> TPush: ...  # noqa: D102


@dataclass
class BackendDefinition(Generic[TPull, TPush]):
    """Holds the 3 core callables for a backend.

    - is_available() -> bool
    - pull(base_image: str, dest_dir: Path, **kwargs) -> TPull
    - push(local_image_path: Path, oci_ref: str, **kwargs) -> TPush.
    """

    is_available: Callable[[], bool]
    pull: PullFn[TPull]
    push: PushFn[TPush]


def _kwargs_to_params(kwargs: dict[str, str]) -> list[str]:
    """Convert kwargs to list of params.

    Args:
        kwargs: The keyword args dict.
    """
    args = []
    for k, v in kwargs.items():
        args.append(f"{k}")
        args.append(str(v))
    return args


def _get_skopeo_backend(
    pull_args: list[str] | None = None, push_args: list[str] | None = None
) -> BackendDefinition[CompletedProcess[bytes], CompletedProcess[bytes]]:
    try:
        from olot.backend.skopeo import is_skopeo, skopeo_pull, skopeo_push
    except ImportError as e:
        msg = "Could not import 'olot.backend.skopeo'. Ensure that 'olot' is installed if you want to use the 'skopeo' backend."
        raise ImportError(msg) from e

    def wrapped_pull(base_image: str, dest: Path, **kwargs) -> CompletedProcess[bytes]:
        kwargs = _backend_specific_params("skopeo", "pull", **kwargs)
        params = _kwargs_to_params(kwargs)
        params.extend(pull_args or [])

        return _scrub_errors(lambda: skopeo_pull(base_image, dest, params))

    def wrapped_push(src: Path, oci_ref: str, **kwargs) -> CompletedProcess[bytes]:
        kwargs = _backend_specific_params("skopeo", "push", **kwargs)
        params = _kwargs_to_params(kwargs)
        params.extend(push_args or [])

        return _scrub_errors(lambda: skopeo_push(src, oci_ref, params))

    return BackendDefinition(
        is_available=is_skopeo, pull=cast(PullFn[CompletedProcess[bytes]], wrapped_pull), push=wrapped_push
    )


def _get_oras_backend(
    pull_args: list[str] | None = None, push_args: list[str] | None = None
) -> BackendDefinition[CompletedProcess[bytes], CompletedProcess[bytes]]:
    try:
        from olot.backend.oras_cp import is_oras, oras_pull, oras_push
    except ImportError as e:
        msg = "Could not import 'olot.backend.oras_cp'. Ensure that 'olot' is installed if you want to use the 'oras_cp' backend."
        raise ImportError(msg) from e

    def wrapped_pull(base_image: str, dest: Path, **kwargs) -> CompletedProcess[bytes]:
        kwargs = _backend_specific_params("oras", "pull", **kwargs)
        params = _kwargs_to_params(kwargs)
        params.extend(pull_args or [])

        return _scrub_errors(lambda: oras_pull(base_image, dest, params))

    def wrapped_push(src: Path, oci_ref: str, **kwargs) -> CompletedProcess[bytes]:
        kwargs = _backend_specific_params("oras", "push", **kwargs)
        params = _kwargs_to_params(kwargs)
        params.extend(push_args or [])

        return _scrub_errors(lambda: oras_push(src, oci_ref, params))

    return BackendDefinition(
        is_available=is_oras,
        pull=cast(PullFn[CompletedProcess[bytes]], wrapped_pull),
        push=wrapped_push,
    )


def _backend_specific_params(backend: Literal["skopeo", "oras"], type: Literal["push", "pull"], **kwargs) -> dict:
    """Generate params based on the backend and action.

    Args:
        backend: The backend to use supported in olot.
        type: The action to perform.
        kwargs: Additional args defined below.

    Keyword Args:
        username: the usrername of the registry.
        password: the password of the registry.

    See documention:
      skopeo-copy(1): https://github.com/containers/skopeo/blob/main/docs/skopeo-copy.1.md
      oras cp: https://oras.land/docs/commands/oras_cp

    Example resulting commands:
      skopeo copy --src-authfile /tmp/tmp-123.json
      skopeo copy --dest-authfile /tmp/tmp-123.json
      oras cp --from-registry-config /tmp/tmp-123.json
      oras cp --to-registry-config /tmp/tmp-123.json
    """
    # Determine backend
    if backend == "skopeo":
        prefix = "--src" if type == "pull" else "--dest"
        auth_suffix = "authfile"
    elif backend == "oras":
        prefix = "--from" if type == "pull" else "--to"
        auth_suffix = "registry-config"
    else:
        # This is defensive code - the Literal type hint makes this unreachable
        # but we keep it for runtime safety if the type hint is bypassed
        msg = f"invalid backend: {backend!r}"
        raise ValueError(msg)

    # Actual param specifications
    if authfile := kwargs.pop("authfile", None):
        kwargs[f"{prefix}-{auth_suffix}"] = authfile

    return kwargs


def _scrub_errors(func: Callable[[], CompletedProcess[bytes]]) -> CompletedProcess[bytes]:
    """Scrub errors of any subprocess command with sensitive data.

    Args:
        func: A partial or lambda function that has not been yet executed.
    """
    try:
        return func()
    except (CalledProcessError, Exception) as e:
        msg = """Problem with command"""
        raise RuntimeError(msg, e.returncode, e.stderr) from None  # type: ignore[attr-defined]


@dataclass
class OCIParams:
    """Parameters for the OCI client to perform the upload.

    Allows for some customization of how to perform the upload step when uploading via OCI
    """

    base_image: str
    oci_ref: str
    dest_dir: str | os.PathLike | None = None
    backend: str = "skopeo"
    modelcard: os.PathLike | None = None
    custom_oci_backend: BackendDefinition[CompletedProcess[bytes], CompletedProcess[bytes]] | None = None
    oci_auth_env_var: str | None = None
    oci_username: str | None = None
    oci_password: str | None = None


@dataclass
class S3Params:
    """Parameters for the S3 Client (boto3) to perform the upload.

    Allows for some amount of customization when performing an upload, such as providing a custom endpoint url, access keys, etc.
    """

    bucket_name: str
    s3_prefix: str
    endpoint_url: str | None = None
    access_key_id: str | None = None
    secret_access_key: str | None = None
    region: str | None = None
    multipart_threshold: int = 1024 * 1024
    multipart_chunksize: int = 1024 * 1024
    max_pool_connections: int = 10


# A dict mapping backend names to their definitions
BackendDict = dict[str, Callable[[], BackendDefinition[CompletedProcess[bytes], CompletedProcess[bytes]]]]

DEFAULT_BACKENDS: BackendDict = {
    "skopeo": _get_skopeo_backend,
    "oras": _get_oras_backend,
}


def save_to_oci_registry(  # noqa: C901 ( complex args >8 )
    base_image: str,
    oci_ref: str,
    model_files_path: str | os.PathLike,
    dest_dir: str | os.PathLike | None = None,
    backend: str = "skopeo",
    modelcard: os.PathLike | None = None,
    custom_oci_backend: BackendDefinition[CompletedProcess[bytes], CompletedProcess[bytes]] | None = None,
    oci_auth_env_var: str | None = None,
    oci_username: str | None = None,
    oci_password: str | None = None,
) -> str:
    """Appends a list of files to an OCI-based image.

    Args:
        base_image: The image to append model files to. This image will be downloaded to the location at `dest_dir`
        dest_dir: The location to save the downloaded and extracted base image to.
        oci_ref: Destination of where to push the newly layered image to. eg, "quay.io/my-org/my-registry:1.0.0"
        model_files_path: Path to the files to add to the base_image as layers
        backend: The CLI tool to use to perform the oci image pull/push. One of: "skopeo", "oras"
        modelcard: [Optional] Path to the modelcard to additionally include as a layer
        custom_oci_backend: [Optional] If you would like to use your own OCI Backend layer, you can provide it here
        oci_auth_env_var: [Optional] The environment variable that holds the auth/config JSON for OCI registry auth.
        oci_username: [Optional] The username to the OCI registry.
        oci_password: [Optional] (Must be used with OCI username) The password to the OCI registry.

    Raises:
        ValueError: If the chosen backend is not installed on the host
        ValueError: If the chosen backend is an invalid option
        StoreError: If `olot` is not installed as a python package
    Returns:
        uri: The OCI URI of the uploaded model.
    """
    try:
        from olot.basics import oci_layers_on_top
    except ImportError as e:
        msg = """Package `olot` is not installed.
To save models to OCI compatible storage, start by installing the `olot` package, either directly or as an
extra (available as `model-registry[olot]`), e.g.:
```sh
!pip install --pre model-registry[olot]
```
or
```sh
!pip install olot
```
        """
        raise StoreError(msg) from e

    # Check for OCI Auth Env and a default
    auth: str | None = None
    if oci_auth_env_var:
        auth = _validate_env_var(oci_auth_env_var)
    elif ".dockerconfigjson" in os.environ:
        auth = os.environ[".dockerconfigjson"]  # noqa: SIM112

    elif oci_username and oci_password:
        auth = json.dumps(create_auth_object(oci_ref, oci_username, oci_password))

    # If a custom backend is provided, use it, else fetch the backend out of the registry
    if custom_oci_backend:
        backend_def = custom_oci_backend
    elif backend in DEFAULT_BACKENDS:
        # Fetching the backend definition can throw an error, but it should bubble up as it has the appropriate messaging
        backend_def = DEFAULT_BACKENDS[backend]()
    else:
        msg = f"'{backend}' is not an available backend to use. Available backends: {DEFAULT_BACKENDS.keys()}"
        raise ValueError(msg)
    if not backend_def.is_available():
        msg = f"Backend '{backend}' is selected, but not available on the system. Ensure the dependencies for '{backend}' are installed in your environment."
        raise ValueError(msg)

    dest_dir_cleanup = False
    if dest_dir is None:
        dest_dir = tempfile.mkdtemp()
        dest_dir_cleanup = True
    local_image_path = Path(dest_dir)

    params = {}
    with temp_auth_file(auth) as auth_file:
        if auth_file is not None:
            params["authfile"] = auth_file.name
        backend_def.pull(base_image, local_image_path, **params)
        # Extract the absolute path from the files found in the path
        files = [file[0] for file in _get_files_from_path(model_files_path)]  # type: ignore[arg-type]
        oci_layers_on_top(local_image_path, files, modelcard, root_dir=model_files_path)
        backend_def.push(local_image_path, oci_ref, **params)

    # Return the OCI URI
    if dest_dir_cleanup:
        shutil.rmtree(dest_dir)
    return f"oci://{oci_ref}"


@overload
@contextmanager
def temp_auth_file(auth: str) -> Generator[TextIO, None, None]: ...


@overload
@contextmanager
def temp_auth_file(auth: None) -> Generator[None, None, None]: ...


@contextmanager
def temp_auth_file(auth: str | None) -> Generator[TextIO | None, None, None]:
    """Create a temporary auth file with optional auth data.

    If auth is None, yields None. Otherwise creates a temporary JSON file
    containing the auth string and yields the file handle.
    """
    if auth is None:
        yield None
    else:
        # Yield the file after closing so it can be read by another process,
        # and use delete=False so closing doesn't delete the file. We add our
        # own cleanup code to achieve an effect similar to using
        # delete=True, delete_on_close=False with Python 3.12 or later.
        path: str | None = None
        try:
            with tempfile.NamedTemporaryFile(
                mode="w+", encoding="utf-8", suffix=".json", delete=False
            ) as temp_auth_file:
                path = temp_auth_file.name
                temp_auth_file.write(auth)
            yield temp_auth_file  # type: ignore[misc]
        finally:
            if path is not None:
                with suppress(OSError):
                    os.unlink(path)


def _s3_creds(
    endpoint_url: str | None = None,
    access_key_id: str | None = None,
    secret_access_key: str | None = None,
    region: str | None = None,
):
    """Internal method to return mix and matched S3 credentials based on presence.

    Args:
        endpoint_url: The S3 compatible object storage endpoint.
        access_key_id: The S3 compatible object storage access key ID.
        secret_access_key: The S3 compatible object storage secret access key.
        region: The region name for the S3 object storage.

    Raises:
        ValueError if the required values are None.

    Returns:
        tuple(endpoint, access_key_id, secret_access_key, region)
    """
    aws_s3_endpoint = os.getenv("AWS_S3_ENDPOINT")
    aws_access_key_id = os.getenv("AWS_ACCESS_KEY_ID")
    aws_secret_access_key = os.getenv("AWS_SECRET_ACCESS_KEY")
    aws_default_region = os.getenv("AWS_DEFAULT_REGION")

    # Set values to parameter values or environment values
    endpoint_url = endpoint_url or aws_s3_endpoint
    access_key_id = access_key_id or aws_access_key_id
    secret_access_key = secret_access_key or aws_secret_access_key
    region = region or aws_default_region

    if not any((aws_s3_endpoint, endpoint_url)):
        msg = """Please set either `AWS_S3_ENDPOINT` as environment variable
            or specify `endpoint_url` as the parameter.
            """
        raise ValueError(msg)

    if not (access_key_id and secret_access_key):
        msg = """Environment variables `AWS_ACCESS_KEY_ID` or `AWS_SECRET_ACCESS_KEY` were not set.
            Please either set these environment variables or pass them in as parameters using
            `access_key_id` or `secret_access_key`.
            """
        raise ValueError(msg)

    return endpoint_url, access_key_id, secret_access_key, region


def _upload_to_s3(  # noqa: C901
    path: str,
    bucket: str,
    s3: BaseClient,
    path_prefix: str,
    *,
    endpoint_url: str | None = None,
    region: str | None = None,
    transfer_config: TransferConfig | None = None,
) -> str:
    """Internal method for recursively uploading all files to S3.

    Args:
        path: The path to where the models or artifacts are.
        bucket: The name of the S3 bucket.
        s3: The S3 Client object.
        path_prefix: The folder prefix to store under the root of the bucket.
        transfer_config: The transfer config to use for the upload.

    Keyword Args:
        endpoint_url: The endpoint url for the S3 bucket.
        region: The region name for the S3 bucket.

    Returns:  The S3 URI path of the uploaded files.

    Raises:
        StoreError if `path` does not exist.
        StoreError if `path_prefix` is not set.
    """
    path_prefix = path_prefix.rstrip("/")

    uri = s3_uri_from(
        path=path_prefix,
        bucket=bucket,
        endpoint=endpoint_url,  # type: ignore[arg-type]
        region=region,  # type: ignore[arg-type]
    )
    files = _get_files_from_path(path)
    for absolute_path_filename, relative_path_filename in files:
        s3_key = os.path.join(path_prefix, relative_path_filename)
        s3.upload_file(
            Filename=absolute_path_filename,
            Bucket=bucket,
            Key=s3_key,
            Config=transfer_config,
        )

    return uri


def _connect_to_s3(
    endpoint_url: str | None = None,
    access_key_id: str | None = None,
    secret_access_key: str | None = None,
    region: str | None = None,
    multipart_threshold: int | None = None,
    multipart_chunksize: int | None = None,
    max_pool_connections: int | None = None,
) -> tuple[BaseClient, TransferConfig]:
    """Internal method to connect to Boto3 Client.

    Args:
        endpoint_url: The S3 compatible object storage endpoint.
        access_key_id: The S3 compatible object storage access key ID.
        secret_access_key: The S3 compatible object storage secret access key.
        region: The region name for the S3 object storage.
        multipart_threshold: The threshold for multipart uploads.
        multipart_chunksize: The size of chunks for multipart uploads.
        max_pool_connections: The maximum number of connections in the pool.

    Returns:
        tuple(client, config): A tuple of the Boto3 client and the TransferConfig.

    Raises:
        StoreError: If Boto3 is not installed.
        ValueError: If the appropriate values are not supplied.
    """
    try:
        from boto3 import client
        from boto3.s3.transfer import TransferConfig
        from botocore.config import Config

    except ImportError as e:
        msg = """package `boto3` is not installed.
            To save models to an S3 compatible storage, start by installing the `boto3` package, either directly or as an
            extra (available as `model-registry[boto3]`), e.g.:
            ```sh
            !pip install --pre model-registry[boto3]
            ```            or
            ```sh
            !pip install boto3
            ```            """
        raise StoreError(msg) from e

    config = Config(
        max_pool_connections=max_pool_connections,
    )
    transfer_config = TransferConfig(
        multipart_threshold=multipart_threshold,
        multipart_chunksize=multipart_chunksize,
    )
    return client(
        "s3",
        endpoint_url=endpoint_url,
        aws_access_key_id=access_key_id,
        aws_secret_access_key=secret_access_key,
        region_name=region,
        config=config,
    ), transfer_config


def _get_files_from_path(path: str) -> list[tuple[str, str]]:
    """Given a path, get the list of files.

    If the path points to a single file, that file's absolute_path and filename will be the only entry returned

    If the path points to a directory, the directory will be walked to fetch all the absolute and relative filepaths for each file

    Args:
        path: Location (directory or file) to extract filenames from

    Returns:
        A list of 2-entry tuples containing (absolute_path, relative_path) from the path provided

    Raises:
        ValueError: If the path provided does not already exist.
    """
    if not os.path.exists(path):
        msg = f"Path '{path}' does not exist. Please ensure path is correct."
        raise ValueError(msg)

    files = []

    is_file = os.path.isfile(path)
    if is_file:
        # When just a single file, return it
        filename = os.path.basename(path)
        file = (path, filename)
        files.append(file)
        return files

    for root, _, filenames in os.walk(path):
        for filename in filenames:
            absolute_path = os.path.join(root, filename)
            relative_path = os.path.relpath(absolute_path, path)
            files.append((absolute_path, relative_path))

    return files


def _validate_env_var(var: str) -> str:
    """Validate that an env var exists.

    Args:
        var: The env var to lookup.
    """
    if not (env_var := os.getenv(var)):
        msg = f"Cannot find environment variable '{var}'."
        raise ValueError(msg)
    return env_var


def _extract_auth_json(auth_data: str) -> str:
    """Extract the auth JSON from a string value.

    Args:
        auth_data: The Auth JSON string.
    """
    try:
        auth_json = json.loads(auth_data)
        if type(auth_json) is not dict:
            msg = ""
            raise TypeError(msg)
        registries = auth_json["auths"]
        reg_keys = list(registries.keys())
        if len(reg_keys) > 1:
            msg = f"Auth JSON has multiple registries ({', '.join(reg_keys)}). This is not supported."
            raise ValueError(msg)

        key = registries[reg_keys[0]]["auth"]
        auth = base64.b64decode(key)
        return auth.decode()

    except (AttributeError, KeyError) as e:
        msg = "This is an invalid Auth JSON."
        raise ValueError(msg) from e
    except json.JSONDecodeError as e:
        invalid_json_msg = "Auth data does not contain valid JSON."
        raise ValueError(invalid_json_msg) from e


def get_auth_reference(image_path: str) -> str:
    """Parses an arbitrary container image path to extract a valid reference.

    for use as a key in a container registry auth.json file.

    Examples:
        'quay.io/my-org/my-registry:1.0.0' -> 'quay.io/my-org/my-registry'
        'my-private-registry:5000/team/app:latest' -> 'my-private-registry:5000/team/app'
        'ubuntu' -> 'docker.io'
        'ubuntu:22.04' -> 'docker.io'
        'my-user/my-app' -> 'docker.io'
        'my-user/my-app:v2' -> 'docker.io'
        'quay.io/my-org/my-registry@sha256:f1b3f5a2d...' -> 'quay.io/my-org/my-registry'
        'localhost:5000/my-local-image' -> 'localhost:5000/my-local-image'
        'localhost:5000/my-local-image:test-tag' -> 'localhost:5000/my-local-image'
    """
    repo_path = image_path

    # Remove digest if it exists
    if "@" in repo_path:
        repo_path = repo_path.split("@", 1)[0]

    # Separate the tag from the repository path.
    # The tag is what comes after the last colon, but only if that colon
    # is not part of the hostname/port. A colon indicates a tag if it
    # appears after the last slash in the path.
    last_colon = repo_path.rfind(":")
    last_slash = repo_path.rfind("/")

    if last_colon > last_slash:
        # This is a tag, not a port, so we strip it.
        repo_path = repo_path[:last_colon]

    # Handle default Docker Hub images (e.g., 'ubuntu', 'user/repo').
    # The hostname is the part of the path before the first slash.
    first_slash_index = repo_path.find("/")
    hostname = repo_path
    if first_slash_index != -1:
        hostname = repo_path[:first_slash_index]

    # If the hostname part doesn't contain a '.' (like quay.io) or a ':' (like localhost:5000),
    # it's a short name for an image on Docker Hub.
    if "." not in hostname and ":" not in hostname:
        return "docker.io"

    # For all other images, the full repository path is the reference.
    return repo_path


def create_auth_object(oci_ref: str, username: str, password: str) -> dict[str, dict[str, dict[str, str]]]:
    """Create an auth object for container registry authentication.

    This object can be encoded as json with json.dumps() producing the
    contents for valid authfile.
    """
    auth_ref = get_auth_reference(oci_ref)
    auth_value = base64.b64encode(f"{username}:{password}".encode()).decode("utf-8")
    return {"auths": {auth_ref: {"auth": auth_value}}}


def upload_to_s3(
    s3_auth: S3Params,
    path: str,
    s3_client: BaseClient | None = None,
    transfer_config: TransferConfig | None = None,
) -> str:
    """Upload to S3.

    Args:
        s3_auth: The S3 authentication parameters.
        path: The path to the file or directory to upload.
        s3_client: The S3 client to use. If not provided, a new client will be created.
        transfer_config: The transfer config to use for the upload. If not provided, a new transfer config will be created.
    """
    if s3_client and not transfer_config:
        msg = "Both `transfer_config` and `s3_client` must be provided if S3 is provided."
        raise ValueError(msg)

    if not s3_client:
        s3_auth_dict = asdict(s3_auth)
        s3_auth_dict.pop("bucket_name")
        s3_auth_dict.pop("s3_prefix")
        s3, transfer_config = _connect_to_s3(**s3_auth_dict)
    else:
        s3 = s3_client

    return _upload_to_s3(
        path=path,
        bucket=s3_auth.bucket_name,
        s3=s3,
        path_prefix=s3_auth.s3_prefix,
        transfer_config=transfer_config,
    )
