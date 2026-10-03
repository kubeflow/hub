"""Pytest fixtures for Catalog API fuzzing tests."""

import atexit
import logging
import os
import tempfile
from pathlib import Path

import pytest
import schemathesis
from schemathesis.specs.openapi.schemas import BaseOpenAPISchema

from tests.constants import CATALOG_URL, get_verify_ssl

logger = logging.getLogger("model-catalog.fuzz")

# Relative path from clients/python to openapi spec directory
OPENAPI_SPEC_REL_PATH = "../../../api/openapi"

# Track temp files for cleanup on process exit
_temp_files_to_cleanup: list[str] = []


def _register_temp_file_cleanup(temp_path: str) -> None:
    """Register a temp file for cleanup on process exit."""
    _temp_files_to_cleanup.append(temp_path)


def _cleanup_all_temp_files() -> None:
    """Clean up all registered temp files on process exit."""
    for temp_path in _temp_files_to_cleanup:
        try:
            if os.path.exists(temp_path):
                os.unlink(temp_path)
        except OSError:
            pass  # Best effort cleanup


# Register cleanup handler for process exit (handles SIGTERM but not SIGKILL)
atexit.register(_cleanup_all_temp_files)


@pytest.fixture(scope="session")
def verify_ssl() -> bool:
    """Get SSL verification setting from environment."""
    return get_verify_ssl(logger)


@pytest.fixture(scope="session")
def user_token() -> str | None:
    """Get user token from environment."""
    return os.getenv("AUTH_TOKEN")


@pytest.fixture
def auth_headers(user_token: str | None) -> dict[str, str]:
    """Provides authorization headers for API requests.

    Args:
        user_token: Bearer token from AUTH_TOKEN environment variable.

    Returns:
        Headers dict with Content-Type and optional Authorization.

    Raises:
        ValueError: If AUTH_TOKEN is set but empty or whitespace-only.
    """
    headers = {"Content-Type": "application/json"}
    if user_token is not None:
        if not isinstance(user_token, str) or not user_token.strip():
            msg = "AUTH_TOKEN must be a non-empty string"
            raise ValueError(msg)
        headers["Authorization"] = f"Bearer {user_token}"
    return headers


def _cleanup_temp_file(temp_path: str | None) -> None:
    """Clean up temporary file if it exists.

    Also removes the path from the atexit cleanup list to prevent stale entries.

    Args:
        temp_path: Path to temporary file, or None if no cleanup needed.
    """
    if temp_path is not None:
        try:
            os.unlink(temp_path)
            logger.debug("Cleaned up temporary config file: %s", temp_path)
            # Remove from atexit list to prevent stale entries
            if temp_path in _temp_files_to_cleanup:
                _temp_files_to_cleanup.remove(temp_path)
        except OSError as e:
            logger.warning("Failed to clean up temp file %s: %s", temp_path, e)


@pytest.fixture
def generated_schema(
    request: pytest.FixtureRequest,
    pytestconfig: pytest.Config,
    verify_ssl: bool,
) -> BaseOpenAPISchema:
    """Generate schema for the API based on the schema_file parameter.

    Args:
        request: Pytest fixture request containing optional schema_file parameter.
        pytestconfig: Pytest configuration object for accessing root path.
        verify_ssl: Whether to verify SSL certificates.

    Returns:
        BaseOpenAPISchema: The generated OpenAPI schema for testing.

    Raises:
        FileNotFoundError: If the schema file or config file cannot be found.
        schemathesis.SchemaError: If the schema is invalid.
    """
    schema_file = getattr(request, "param", "catalog-v1.yaml")
    os.environ["API_HOST"] = CATALOG_URL

    # Read and modify schemathesis.toml if verify_ssl is False
    toml_path = f"{pytestconfig.rootpath}/schemathesis.toml"
    config = schemathesis.config.SchemathesisConfig.from_path(toml_path)

    # tls-verify is by default true - modify config if SSL verification disabled
    temp_path: str | None = None
    try:
        if verify_ssl is False:
            with open(toml_path) as f:
                toml_content = f.read()

            modified_content = toml_content.replace("tls-verify = true", "tls-verify = false")

            # Create temp file with modified config
            with tempfile.NamedTemporaryFile(mode="w", suffix=".toml", delete=False) as temp_file:
                temp_file.write(modified_content)
                temp_file.flush()
                temp_path = temp_file.name
                # Register for cleanup on process exit (safety net for SIGTERM)
                _register_temp_file_cleanup(temp_path)
                logger.debug("Created temporary config file: %s", temp_path)

            config = schemathesis.config.SchemathesisConfig.from_path(temp_path)

        logger.info("Generating schema from %s with config %s", schema_file, config)
        openapi_path = Path(pytestconfig.rootpath) / OPENAPI_SPEC_REL_PATH / schema_file
        schema = schemathesis.openapi.from_path(
            path=str(openapi_path),
            config=config,
        )
        schema.config.output.sanitization.update(enabled=False)
        return schema
    except Exception as e:
        # Log temp file path to help debugging if schema generation fails
        if temp_path is not None:
            logger.error(
                "Schema generation failed. Temporary config was at: %s. Error: %s",
                temp_path,
                e,
            )
        raise
    finally:
        # Always clean up temp file, even if an exception occurred
        _cleanup_temp_file(temp_path)
