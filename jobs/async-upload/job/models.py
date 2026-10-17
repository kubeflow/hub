"""
Configuration models for the async upload job using Pydantic for type safety and validation.
"""
from __future__ import annotations
from enum import StrEnum
import logging
from typing import Union, Annotated, Literal
from pydantic import BaseModel, Field, model_validator, ConfigDict


class SourceType(StrEnum):
    """Supported source types for model artifacts."""
    S3 = "s3"
    OCI = "oci"
    URI = "uri"


class DestinationType(StrEnum):
    """Supported destination types for model artifacts."""
    S3 = "s3"
    OCI = "oci"


class BaseStorageConfig(BaseModel):
    """Base configuration for storage types."""
    credentials_path: str | None = None


class S3Config(BaseModel):
    """Basic S3 storage configuration. To be used as an intermediary model for the S3StorageConfig, allowing for additional values to be overlaid until the final config is created and validated via the S3StorageConfig model."""
    bucket: str | None = None
    key: str | None = None # 'path' in bucket
    region: str | None = None
    access_key_id: str | None = None
    secret_access_key: str | None = None
    endpoint: str | None = None


class OCIConfig(BaseModel):
    """Basic OCI registry configuration. To be used as an intermediary model for the OCIStorageConfig, allowing for additional values to be overlaid until the final config is created and validated via the OCIStorageConfig model."""
    uri: str
    registry: str | None = None
    username: str | None = None
    password: str | None = None
    email: str | None = None
    base_image: str = "public.ecr.aws/docker/library/busybox:latest"
    base_image_source: Literal["explicit", "deployment-config", "bff-default", "job-default"] = "explicit"
    enable_tls_verify: bool = True

    @model_validator(mode='before')
    @classmethod
    def default_base_image_source(cls, values):
        if isinstance(values, dict) and "base_image" not in values and "base_image_source" not in values:
            values = {**values, "base_image_source": "job-default"}
        return values

    @model_validator(mode='after')
    def validate_oci_credentials(self) -> 'OCIConfig':
        """Validate OCI credentials consistency."""
        if (self.username is None) != (self.password is None):
            raise ValueError("Both OCI username and password must be set together or both be None")
        return self


class S3StorageConfig(BaseStorageConfig, S3Config):
    """S3 storage configuration with validation - can be used for both source and destination."""

    @model_validator(mode='after')
    def validate_s3_storage(self) -> 'S3StorageConfig':
        """Validate that required S3 fields are present."""
        if not all([self.access_key_id, self.secret_access_key, self.bucket, self.key]):
            raise ValueError("S3 credentials (access_key_id, secret_access_key), bucket, and key must be set")
        return self


class OCIStorageConfig(BaseStorageConfig, OCIConfig):
    """OCI storage configuration with validation - can be used for both source and destination."""

    @model_validator(mode='after')
    def validate_oci_storage(self) -> 'OCIStorageConfig':
        """Validate that required OCI fields are present."""
        if not all([self.registry, self.uri]):
            raise ValueError("OCI registry and URI must be set")
        return self


class URISourceConfig(BaseModel):
    """Basic URI source configuration. To be used as an intermediary model for the URISourceStorageConfig, allowing for additional values to be overlaid until the final config is created and validated via the URISourceStorageConfig model."""
    uri: str | None = None


class URISourceStorageConfig(BaseStorageConfig, URISourceConfig):
    """URI source storage configuration with validation - only used for sources, not destinations."""

    @model_validator(mode='after')
    def validate_uri_storage(self) -> 'URISourceStorageConfig':
        """Validate that required URI field is present."""
        if not self.uri:
            raise ValueError("URI must be set for URI type sources")
        return self


# Union types for source and destination configurations - this enables isinstance() checks
SourceConfig = Union[S3StorageConfig, OCIStorageConfig, URISourceStorageConfig]
DestinationConfig = Union[S3StorageConfig, OCIStorageConfig]

class UploadIntent(StrEnum):
    create_model = "create_model"
    create_version = "create_version"
    update_artifact = "update_artifact"


class ModelInputArgs(BaseModel):
    model_config = ConfigDict(frozen=True)

    intent_type: UploadIntent = Field(description="Upload intent type")
    model_id: str | None = Field(description="Registered model ID")
    version_id: str | None = Field(description="Model version ID")
    artifact_id: str | None = Field(description="Model artifact ID")


class CreateModelIntent(BaseModel):
    intent_type: Literal[UploadIntent.create_model] = UploadIntent.create_model


class CreateVersionIntent(BaseModel):
    intent_type: Literal[UploadIntent.create_version] = UploadIntent.create_version
    model_id: str = Field(..., description="Registered model ID for the existing model")


class UpdateArtifactIntent(BaseModel):
    intent_type: Literal[UploadIntent.update_artifact] = UploadIntent.update_artifact
    artifact_id: str = Field(..., description="Model artifact ID to update")
    # Optional IDs to pass through to termination message output
    model_id: str | None = Field(default=None, description="Optional registered model ID to include in output")
    version_id: str | None = Field(default=None, description="Optional model version ID to include in output")


IntentConfig = Union[CreateModelIntent, CreateVersionIntent, UpdateArtifactIntent]


class ModelConfig(BaseModel):
    """Model registry model information with intent-specific configuration."""
    intent: IntentConfig = Field(..., description="Model upload intent configuration", discriminator='intent_type')


class StorageConfig(BaseModel):
    """Storage configuration for temporary files."""
    path: str = Field(default="/tmp/model-sync", description="Local storage path for temporary files")


class RegistryConfig(BaseModel):
    """Model registry client configuration."""
    server_address: str = Field(..., description="Model registry server address")
    port: int = Field(default=443, description="Model registry server port")
    is_secure: bool = Field(default=True, description="Use secure connection")
    author: str | None = Field(default=None, description="Author for model registration")
    user_token: str | None = Field(default=None, description="User authentication token")
    user_token_envvar: str | None = Field(default=None, description="Environment variable containing user token")
    custom_ca: str | None = Field(default=None, description="Custom CA certificate")
    custom_ca_envvar: str | None = Field(default=None, description="Environment variable containing custom CA")
    log_level: int = Field(default=logging.WARNING, description="Logging level for registry client")

    @model_validator(mode='after')
    def validate_registry_config(self) -> 'RegistryConfig':
        """Validate registry configuration."""
        if not self.server_address:
            raise ValueError("Registry server address must be set")
        return self


class RegisteredModelMetadata(BaseModel):
    """Metadata for creating a RegisteredModel."""
    name: str | None = None
    id: str | None = None  # Alternative to name for existing models
    description: str | None = None
    owner: str | None = None
    custom_properties: dict | None = None

    @model_validator(mode='after')
    def validate_name_or_id(self) -> 'RegisteredModelMetadata':
        """Validate that either name or id is provided, but not both."""
        if self.name and self.id:
            raise ValueError("Cannot provide both name and id for RegisteredModel")
        if not self.name and not self.id:
            raise ValueError("Must provide either name or id for RegisteredModel")
        return self


class ModelVersionMetadata(BaseModel):
    """Metadata for creating a ModelVersion."""
    name: str | None = None
    description: str | None = None
    author: str | None = None
    custom_properties: dict | None = None


class ModelArtifactMetadata(BaseModel):
    """Metadata for creating a ModelArtifact."""
    name: str | None = None
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
    custom_properties: dict | None = None


class ConfigMapMetadata(BaseModel):
    """Metadata from ConfigMap for creating model registry entries."""
    registered_model: RegisteredModelMetadata | None = None
    model_version: ModelVersionMetadata | None = None
    model_artifact: ModelArtifactMetadata | None = None

    @model_validator(mode='after')
    def validate_metadata_for_intent(self) -> 'ConfigMapMetadata':
        """Validate that metadata is compatible with the intent."""
        # This validation will be enhanced when we know the intent type
        return self


class AsyncUploadConfig(BaseModel):
    """Main configuration for the async upload job."""
    model_config = ConfigDict(
        # Allow extra fields for backward compatibility
        extra='forbid',
        # Validate assignments
        validate_assignment=True,
        # Populate by name (allows both snake_case and field names)
        populate_by_name=True
    )

    source: SourceConfig
    destination: DestinationConfig
    model: ModelConfig
    storage: StorageConfig = Field(default_factory=StorageConfig)
    registry: RegistryConfig
    metadata: ConfigMapMetadata | None = None  # Optional ConfigMap metadata
    signing_enabled: bool = False
    # If None, Signer defaults to /var/run/secrets/kubernetes.io/serviceaccount/token
    signing_identity_token_path: str | None = None
