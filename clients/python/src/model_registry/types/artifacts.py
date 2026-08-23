"""Artifact types for model registry.

Artifacts represent pieces of data.
This could be datasets, models, metrics, or any other piece of data produced or consumed by an
execution.

Those types are used to map between proto types based on artifacts and Python objects.

Todo:
    * Move part of the description to API Reference docs (#120).
"""

from __future__ import annotations  # noqa: I001

import warnings
from abc import ABC, abstractmethod
from typing import Any, TypeVar

from typing_extensions import override


import json

from mr_openapi import (
    Artifact as ArtifactBaseModel,
)
from mr_openapi import (
    DataSet as DataSetBaseModel,
)
from mr_openapi import (
    Metric as MetricBaseModel,
)
from mr_openapi import (
    Parameter as ParameterBaseModel,
)

from mr_openapi import (
    ArtifactState,
    DocArtifactCreate,
    DocArtifactUpdate,
    ModelArtifactCreate,
    ModelArtifactUpdate,
    DataSetCreate,
    DataSetUpdate,
    MetricCreate,
    MetricUpdate,
    ParameterCreate,
    ParameterUpdate,
    ParameterType,
)
from mr_openapi import (
    DocArtifact as DocArtifactBaseModel,
)
from mr_openapi import (
    ModelArtifact as ModelArtifactBaseModel,
)

from .base import BaseResourceModel

A = TypeVar("A", bound="Artifact")


class Artifact(BaseResourceModel, ABC):
    """Base class for all artifacts.

    Attributes:
        name: Name of the artifact.
        state: State of the artifact.
        description: Description of the artifact.
        external_id: Customizable ID. Has to be unique among instances of the same type.
    """

    name: str | None = None
    state: ArtifactState = ArtifactState.UNKNOWN
    description: str | None = None
    external_id: str | None = None

    @classmethod
    def from_artifact(cls: type[A], source: ArtifactBaseModel) -> A:
        """Convert a base artifact."""
        model = source.actual_instance
        assert model
        return cls.from_basemodel(model)

    @staticmethod
    def validate_artifact(
        source: ArtifactBaseModel,
    ) -> DocArtifact | ModelArtifact | DataSet | Metric | Parameter:
        """Validate an artifact."""
        model = source.actual_instance
        assert model
        if isinstance(model, DocArtifactBaseModel):
            return DocArtifact.from_basemodel(model)
        if isinstance(model, ModelArtifactBaseModel):
            return ModelArtifact.from_basemodel(model)
        if isinstance(model, DataSetBaseModel):
            return DataSet.from_basemodel(model)
        if isinstance(model, MetricBaseModel):
            return Metric.from_basemodel(model)
        if isinstance(model, ParameterBaseModel):
            return Parameter.from_basemodel(model)
        msg = f"Invalid artifact type: {type(model)}"
        raise ValueError(msg)

    @abstractmethod
    def as_basemodel(self) -> Any:
        """Wrap the object in a BaseModel object."""

    def wrap(self) -> ArtifactBaseModel:
        """Wrap the object in a ArtifactBaseModel object."""
        return ArtifactBaseModel(self.as_basemodel())


class DocArtifact(Artifact):
    """Represents a Document Artifact.

    Attributes:
        name: Name of the document.
        uri: URI of the document.
        description: Description of the object.
        external_id: Customizable ID. Has to be unique among instances of the same type.
    """

    uri: str | None = None

    @override
    def create(self, **kwargs) -> DocArtifactCreate:
        """Create a new DocArtifactCreate object."""
        return DocArtifactCreate(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("id", "custom_properties")),
            artifactType="doc-artifact",
            **kwargs,
        )

    @override
    def update(self, **kwargs) -> DocArtifactUpdate:
        """Create a new DocArtifactUpdate object."""
        return DocArtifactUpdate(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("id", "name", "custom_properties")),
            artifactType="doc-artifact",
            **kwargs,
        )

    @override
    def as_basemodel(self) -> DocArtifactBaseModel:
        return DocArtifactBaseModel(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("custom_properties")),
            artifactType="doc-artifact",
        )

    @classmethod
    @override
    def from_basemodel(cls, source: DocArtifactBaseModel) -> DocArtifact:
        assert source.name
        assert source.state
        return cls(
            id=source.id,
            name=source.name,
            description=source.description,
            external_id=source.external_id,
            create_time_since_epoch=source.create_time_since_epoch,
            last_update_time_since_epoch=source.last_update_time_since_epoch,
            uri=source.uri,
            state=source.state,
            custom_properties=cls._unmap_custom_properties(source.custom_properties)
            if source.custom_properties
            else None,
        )


class ModelArtifact(Artifact):
    """Represents a Model.

    Attributes:
        name: Name of the model.
        uri: URI of the model.
        description: Description of the object.
        external_id: Customizable ID. Has to be unique among instances of the same type.
        model_format_name: Name of the model format.
        model_format_version: Version of the model format.
        storage_key: Storage secret name.
        storage_path: Storage path of the model.
        service_account_name: Name of the service account with storage secret.
        model_source_kind: A string identifier describing the source kind.
        model_source_class: A subgroup within the source kind.
        model_source_group: This identifies a source group for models from source class.
        model_source_id: A unique identifier for a source model within kind, class, and group.
        model_source_name: A human-readable name for the source model.
    """

    # TODO: this could be an enum of valid formats
    model_format_name: str | None = None
    model_format_version: str | None = None
    storage_key: str | None = None
    storage_path: str | None = None
    service_account_name: str | None = None
    model_source_kind: str | None = None
    model_source_class: str | None = None
    model_source_group: str | None = None
    model_source_id: str | None = None
    model_source_name: str | None = None
    uri: str | None = None

    _model_version_id: str | None = None

    @override
    def create(self, **kwargs) -> ModelArtifactCreate:
        """Create a new ModelArtifactCreate object."""
        return ModelArtifactCreate(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("id", "custom_properties")),
            artifactType="model-artifact",
            **kwargs,
        )

    @override
    def update(self, **kwargs) -> ModelArtifactUpdate:
        """Create a new ModelArtifactUpdate object."""
        return ModelArtifactUpdate(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("id", "name", "custom_properties")),
            artifactType="model-artifact",
            **kwargs,
        )

    @override
    def as_basemodel(self) -> ModelArtifactBaseModel:
        return ModelArtifactBaseModel(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("custom_properties")),
            artifactType="model-artifact",
        )

    @classmethod
    @override
    def from_basemodel(cls, source: ModelArtifactBaseModel) -> ModelArtifact:
        """Create a new ModelArtifact object from a BaseModel object."""
        assert source.name
        assert source.uri
        assert source.state
        return cls(
            id=source.id,
            name=source.name,
            description=source.description,
            external_id=source.external_id,
            create_time_since_epoch=source.create_time_since_epoch,
            last_update_time_since_epoch=source.last_update_time_since_epoch,
            uri=source.uri,
            model_format_name=source.model_format_name,
            model_format_version=source.model_format_version,
            storage_key=source.storage_key,
            storage_path=source.storage_path,
            service_account_name=source.service_account_name,
            model_source_kind=source.model_source_kind,
            model_source_class=source.model_source_class,
            model_source_group=source.model_source_group,
            model_source_id=source.model_source_id,
            model_source_name=source.model_source_name,
            state=source.state,
            custom_properties=cls._unmap_custom_properties(source.custom_properties)
            if source.custom_properties
            else None,
        )


warnings.filterwarnings("ignore", message='Field name "schema" in "DataSet" shadows an attribute')


class DataSet(Artifact):
    """Represents a DataSet.

    Attributes:
        name: Name of the data set.
        uri: URI of the data set.
        description: Description of the object.
        external_id: Customizable ID. Has to be unique among instances of the same type.
        digest: A unique hash or identifier for the dataset content.
        source_type: The type of source for the dataset.
        source: The location or connection string for the dataset source.
        schema: JSON schema or description of the dataset structure.
        profile: Statistical profile or summary of the dataset.
    """

    uri: str | None = None
    digest: str | None = None
    source_type: str | None = None
    source: str | None = None
    schema: str | None = None  # type: ignore[assignment]
    profile: str | None = None

    @override
    def create(self, **kwargs) -> DataSetCreate:
        """Create a new DataSetCreate object."""
        return DataSetCreate(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("id", "custom_properties")),
            artifactType="dataset-artifact",
            **kwargs,
        )

    @override
    def update(self, **kwargs) -> DataSetUpdate:
        """Create a new DataSetUpdate object."""
        return DataSetUpdate(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("id", "name", "custom_properties")),
            artifactType="dataset-artifact",
            **kwargs,
        )

    @override
    def as_basemodel(self) -> DataSetBaseModel:
        return DataSetBaseModel(
            customProperties=self._map_custom_properties(),
            **self._props_as_dict(exclude=("custom_properties")),
            artifactType="dataset-artifact",
        )

    @classmethod
    @override
    def from_basemodel(cls, source: DataSetBaseModel) -> DataSet:
        """Create a new DataSet object from a BaseModel object."""
        assert source.name
        return cls(
            id=source.id,
            name=source.name,
            description=source.description,
            external_id=source.external_id,
            create_time_since_epoch=source.create_time_since_epoch,
            last_update_time_since_epoch=source.last_update_time_since_epoch,
            uri=source.uri,
            digest=source.digest,
            source_type=source.source_type,
            source=source.source,
            schema=source.var_schema,
            profile=source.profile,
            state=source.state,  # type: ignore[arg-type]
            custom_properties=cls._unmap_custom_properties(source.custom_properties)
            if source.custom_properties
            else None,
        )


class Metric(Artifact):
    """Represents a Metric.

    Attributes:
        name: Name of the metric.
        description: Description of the object.
        external_id: Customizable ID. Has to be unique among instances of the same type.
        value: The numeric value of the metric.
        timestamp: Unix timestamp in milliseconds when the metric was recorded.
        step: The step number for multi-step metrics (e.g., training epochs)
    """

    value: float
    timestamp: str | None = None
    step: int = 0

    @override
    def create(self, **kwargs) -> MetricCreate:
        """Create a new MetricCreate object."""
        return MetricCreate(
            customProperties=self._map_custom_properties(),
            timestamp=self.timestamp,
            **self._props_as_dict(exclude=("id", "timestamp", "custom_properties")),
            artifactType="metric",
            **kwargs,
        )

    @override
    def update(self, **kwargs) -> MetricUpdate:
        """Create a new MetricUpdate object."""
        return MetricUpdate(
            customProperties=self._map_custom_properties(),
            timestamp=self.timestamp,
            **self._props_as_dict(exclude=("id", "name", "timestamp", "custom_properties")),
            artifactType="metric",
            **kwargs,
        )

    @override
    def as_basemodel(self) -> MetricBaseModel:
        return MetricBaseModel(
            customProperties=self._map_custom_properties(),
            timestamp=self.timestamp,
            **self._props_as_dict(exclude=("timestamp", "custom_properties")),
            artifactType="metric",
        )

    @classmethod
    @override
    def from_basemodel(cls, source: MetricBaseModel) -> Metric:
        """Create a new Metric object from a BaseModel object."""
        assert source.name
        return cls(
            id=source.id,
            name=source.name,
            description=source.description,
            external_id=source.external_id,
            create_time_since_epoch=source.create_time_since_epoch,
            last_update_time_since_epoch=source.last_update_time_since_epoch,
            value=source.value,  # type: ignore[arg-type]
            timestamp=source.timestamp,
            step=source.step,  # type: ignore[arg-type]
            state=source.state,  # type: ignore[arg-type]
            custom_properties=cls._unmap_custom_properties(source.custom_properties)
            if source.custom_properties
            else None,
        )


class Parameter(Artifact):
    """Represents a Parameter.

    Attributes:
        name: Name of the parameter.
        description: Description of the object.
        external_id: Customizable ID. Has to be unique among instances of the same type.
        parameter_type: The data type of the parameter (e.g., "string", "number", "boolean", "object").
        value: The value of the parameter.
    """

    value: str | bool | int | float | dict
    parameter_type: ParameterType

    @override
    def create(self, **kwargs) -> ParameterCreate:
        """Create a new ParameterCreate object."""
        return ParameterCreate(
            customProperties=self._map_custom_properties(),
            value=str(self.value),
            **self._props_as_dict(exclude=("id", "value", "custom_properties")),
            artifactType="parameter",
            **kwargs,
        )

    @override
    def update(self, **kwargs) -> ParameterUpdate:
        """Create a new ParameterUpdate object."""
        return ParameterUpdate(
            customProperties=self._map_custom_properties(),
            value=str(self.value),
            **self._props_as_dict(exclude=("id", "name", "value", "custom_properties")),
            artifactType="parameter",
            **kwargs,
        )

    @override
    def as_basemodel(self) -> ParameterBaseModel:
        return ParameterBaseModel(
            customProperties=self._map_custom_properties(),
            value=str(self.value),
            **self._props_as_dict(exclude=("value", "custom_properties")),
            artifactType="parameter",
        )

    @classmethod
    @override
    def from_basemodel(cls, source: ParameterBaseModel) -> Parameter:
        """Create a new Parameter object from a BaseModel object."""
        assert source.name
        assert source.parameter_type
        value = source.value
        if source.parameter_type is ParameterType.NUMBER:
            value = float(value)  # type: ignore[assignment,arg-type]
        elif source.parameter_type is ParameterType.BOOLEAN:
            value = bool(value)  # type: ignore[assignment]
        elif source.parameter_type is ParameterType.OBJECT:
            value = json.loads(value)  # type: ignore[arg-type]
        return cls(
            id=source.id,
            name=source.name,
            description=source.description,
            external_id=source.external_id,
            create_time_since_epoch=source.create_time_since_epoch,
            last_update_time_since_epoch=source.last_update_time_since_epoch,
            value=value,  # type: ignore[arg-type]
            parameter_type=source.parameter_type,
            state=source.state,  # type: ignore[arg-type]
            custom_properties=cls._unmap_custom_properties(source.custom_properties)
            if source.custom_properties
            else None,
        )
