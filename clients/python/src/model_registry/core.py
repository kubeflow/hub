"""Client for the model registry."""

from __future__ import annotations

from collections.abc import AsyncIterator
from contextlib import asynccontextmanager
from dataclasses import dataclass
from typing import TypeVar, cast, overload
from urllib.parse import urlparse

from mr_openapi import (
    ApiClient,
    Configuration,
    ModelRegistryServiceApi,
)
from mr_openapi import (
    exceptions as mr_exceptions,
)

from ._utils import required_args
from .types import (
    Artifact,
    ListOptions,
    ModelArtifact,
    ModelVersion,
    RegisteredModel,
)

ArtifactT = TypeVar("ArtifactT", bound=Artifact)


@dataclass
class ModelRegistryAPIClient:
    """Model registry API."""

    config: Configuration

    @classmethod
    def secure_connection(
        cls,
        server_address: str,
        port: int = 443,
        *,
        user_token: str,
        custom_ca: str | None = None,
    ) -> ModelRegistryAPIClient:
        """Constructor.

        Args:
            server_address: Server address.
            port: Server port. Defaults to 443.

        Keyword Args:
            user_token: The PEM-encoded user token as a string.
            custom_ca: The path to a PEM-
        """
        parsed_url = urlparse(server_address)
        host = server_address if parsed_url.port else f"{server_address}:{port}"

        return cls(
            Configuration(
                host=host,
                access_token=user_token,
                ssl_ca_cert=custom_ca,
            )
        )

    @classmethod
    def insecure_connection(
        cls,
        server_address: str,
        port: int,
        user_token: str | None = None,
    ) -> ModelRegistryAPIClient:
        """Constructor.

        Args:
            server_address: Server address.
            port: Server port.
            user_token: The PEM-encoded user token as a string.
        """
        parsed_url = urlparse(server_address)
        host = server_address if parsed_url.port else f"{server_address}:{port}"

        config = Configuration(
            host=host,
            access_token=user_token,
            verify_ssl=False,
        )
        return cls(config)

    @asynccontextmanager
    async def get_client(self) -> AsyncIterator[ModelRegistryServiceApi]:
        """Get a client for the model registry."""
        api_client = ApiClient(self.config)
        client = ModelRegistryServiceApi(api_client)

        try:
            yield client
        finally:
            await api_client.close()

    async def upsert_registered_model(self, registered_model: RegisteredModel) -> RegisteredModel:
        """Upsert a registered model.

        Updates or creates a registered model on the server.

        Args:
            registered_model: Registered model.

        Returns:
            New registered model.
        """
        async with self.get_client() as client:
            if registered_model.id:
                rm = await client.update_registered_model(registered_model.id, registered_model.update())
            else:
                rm = await client.create_registered_model(registered_model.create())

        return RegisteredModel.from_basemodel(rm)

    async def get_registered_model_by_id(self, id: str) -> RegisteredModel | None:
        """Fetch a registered model by its ID.

        Args:
            id: Registered model ID.

        Returns:
            Registered model.
        """
        async with self.get_client() as client:
            try:
                rm = await client.get_registered_model(id)
            except mr_exceptions.NotFoundException:
                return None

        return RegisteredModel.from_basemodel(rm)

    @overload
    async def get_registered_model_by_params(self, name: str): ...

    @overload
    async def get_registered_model_by_params(self, *, external_id: str): ...

    @required_args(("name",), ("external_id",))
    async def get_registered_model_by_params(
        self, name: str | None = None, external_id: str | None = None
    ) -> RegisteredModel | None:
        """Fetch a registered model by its name or external ID.

        Args:
            name: Registered model name.
            external_id: Registered model external ID.

        Returns:
            Registered model.
        """
        async with self.get_client() as client:
            try:
                rm = await client.find_registered_model(name=name, external_id=external_id)
            except mr_exceptions.NotFoundException:
                return None

        return RegisteredModel.from_basemodel(rm)

    async def get_registered_models(self, options: ListOptions | None = None) -> list[RegisteredModel]:
        """Fetch registered models.

        Args:
            options: Options for listing registered models.

        Returns:
            Registered models.
        """
        async with self.get_client() as client:
            rm_list = await client.get_registered_models(**(options or ListOptions()).as_options())

        if options:
            options.next_page_token = rm_list.next_page_token

        return [RegisteredModel.from_basemodel(rm) for rm in rm_list.items or []]

    async def upsert_model_version(
        self, model_version: ModelVersion, registered_model_id: str | None = None
    ) -> ModelVersion:
        """Upsert a model version.

        Updates or creates a model version on the server.

        Args:
            model_version: Model version to upsert.
            registered_model_id: ID of the registered model this version will be associated to. Can be None when updating an existing model version.

        Returns:
            New model version.
        """
        async with self.get_client() as client:
            if model_version.id:
                mv = await client.update_model_version(model_version.id, model_version.update())
            elif registered_model_id:
                mv = await client.create_model_version(model_version.create(registered_model_id=registered_model_id))
            else:
                msg = f"Registered model ID required for creating a new model version: {model_version}"
                raise ValueError(msg)

        return ModelVersion.from_basemodel(mv)

    async def get_model_version_by_id(self, model_version_id: str) -> ModelVersion | None:
        """Fetch a model version by its ID.

        Args:
            model_version_id: Model version ID.

        Returns:
            Model version.
        """
        async with self.get_client() as client:
            try:
                mv = await client.get_model_version(model_version_id)
            except mr_exceptions.NotFoundException:
                return None

        return ModelVersion.from_basemodel(mv)

    async def get_model_versions(
        self, registered_model_id: str, options: ListOptions | None = None
    ) -> list[ModelVersion]:
        """Fetch model versions by registered model ID.

        Args:
            registered_model_id: Registered model ID.
            options: Options for listing model versions.

        Returns:
            Model versions.
        """
        async with self.get_client() as client:
            mv_list = await client.get_registered_model_versions(
                registered_model_id, **(options or ListOptions()).as_options()
            )

        if options:
            options.next_page_token = mv_list.next_page_token

        return [ModelVersion.from_basemodel(mv) for mv in mv_list.items or []]

    @overload
    async def get_model_version_by_params(self, registered_model_id: str, name: str): ...

    @overload
    async def get_model_version_by_params(self, *, external_id: str): ...

    @required_args(
        (
            "registered_model_id",
            "name",
        ),
        ("external_id",),
    )
    async def get_model_version_by_params(
        self,
        registered_model_id: str | None = None,
        name: str | None = None,
        external_id: str | None = None,
    ) -> ModelVersion | None:
        """Fetch a model version by associated parameters.

        Either fetches by using external ID or by using registered model ID and version name.

        Args:
            registered_model_id: Registered model ID.
            name: Model version.
            external_id: Model version external ID.

        Returns:
            Model version.
        """
        async with self.get_client() as client:
            try:
                mv = await client.find_model_version(
                    name=name,
                    external_id=external_id,
                    parent_resource_id=registered_model_id,
                )
            except mr_exceptions.NotFoundException:
                return None

        return ModelVersion.from_basemodel(mv)

    async def upsert_model_artifact(self, model_artifact: ModelArtifact) -> ModelArtifact:
        """Upsert a model artifact.

        Updates or creates a model artifact on the server.

        Args:
            model_artifact: Model artifact to upsert.
            model_version_id: ID of the model version this artifact will be associated to.

        Returns:
            New model artifact.
        """
        async with self.get_client() as client:
            if not model_artifact.id:
                ma = await client.create_model_artifact(model_artifact.create())
            else:
                ma = await client.update_model_artifact(model_artifact.id, model_artifact.update())
        return ModelArtifact.from_basemodel(ma)

    async def upsert_model_version_artifact(self, artifact: ArtifactT, model_version_id: str) -> ArtifactT:
        """Creates a model version artifact.

        Creates a model version artifact on the server.

        Args:
            artifact: Model version artifact to upsert.
            model_version_id: ID of the model version this artifact will be associated to.

        Returns:
            New model version artifact.
        """
        async with self.get_client() as client:
            return cast(
                ArtifactT,
                Artifact.validate_artifact(
                    await client.upsert_model_version_artifact(model_version_id, artifact.wrap())
                ),
            )

    async def get_model_artifact_by_id(self, id: str) -> ModelArtifact | None:
        """Fetch a model artifact by its ID.

        Args:
            id: Model artifact ID.

        Returns:
            Model artifact.
        """
        async with self.get_client() as client:
            try:
                ma = await client.get_model_artifact(id)
            except mr_exceptions.NotFoundException:
                return None

        return ModelArtifact.from_basemodel(ma)

    @overload
    async def get_model_artifact_by_params(
        self,
        name: str,
        model_version_id: str,
    ): ...

    @overload
    async def get_model_artifact_by_params(self, *, external_id: str): ...

    @required_args(
        (
            "name",
            "model_version_id",
        ),
        ("external_id",),
    )
    async def get_model_artifact_by_params(
        self,
        name: str | None = None,
        model_version_id: str | None = None,
        external_id: str | None = None,
    ) -> ModelArtifact | None:
        """Fetch a model artifact either by external ID or by its name and the ID of its associated model version.

        Args:
            name: Model artifact name.
            model_version_id: ID of the associated model version.
            external_id: Model artifact external ID.

        Returns:
            Model artifact.
        """
        async with self.get_client() as client:
            try:
                ma = await client.find_model_artifact(
                    name=name,
                    parent_resource_id=model_version_id,
                    external_id=external_id,
                )
            except mr_exceptions.NotFoundException:
                return None

        return ModelArtifact.from_basemodel(ma)

    async def get_model_artifacts(
        self,
        model_version_id: str | None = None,
        options: ListOptions | None = None,
    ) -> list[ModelArtifact]:
        """Fetches model artifacts.

        Args:
            model_version_id: ID of the associated model version.
            options: Options for listing model artifacts.

        Returns:
            Model artifacts.
        """
        async with self.get_client() as client:
            if model_version_id:
                art_list = await client.get_model_version_artifacts(
                    model_version_id, **(options or ListOptions()).as_options()
                )
                if options:
                    options.next_page_token = art_list.next_page_token
                models = []
                for art in art_list.items or []:
                    converted = Artifact.validate_artifact(art)
                    if isinstance(converted, ModelArtifact):
                        models.append(converted)
                return models

            ma_list = await client.get_model_artifacts(**(options or ListOptions()).as_options())
            if options:
                options.next_page_token = ma_list.next_page_token
            return [ModelArtifact.from_basemodel(ma) for ma in ma_list.items or []]

    async def get_model_version_artifacts(
        self,
        model_version_id: str,
        options: ListOptions | None = None,
    ) -> list[Artifact]:
        """Fetches model artifacts.

        Args:
            model_version_id: ID of the associated model version.
            options: Options for listing model artifacts.

        Returns:
            Model artifacts.
        """
        async with self.get_client() as client:
            art_list = await client.get_model_version_artifacts(
                model_version_id, **(options or ListOptions()).as_options()
            )
        if options:
            options.next_page_token = art_list.next_page_token
        return [Artifact.validate_artifact(art) for art in art_list.items or []]

    async def get_artifacts(
        self,
        filter_query: str | None = None,
        artifact_type: str | None = None,
        options: ListOptions | None = None,
    ) -> list[Artifact]:
        """Get artifacts with filtering capabilities.

        Args:
            filter_query: A SQL-like query string to filter the list of entities.
            artifact_type: Specifies the artifact type for listing artifacts.
            options: Options for listing artifacts.

        Returns:
            List of artifacts matching the filter criteria.
        """
        # Create options dict with additional filters
        options_dict = (options or ListOptions()).as_options()
        if filter_query is not None:
            options_dict["filter_query"] = filter_query
        if artifact_type is not None:
            options_dict["artifact_type"] = artifact_type

        async with self.get_client() as client:
            try:
                response = await client.get_artifacts(**options_dict)
            except mr_exceptions.NotFoundException:
                return []

        if options:
            options.next_page_token = response.next_page_token
        return [Artifact.validate_artifact(artifact) for artifact in response.items or []]
