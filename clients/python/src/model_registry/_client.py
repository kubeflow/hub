"""Standard client for the model registry."""

from __future__ import annotations

import contextlib
import inspect
import logging
import os
from collections.abc import Callable, Coroutine, Mapping
from dataclasses import asdict
from pathlib import Path
from typing import (
    Any,
    TypeVar,
    get_args,
    overload,
)
from warnings import warn

from model_registry.types.base import BaseResourceModel

from .core import ModelRegistryAPIClient
from .exceptions import StoreError
from .types import (
    ListOptions,
    ModelArtifact,
    ModelVersion,
    Pager,
    RegisteredModel,
    SupportedTypes,
)
from .utils import (
    OCIParams,
    S3Params,
    _connect_to_s3,
    _s3_creds,
    _upload_to_s3,
    save_to_oci_registry,
)

TModel = TypeVar("TModel", bound=BaseResourceModel)
T = TypeVar("T")

logging.basicConfig(
    format="%(asctime)s.%(msecs)03d - %(name)s:%(levelname)s: %(message)s",
    datefmt="%H:%M:%S",
    level=logging.WARNING,  # the default loglevel
    handlers=[
        # logging.FileHandler(
        #     LOGS
        #     / "log-{}-{}.log".format(
        #         datetime.now(tz=datetime.now().astimezone().tzinfo).strftime(
        #             "%Y-%m-%d-%H-%M-%S"
        #         ),
        #         os.getpid(),
        #     ),
        #     encoding="utf-8",
        #     delay=False,
        # ),
        logging.StreamHandler(),
    ],
)

logger = logging.getLogger("model-registry")

DEFAULT_USER_TOKEN_ENVVAR = "KF_PIPELINES_SA_TOKEN_PATH"  # noqa: S105
DEFAULT_K8S_SA_TOKEN_PATH = "/var/run/secrets/kubernetes.io/serviceaccount/token"  # noqa: S105


class ModelRegistry:
    """Model registry client."""

    def __init__(  # noqa: C901
        self,
        server_address: str,
        port: int | None = None,
        *,
        author: str,
        is_secure: bool | None = None,
        user_token: str | None = None,
        user_token_envvar: str | None = None,
        custom_ca: str | None = None,
        custom_ca_envvar: str | None = None,
        log_level: int = logging.WARNING,
        async_runner: Callable[[Coroutine[Any, Any, T]], T] | None = None,
    ):
        """Constructor.

        Args:
            server_address: Server address.
            port: Server port. Inferred from URL scheme if not provided;
                defaults to 8080 for http:// URLs and 443 otherwise.

        Keyword Args:
            author: Name of the author.
            is_secure: Whether to use a secure connection. Inferred from URL scheme if not provided; defaults to True.
            user_token: The PEM-encoded user token as a string.
            user_token_envvar: Environment variable to read the user token from if it's not passed as an arg. Defaults to KF_PIPELINES_SA_TOKEN_PATH.
            custom_ca: Path to the PEM-encoded root certificates as a string.
            custom_ca_envvar: Environment variable to read the custom CA from if it's not passed as an arg.
            log_level: Log level. Defaults to logging.WARNING.
            async_runner: A modular async scheduler Callable (either a method or function) that takes in a coroutine for scheduling.
        """
        logger.setLevel(log_level)

        # TODO: get remaining args from env
        self._author = author
        # Set the user's defined async runner
        if async_runner:
            if not (inspect.ismethod(async_runner) or inspect.isfunction(async_runner)):
                msg = "`async_runner` must be a bound method or a function that takes in a coroutine to run."
                raise ValueError(msg)
            self._user_async_runner = async_runner
        else:
            import nest_asyncio2 as nest_asyncio

            logger.debug("Setting up reentrant async event loop")
            nest_asyncio.apply()

        if not user_token:
            user_token = self._get_user_token(user_token_envvar)
            if not user_token:
                warn("User access token is missing", stacklevel=2)

        from urllib.parse import urlparse

        parsed = urlparse(server_address)
        if parsed.scheme in ("http", "https"):
            if parsed.port:
                port = parsed.port
            elif port is None:
                port = 8080 if parsed.scheme == "http" else 443
            # Infer is_secure from scheme only if user didn't explicitly pass it
            if is_secure is None:
                is_secure = parsed.scheme == "https"
        elif port is None:
            port = 443
        # Fall back to secure by default if still unset
        if is_secure is None:
            is_secure = True
        self.hint_server_address_port(server_address, port)
        if is_secure:
            if not custom_ca and custom_ca_envvar and (cert := os.getenv(custom_ca_envvar)):
                logger.info(
                    "Using custom CA envvar %s",
                    custom_ca_envvar,
                )
                custom_ca = cert
                # client might have a default CA setup

            if not user_token:
                msg = "user token must be provided for secure connection"
                raise StoreError(msg)

            self._api = ModelRegistryAPIClient.secure_connection(
                server_address, port, user_token=user_token, custom_ca=custom_ca
            )
        else:
            self._api = ModelRegistryAPIClient.insecure_connection(server_address, port, user_token)
        self.get_registered_models().page_size(1)._next_page()

    @staticmethod
    def _get_user_token(user_token_envvar: str | None = None) -> str | None:
        sa_token_path: str
        user_provided: bool = True
        if user_token_envvar:
            try:
                sa_token_path = os.environ[user_token_envvar]
            except KeyError:
                msg = f"user_token_envvar is {user_token_envvar!r} but no such env var is set"
                raise ValueError(msg) from None
            logger.info(
                "Reading user token from path: user_token_envvar %r specifies path %r",
                user_token_envvar,
                sa_token_path,
            )
        elif DEFAULT_USER_TOKEN_ENVVAR in os.environ:
            sa_token_path = os.environ[DEFAULT_USER_TOKEN_ENVVAR]
            logger.info(
                "Reading user token from path: The default user token env var value %r specifies path %r",
                DEFAULT_USER_TOKEN_ENVVAR,
                sa_token_path,
            )
        else:
            sa_token_path = DEFAULT_K8S_SA_TOKEN_PATH
            user_provided = False
            logger.info(
                "Reading user token from path: No user_token_envvar. Attempting to read from default K8s service account path %r",
                DEFAULT_K8S_SA_TOKEN_PATH,
            )
        try:
            return Path(sa_token_path).read_text()
        except OSError as exc:
            msg = f"Unable read user token from {sa_token_path!r}"
            if user_provided:
                raise StoreError(msg) from exc
            logger.info(msg)
        return None

    @staticmethod
    def hint_server_address_port(server_address: str, port: int) -> None:
        """Hint based on server_address protocol if the port may not be the correct one."""
        if server_address.startswith("https://") and not str(port).endswith("443"):
            logger.warning(
                "Server address protocol is https://, but port is not 443 or ending with 443. You may want to verify the configuration is correct."
            )
        if server_address.startswith("http://") and not str(port).endswith("80"):
            logger.warning(
                "Server address protocol is http://, but port is not 80 or ending with 80. You may want to verify the configuration is correct."
            )

    @overload
    def async_runner(self, coro: Coroutine[Any, Any, TModel]) -> TModel: ...

    @overload
    def async_runner(self, coro: Coroutine[Any, Any, list[TModel]]) -> list[TModel]: ...

    @overload
    def async_runner(self, coro: Coroutine[Any, Any, TModel | None]) -> TModel | None: ...

    def async_runner(self, coro: Coroutine[Any, Any, T]) -> T:
        if hasattr(self, "_user_async_runner"):
            return self._user_async_runner(coro)

        import asyncio

        try:
            loop = asyncio.get_event_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
        return loop.run_until_complete(coro)

    async def _register_model(self, name: str, **kwargs) -> RegisteredModel:
        if rm := await self._api.get_registered_model_by_params(name):
            return rm

        return await self._api.upsert_registered_model(RegisteredModel(name=name, **kwargs))

    async def _register_new_version(self, rm: RegisteredModel, version: str, author: str, /, **kwargs) -> ModelVersion:
        assert rm.id is not None, "Registered model must have an ID"
        if await self._api.get_model_version_by_params(rm.id, version):
            msg = f"Version {version} already exists"
            raise StoreError(msg)

        return await self._api.upsert_model_version(ModelVersion(name=version, author=author, **kwargs), rm.id)

    async def _register_model_artifact(self, mv: ModelVersion, name: str, uri: str, /, **kwargs) -> ModelArtifact:
        assert mv.id is not None, "Model version must have an ID"
        return await self._api.upsert_model_version_artifact(ModelArtifact(name=name, uri=uri, **kwargs), mv.id)

    def upload_artifact_and_register_model(
        self,
        name: str,
        model_files_path: str,
        *,
        # Upload/client Params
        upload_params: OCIParams | S3Params,
        # Artifact/Model Params
        version: str,
        model_format_name: str,
        model_format_version: str,
        storage_path: str | None = None,
        storage_key: str | None = None,
        service_account_name: str | None = None,
        author: str | None = None,
        owner: str | None = None,
        version_description: str | None = None,
        description: str | None = None,
        metadata: Mapping[str, SupportedTypes] | None = None,
    ) -> RegisteredModel:
        """Convenience method to perform 2 operations; uploading an artifact to a storage location, and registers the model in model registry.

        Args:
            name: Name of the model.
            model_files_path: The path where the model files are located. If a directory, uploads the entire directory.

        Keyword Args:
            upload_params: Parameters to configure which storage client to use as well as that client's configuration when uploading the model.
            version: Version of the model. Has to be unique.
            model_format_name: Name of the model format.
            model_format_version: Version of the model format.
            version_description: Description of the model version.
            description: Deprecated. Use version_description instead.
            author: Author of the model. Defaults to the client author.
            owner: Owner of the model. Defaults to the client author.
            storage_key: Storage key.
            storage_path: Storage path.
            service_account_name: Service account name.
            metadata: Additional version metadata. Defaults to values returned by `default_metadata()`.

        Raises:
            ValueError: When the provided `upload_params` is missing or invalid

        Returns:
            Registered model. See: :meth:`~ModelRegistry.register_model`
        """
        # Handle deprecated parameter
        if description is not None:
            warn(
                "The 'description' parameter is deprecated and will be removed in a future version. "
                "Use 'version_description' instead.",
                DeprecationWarning,
                stacklevel=2,
            )
            if version_description is None:
                version_description = description

        # Check if model does not already exist in Registry
        ver = None
        with contextlib.suppress(StoreError):
            ver = self.get_model_version(name=name, version=version)

        if ver:
            msg = f"Model `{name}:{version}` already exists in Model Registry."
            raise StoreError(msg)

        if isinstance(upload_params, S3Params):
            destination_uri = self.save_to_s3(**asdict(upload_params), path=model_files_path)
        elif isinstance(upload_params, OCIParams):
            dict_params = asdict(upload_params)
            del dict_params["custom_oci_backend"]
            destination_uri = save_to_oci_registry(
                **dict_params,
                custom_oci_backend=upload_params.custom_oci_backend,
                model_files_path=model_files_path,
            )
        else:
            msg = 'Param "upload_params" is required to perform an upload. Please ensure the value provided is valid'
            raise ValueError(msg)

        return self.register_model(
            name,
            destination_uri,
            model_format_name=model_format_name,
            model_format_version=model_format_version,
            version=version,
            storage_key=storage_key,
            storage_path=storage_path,
            service_account_name=service_account_name,
            author=author,
            owner=owner,
            version_description=version_description,
            metadata=metadata,
        )

    def register_model(
        self,
        name: str,
        uri: str,
        *,
        model_format_name: str,
        model_format_version: str,
        version: str,
        storage_key: str | None = None,
        storage_path: str | None = None,
        service_account_name: str | None = None,
        model_source_kind: str | None = None,
        model_source_class: str | None = None,
        model_source_group: str | None = None,
        model_source_id: str | None = None,
        model_source_name: str | None = None,
        author: str | None = None,
        owner: str | None = None,
        version_description: str | None = None,
        description: str | None = None,
        metadata: Mapping[str, SupportedTypes] | None = None,
    ) -> RegisteredModel:
        """Register a model.

        This registers a model in the model registry. The model is not downloaded, and has to be stored prior to
        registration.

        Most models can be registered using their URI, along with optional connection-specific parameters, `storage_key`
        and `storage_path` or, simply a `service_account_name`.
        URI builder utilities are recommended when referring to specialized storage; for example `utils.s3_uri_from`
        helper when using S3 object storage data connections.

        Args:
            name: Name of the model.
            uri: URI of the model.

        Keyword Args:
            version: Version of the model. Has to be unique.
            model_format_name: Name of the model format.
            model_format_version: Version of the model format.
            version_description: Description of the model version.
            description: Deprecated. Use version_description instead.
            author: Author of the model. Defaults to the client author.
            owner: Owner of the model. Defaults to the client author.
            storage_key: Storage key.
            storage_path: Storage path.
            service_account_name: Service account name.
            model_source_kind: A string identifier describing the source kind.
            model_source_class: A subgroup within the source kind.
            model_source_group: This identifies a source group for models from source class.
            model_source_id: A unique identifier for a source model within kind, class, and group.
            model_source_name: A human-readable name for the source model.
            metadata: Additional version metadata. Defaults to values returned by `default_metadata()`.

        Returns:
            Registered model.
        """
        # Handle deprecated parameter
        if description is not None:
            warn(
                "The 'description' parameter is deprecated and will be removed in a future version. "
                "Use 'version_description' instead.",
                DeprecationWarning,
                stacklevel=2,
            )
            if version_description is None:
                version_description = description

        rm = self.async_runner(self._register_model(name, owner=owner or self._author))
        mv = self.async_runner(
            self._register_new_version(
                rm,
                version,
                author or self._author,
                description=version_description,
                custom_properties=metadata or {},
            )
        )
        self.async_runner(
            self._register_model_artifact(
                mv,
                name,
                uri,
                model_format_name=model_format_name,
                model_format_version=model_format_version,
                storage_key=storage_key,
                storage_path=storage_path,
                service_account_name=service_account_name,
                model_source_kind=model_source_kind,
                model_source_class=model_source_class,
                model_source_group=model_source_group,
                model_source_id=model_source_id,
                model_source_name=model_source_name,
            )
        )

        return rm

    def update(self, model: TModel) -> TModel:
        """Update a model."""
        if not model.id:
            msg = "Model must have an ID"
            raise StoreError(msg)
        if not isinstance(model, BaseResourceModel):
            msg = f"Model must be an instance of {BaseResourceModel.__name__} or a subclass"
            raise StoreError(msg)
        if isinstance(model, RegisteredModel):
            return self.async_runner(self._api.upsert_registered_model(model))  # type: ignore[return-value]
        if isinstance(model, ModelVersion):
            return self.async_runner(self._api.upsert_model_version(model, None))  # type: ignore[return-value]
        return self.async_runner(self._api.upsert_model_artifact(model))  # type: ignore[arg-type,return-value]

    def register_hf_model(  # noqa: C901
        self,
        repo: str,
        path: str,
        *,
        version: str,
        model_format_name: str,
        model_format_version: str,
        author: str | None = None,
        owner: str | None = None,
        model_name: str | None = None,
        version_description: str | None = None,
        description: str | None = None,
        git_ref: str = "main",
    ) -> RegisteredModel:
        """Register a Hugging Face model.

        This imports a model from Hugging Face hub and registers it in the model registry.
        Note that the model is not downloaded.

        Args:
            repo: Name of the repository from Hugging Face hub.
            path: URI of the model.

        Keyword Args:
            version: Version of the model. Has to be unique.
            model_format_name: Name of the model format.
            model_format_version: Version of the model format.
            author: Author of the model. Defaults to repo owner.
            owner: Owner of the model. Defaults to the client author.
            model_name: Name of the model. Defaults to the repo name.
            version_description: Description of the model version.
            description: Deprecated. Use version_description instead.
            git_ref: Git reference to use. Defaults to `main`.

        Returns:
            Registered model.
        """
        # Handle deprecated parameter
        if description is not None:
            warn(
                "The 'description' parameter is deprecated and will be removed in a future version. "
                "Use 'version_description' instead.",
                DeprecationWarning,
                stacklevel=2,
            )
            if version_description is None:
                version_description = description

        try:
            from huggingface_hub import HfApi, hf_hub_url, utils
        except ImportError as e:
            msg = """package `huggingface-hub` is not installed.
            To import models from Hugging Face Hub, start by installing the `huggingface-hub` package,
            either directly or as an extra (available as `model-registry[hf]`), e.g.:
            ```sh
            !pip install --pre model-registry[hf]
            ```
            or
            ```sh
            !pip install huggingface-hub
            ```
            """
            raise StoreError(msg) from e

        api = HfApi()
        try:
            model_info = api.model_info(repo, revision=git_ref)
        except utils.RepositoryNotFoundError as e:
            msg = f"Repository {repo} does not exist"
            raise StoreError(msg) from e
        except utils.RevisionNotFoundError as e:
            # TODO: as all hf-hub client calls default to using main, should we provide a tip?
            msg = f"Revision {git_ref} does not exist"
            raise StoreError(msg) from e

        if not author:
            # model author can be None if the repo is in a "global" namespace (i.e. no / in repo).
            if model_info.author is None:
                model_author = "unknown"
                warn(
                    "Model author is unknown. This is likely because the model is in a global namespace.",
                    stacklevel=2,
                )
            else:
                model_author = model_info.author
        else:
            model_author = author
        source_uri = hf_hub_url(repo, path, revision=git_ref)
        metadata = {
            "repo": repo,
            "source_uri": source_uri,
            "model_origin": "huggingface_hub",
            "model_author": model_author,
        }
        # card_data is the new field, but let's use the old one for backwards compatibility.
        if card_data := model_info.cardData:
            metadata.update(
                {
                    k: v
                    for k, v in card_data.to_dict().items()
                    # TODO: (#151) preserve tags, possibly other complex metadata
                    if isinstance(v, get_args(SupportedTypes))
                }
            )
        return self.register_model(
            model_name or model_info.id,
            source_uri,
            author=author or model_author,
            owner=owner or self._author,
            version=version,
            model_format_name=model_format_name,
            model_format_version=model_format_version,
            version_description=version_description,
            storage_path=path,
            metadata=metadata,
        )

    def get_registered_model(self, name: str) -> RegisteredModel | None:
        """Get a registered model.

        Args:
            name: Name of the model.

        Returns:
            Registered model.
        """
        return self.async_runner(self._api.get_registered_model_by_params(name))

    def get_model_version(self, name: str, version: str) -> ModelVersion | None:
        """Get a model version.

        Args:
            name: Name of the model.
            version: Version of the model.

        Returns:
            Model version.

        Raises:
            StoreException: If the model does not exist.
        """
        if not (rm := self.get_registered_model(name)):
            msg = f"Model {name} does not exist"
            raise StoreError(msg)
        assert rm.id
        return self.async_runner(self._api.get_model_version_by_params(rm.id, version))

    def get_model_artifact(self, name: str, version: str) -> ModelArtifact | None:
        """Get a model artifact.

        Args:
            name: Name of the model.
            version: Version of the model.

        Returns:
            Model artifact.

        Raises:
            StoreException: If either the model or the version don't exist.
        """
        if not (mv := self.get_model_version(name, version)):
            msg = f"Version {version} does not exist"
            raise StoreError(msg)
        assert mv.id
        return self.async_runner(self._api.get_model_artifact_by_params(name, mv.id))

    def get_registered_models(self) -> Pager[RegisteredModel]:
        """Get a pager for registered models.

        Returns:
            Iterable pager for registered models.
        """

        def rm_list(options: ListOptions) -> list[RegisteredModel]:
            return self.async_runner(self._api.get_registered_models(options))

        return Pager[RegisteredModel](rm_list)

    def get_model_versions(self, name: str) -> Pager[ModelVersion]:
        """Get a pager for model versions.

        Args:
            name: Name of the model.

        Returns:
            Iterable pager for model versions.

        Raises:
            StoreException: If the model does not exist.
        """
        if not (rm := self.get_registered_model(name)):
            msg = f"Model {name} does not exist"
            raise StoreError(msg)

        def rm_versions(options: ListOptions) -> list[ModelVersion]:
            # type checkers can't restrict the type inside a nested function:
            # https://mypy.readthedocs.io/en/stable/common_issues.html#narrowing-and-inner-functions
            assert rm.id
            return self.async_runner(self._api.get_model_versions(rm.id, options))

        return Pager[ModelVersion](rm_versions)

    def save_to_s3(
        self,
        path: str,
        bucket_name: str,
        s3_prefix: str,
        *,
        endpoint_url: str | None = None,
        access_key_id: str | None = None,
        secret_access_key: str | None = None,
        region: str | None = None,
        multipart_threshold: int = 1024 * 1024,
        multipart_chunksize: int = 1024 * 1024,
        max_pool_connections: int = 10,
    ) -> str:
        """Saves a model to an S3 compatible storage.

        Args:
            path: Location to where the model(s) or artifact(s) are located. \
                Can recursively upload nested files in folders.
            bucket_name: The bucket to use for the S3 compatible object storage.
            s3_prefix: The path to prefix under root of bucket.

        Keyword Args:
            endpoint_url: The endpoint URL for the S3 comaptible storage if not using AWS S3.
            access_key_id: The S3 compatible object storage access ID.
            secret_access_key: The S3 compatible object storage secret access key.
            region: The region name for the S3 object storage.
            multipart_threshold: The threshold for multipart uploads.
            multipart_chunksize: The size of chunks for multipart uploads.
            max_pool_connections: The maximum number of connections in the pool.

        Returns:
            The S3 URI to the uploaded files.

        Raises:
            StoreError: If there was an issue uploading to S3.
        """
        # Get mixed credentials
        endpoint_url, access_key_id, secret_access_key, region = _s3_creds(
            endpoint_url, access_key_id, secret_access_key, region
        )

        s3, transfer_config = _connect_to_s3(
            endpoint_url=endpoint_url,
            access_key_id=access_key_id,
            secret_access_key=secret_access_key,
            multipart_threshold=multipart_threshold,
            multipart_chunksize=multipart_chunksize,
            max_pool_connections=max_pool_connections,
        )
        return _upload_to_s3(
            path=path,
            path_prefix=s3_prefix,
            bucket=bucket_name,
            s3=s3,
            endpoint_url=endpoint_url,
            region=region,
            transfer_config=transfer_config,
        )

    # TODO: consider porting get_artifacts method here
    # https://github.com/kubeflow/hub/pull/1536
