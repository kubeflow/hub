"""HTTP API client for catalog service.

This is a thin wrapper around the auto-generated OpenAPI client
to provide convenience methods and maintain compatibility with existing tests.

The generated client is in catalog_openapi/ (committed to git).
To regenerate after API changes: cd catalog/clients/python && make generate
"""

import json
import logging
from collections.abc import Callable
from functools import wraps
from typing import Any, TypeVar
from urllib.parse import quote

from catalog_openapi.models import OrderByField, SortOrder

logger = logging.getLogger(__name__)


def _encode_path_param(value: str) -> str:
    """URL-encode a path parameter for safe use in API URLs.

    Encodes ALL characters including slashes to prevent path traversal.
    Do not use for query parameters - use urllib.parse.urlencode instead.

    Args:
        value: The parameter value to encode.

    Returns:
        URL-encoded string safe for use in path segments.

    Raises:
        ValueError: If value contains path traversal patterns.
    """
    # Reject path traversal patterns before encoding
    if ".." in value:
        msg = f"Path traversal pattern '..' not allowed in path parameter: {value}"
        raise ValueError(msg)
    return quote(value, safe="")


def _as_str_list(value: str | list[str] | None) -> list[str] | None:
    """Normalize a single-value-or-list query param to a list, or None.

    An empty string or empty list is treated the same as None (no filter),
    matching how other list-style query params in this client behave.
    """
    if not value:
        return None
    if isinstance(value, str):
        return [value]
    return value


# Type variable for generic function return types
T = TypeVar("T")


class CatalogError(Exception):
    """Base exception for catalog client errors."""

    def __init__(self, message: str, cause: Exception | None = None):
        super().__init__(message)
        self.cause = cause


class CatalogConnectionError(CatalogError):
    """Raised when unable to connect to the catalog service."""


class CatalogAPIError(CatalogError):
    """Raised when the API returns an error response."""

    def __init__(
        self,
        message: str,
        status_code: int | None = None,
        cause: Exception | None = None,
    ):
        super().__init__(message, cause)
        self.status_code = status_code


class CatalogNotFoundError(CatalogAPIError):
    """Raised when a requested resource is not found (404)."""


class CatalogValidationError(CatalogAPIError):
    """Raised when request validation fails (400, 422)."""


def _handle_api_errors(func: Callable[..., T]) -> Callable[..., T]:
    """Decorator to handle API and network errors consistently.

    Re-raises programming errors (ImportError, ValueError, NotImplementedError,
    TypeError, AttributeError) unchanged. Converts expected runtime errors
    (network issues, API errors) to CatalogError hierarchy.
    """

    @wraps(func)
    def wrapper(*args: Any, **kwargs: Any) -> T:
        try:
            return func(*args, **kwargs)
        except (ImportError, NotImplementedError, TypeError, AttributeError):
            # Programming errors - re-raise unchanged
            raise
        except Exception as e:
            # Runtime errors (network, API, validation) - convert to CatalogError hierarchy
            _convert_exception(e)  # Always raises
            raise  # Unreachable, but satisfies mypy

    return wrapper


def _is_connection_error(e: Exception) -> bool:
    """Check if an exception is a connection-related error.

    Uses isinstance checks for known exception types from urllib3 and standard library.
    """
    # Known connection error types to check against
    connection_error_types: list[type] = [ConnectionError, TimeoutError, OSError]

    # Try to add urllib3 exceptions if available
    try:
        from urllib3.exceptions import (
            MaxRetryError,
            NewConnectionError,
        )
        from urllib3.exceptions import (
            TimeoutError as Urllib3TimeoutError,
        )

        connection_error_types.extend([MaxRetryError, NewConnectionError, Urllib3TimeoutError])
    except ImportError:
        pass

    # Try to add urllib exceptions if available
    try:
        from urllib.error import URLError

        connection_error_types.append(URLError)
    except ImportError:
        pass

    return isinstance(e, tuple(connection_error_types))


def _convert_exception(e: Exception) -> None:
    """Convert low-level exceptions to CatalogError hierarchy."""
    # Import here to avoid import errors if catalog_openapi not installed
    try:
        from catalog_openapi.exceptions import (
            ApiException,
            BadRequestException,
            NotFoundException,
            UnprocessableEntityException,
        )
    except ImportError:
        msg = f"Unexpected error: {e}"
        raise CatalogError(msg, cause=e) from e

    # Handle Pydantic validation errors (from OpenAPI client parameter validation)
    try:
        from pydantic import ValidationError as PydanticValidationError

        if isinstance(e, PydanticValidationError):
            msg = f"Validation error: {e}"
            raise CatalogValidationError(msg, cause=e) from e
    except ImportError:
        pass

    # Handle API exceptions
    if isinstance(e, NotFoundException):
        # Include both reason and body for complete error details
        msg = f"Resource not found: {e.body if e.body else e.reason}"
        raise CatalogNotFoundError(msg, status_code=e.status, cause=e) from e

    if isinstance(e, (BadRequestException, UnprocessableEntityException)):
        # Include both reason and body for complete error details
        msg = f"Validation error: {e.body if e.body else e.reason}"
        raise CatalogValidationError(msg, status_code=e.status, cause=e) from e

    if isinstance(e, ApiException):
        # Include both reason and body for complete error details
        msg = f"API error ({e.status}): {e.body if e.body else e.reason}"
        raise CatalogAPIError(msg, status_code=e.status, cause=e) from e

    # Handle network/connection errors using proper isinstance checks
    if _is_connection_error(e):
        msg = f"Failed to connect to catalog service: {e}"
        raise CatalogConnectionError(msg, cause=e) from e

    # Re-raise unknown exceptions wrapped in CatalogError
    logger.debug("Converting unexpected exception: %s", e, exc_info=True)
    msg = f"Unexpected error: {e}"
    raise CatalogError(msg, cause=e) from e


class CatalogAPIClient:
    """Wrapper for catalog API HTTP requests.

    This wraps the auto-generated OpenAPI client to provide:
    - Simpler interface for common operations
    - Compatibility with existing test code
    - Convenience methods for extracting data
    """

    # Maximum allowed timeout (5 minutes)
    MAX_TIMEOUT = 300

    def __init__(self, base_url: str, timeout: int = 10, verify_ssl: bool = True, access_token: str = None):
        """Initialize API client.

        Args:
            base_url: Base URL of the catalog service (e.g., http://localhost:8081)
            timeout: Request timeout in seconds (must be positive, max 300)
            verify_ssl: Whether to verify SSL certificates (default True)
            access_token: Access token for authentication (default None)

        Raises:
            ValueError: If base_url is empty or invalid, or timeout is not positive.
            ImportError: If the generated client is not installed.
        """
        self._validate_base_url(base_url)
        self._validate_timeout(timeout)

        try:
            from catalog_openapi import ApiClient, Configuration
            from catalog_openapi.api.mcp_catalog_service_api import MCPCatalogServiceApi
            from catalog_openapi.api.model_catalog_service_api import ModelCatalogServiceApi
            from catalog_openapi.api.skill_catalog_service_api import SkillCatalogServiceApi
        except ImportError as e:
            msg = (
                "Generated catalog_openapi client not found. "
                "Run 'make generate && poetry install' to generate it. "
                f"Original error: {e}"
            )
            raise ImportError(msg) from e

        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.verify_ssl = verify_ssl
        self.access_token = access_token

        # Configure the generated client
        config = Configuration(host=self.base_url, access_token=self.access_token)
        config.verify_ssl = verify_ssl
        self.api_client = ApiClient(configuration=config)
        self._configure_timeout(config, timeout)
        self.catalog_api = ModelCatalogServiceApi(self.api_client)
        self.mcp_api = MCPCatalogServiceApi(self.api_client)
        self.skill_api = SkillCatalogServiceApi(self.api_client)

    def _validate_base_url(self, base_url: str) -> None:
        """Validate the base URL parameter."""
        if not base_url or not isinstance(base_url, str):
            msg = "base_url must be a non-empty string"
            raise ValueError(msg)
        if not base_url.startswith(("http://", "https://")):
            msg = f"base_url must start with http:// or https://, got: {base_url}"
            raise ValueError(msg)

    def _validate_timeout(self, timeout: int) -> None:
        """Validate the timeout parameter."""
        if not isinstance(timeout, int) or timeout <= 0 or timeout > self.MAX_TIMEOUT:
            msg = f"timeout must be a positive integer <= {self.MAX_TIMEOUT}, got: {timeout}"
            raise ValueError(msg)

    def _configure_timeout(self, config: Any, timeout: int) -> None:
        """Configure request timeout on the API client.

        Tries multiple approaches for compatibility with different urllib3 versions.
        Logs a warning if timeout cannot be configured.
        """
        # Try setting timeout on pool manager (urllib3)
        try:
            self.api_client.rest_client.pool_manager.connection_pool_kw["timeout"] = timeout
            return
        except (AttributeError, KeyError):
            pass

        # Fallback: set timeout on configuration (used by some client versions)
        try:
            config.timeout = timeout
            return
        except AttributeError:
            pass

        logger.warning(
            "Could not configure request timeout. Requests may hang indefinitely. "
            "Consider upgrading urllib3 or the generated client."
        )

    @_handle_api_errors
    def get_sources(
        self,
        asset_type: str | None = None,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get catalog sources.

        Args:
            asset_type: Filter by asset type ('models' or 'mcp_servers').
            page_size: Number of items per page.
            next_page_token: Token for pagination.

        Returns:
            Dict with sources list and pagination info.
        """
        page_size_str = str(page_size) if page_size is not None else None

        # Build query params — always use the manual path so assetType and other
        # new fields are preserved in the raw JSON (the generated Pydantic model
        # does not yet include assetType and would silently drop it).
        query_params: list[tuple[str, str]] = []
        if asset_type is not None:
            query_params.append(("assetType", asset_type))
        if page_size_str is not None:
            query_params.append(("pageSize", page_size_str))
        if next_page_token is not None:
            query_params.append(("nextPageToken", next_page_token))

        _param = self.api_client.param_serialize(
            method="GET",
            resource_path="/api/model_catalog/v1alpha1/sources",
            query_params=query_params,
            header_params={"Accept": "application/json"},
            auth_settings=["Bearer"],
        )
        response_data = self.api_client.call_api(*_param)
        response_data.read()

        # Let the generated client handle error status codes (raises ApiException).
        self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map={"200": "CatalogSourceList", "400": "Error", "401": "Error", "500": "Error"},
        )

        # Return the raw JSON to preserve fields not yet in the Pydantic model.
        return json.loads(response_data.data)

    @_handle_api_errors
    def get_source_by_id(self, source_id: str) -> dict[str, Any]:
        """Get source by ID.

        Args:
            source_id: The source identifier.

        Returns:
            Dict with source details.

        Raises:
            CatalogNotFoundError: If no source matches the given ID.
        """
        # API doesn't have get_source, so we filter find_sources by name
        response = self.catalog_api.find_sources(name=source_id)
        sources = response.to_dict()
        items = sources.get("items", [])

        for source in items:
            if source.get("name") == source_id:
                return source

        msg = f"Source not found: {source_id}"
        raise CatalogNotFoundError(msg)

    @_handle_api_errors
    def get_labels(
        self,
        asset_type: str | None = None,
        page_size: int | None = None,
        order_by: str | None = None,
        sort_order: str | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get catalog labels.

        Args:
            asset_type: Filter by asset type ('models' or 'mcp_servers').
            page_size: Number of items per page.
            order_by: Key to order labels by.
            sort_order: Sort order ('ASC' or 'DESC').
            next_page_token: Token for pagination.

        Returns:
            Dict with labels list and pagination info.
        """
        page_size_str = str(page_size) if page_size is not None else None

        # Build query params — use the manual path so assetType is preserved
        # in the raw JSON (the generated Pydantic model does not yet include
        # assetType and would silently drop it).
        query_params: list[tuple[str, str]] = []
        if asset_type is not None:
            query_params.append(("assetType", asset_type))
        if page_size_str is not None:
            query_params.append(("pageSize", page_size_str))
        if order_by is not None:
            query_params.append(("orderBy", order_by))
        if sort_order is not None:
            query_params.append(("sortOrder", sort_order))
        if next_page_token is not None:
            query_params.append(("nextPageToken", next_page_token))

        _param = self.api_client.param_serialize(
            method="GET",
            resource_path="/api/model_catalog/v1alpha1/labels",
            query_params=query_params,
            header_params={"Accept": "application/json"},
            auth_settings=["Bearer"],
        )
        response_data = self.api_client.call_api(*_param)
        response_data.read()

        # Let the generated client handle error status codes (raises ApiException).
        self.api_client.response_deserialize(
            response_data=response_data,
            response_types_map={"200": "CatalogLabelList", "400": "Error", "401": "Error", "500": "Error"},
        )

        # Return the raw JSON to preserve fields not yet in the Pydantic model.
        return json.loads(response_data.data)

    @_handle_api_errors
    def get_models(
        self,
        source: str | None = None,
        q: str | None = None,
        filter_query: str | None = None,
        order_by: str | None = None,
        sort_order: str | None = None,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get models from catalog.

        Args:
            source: Filter by source ID.
            q: Free-form keyword search to filter the response.
            filter_query: Filter query string.
            order_by: Field to order by (NAME, CREATE_TIME, ACCURACY, etc.).
            sort_order: Sort order (ASC or DESC).
            page_size: Number of items per page.
            next_page_token: Token for pagination.

        Returns:
            Dict with models response.
        """
        source_list = [source] if source else None
        page_size_str = str(page_size) if page_size is not None else None

        # Convert strings to proper enum types
        order_by_enum: OrderByField | None = None
        if order_by:
            order_by_enum = OrderByField(order_by.upper())

        sort_order_enum: SortOrder | None = None
        if sort_order:
            sort_order_enum = SortOrder(sort_order.upper())

        response = self.catalog_api.find_models(
            source=source_list,
            q=q,
            filter_query=filter_query,
            order_by=order_by_enum,
            sort_order=sort_order_enum,
            page_size=page_size_str,
            next_page_token=next_page_token,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_artifacts(
        self,
        source_id: str,
        model_name: str,
        artifact_type: str | list[str] | None = None,
        filter_query: str | None = None,
        order_by: str | None = None,
        sort_order: str | None = None,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get artifacts for a model.

        Args:
            source_id: The source ID containing the model.
            model_name: The model name.
            artifact_type: Optional artifact type(s) to filter by.
                Accepts "model-artifact", "metrics-artifact", or a list of these values.
                Can be a single string or list of strings.
            filter_query: Optional filter query.
            order_by: Optional field to order by (ID, NAME, CREATE_TIME, LAST_UPDATE_TIME,
                or custom property path like "accuracy.double_value").
            sort_order: Optional sort order (ASC or DESC).
            page_size: Optional page size.
            next_page_token: Optional pagination token.

        Returns:
            Dict with artifacts response.
        """
        # Convert artifact_type to list format expected by OpenAPI client
        artifact_type_list = None
        if artifact_type is not None:
            if isinstance(artifact_type, str):
                artifact_type_list = [artifact_type]
            else:
                artifact_type_list = artifact_type

        sort_order_enum: SortOrder | None = None
        if sort_order:
            sort_order_enum = SortOrder(sort_order.upper())

        page_size_str = str(page_size) if page_size is not None else None
        response = self.catalog_api.get_all_model_artifacts(
            source_id=_encode_path_param(source_id),
            model_name=_encode_path_param(model_name),
            artifact_type=artifact_type_list,
            filter_query=filter_query,
            order_by=order_by,
            sort_order=sort_order_enum,
            page_size=page_size_str,
            next_page_token=next_page_token,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_model(self, source: str, model_name: str) -> dict[str, Any]:
        """Get a specific model from a source.

        Args:
            source: The source ID.
            model_name: The model name.

        Returns:
            Dict with model details.
        """
        response = self.catalog_api.get_model(
            source_id=_encode_path_param(source),
            model_name=_encode_path_param(model_name),
        )
        return response.to_dict()

    @_handle_api_errors
    def get_model_artifacts_with_params(
        self,
        source: str,
        model_name: str,
        params: dict[str, Any],
    ) -> dict[str, Any]:
        """Get artifacts for a specific model with arbitrary query parameters.

        Args:
            source: The source ID.
            model_name: The model name.
            params: Additional query parameters to pass to the API.

        Returns:
            Dict with artifacts list and pagination info.
        """
        response = self.catalog_api.get_all_model_artifacts(
            source_id=_encode_path_param(source),
            model_name=_encode_path_param(model_name),
            **params,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_filter_options(self) -> dict[str, Any]:
        """Get available filter options.

        Returns:
            Dict with available filter fields and their options.
        """
        response = self.catalog_api.find_models_filter_options()
        return response.to_dict()

    @_handle_api_errors
    def get_mcp_servers(
        self,
        name: str | None = None,
        q: str | None = None,
        named_query: str | None = None,
        filter_query: str | None = None,
        order_by: str | None = None,
        sort_order: str | None = None,
        include_tools: bool | None = None,
        tool_limit: int | None = None,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get MCP servers from catalog.

        Args:
            name: Filter by server name.
            q: Keyword search query.
            named_query: Predefined filter template name to apply (e.g. "production_ready").
            filter_query: SQL-like filter expression (e.g. "provider='Math Community'").
            order_by: Field to order results by (e.g. "CREATE_TIME", "LAST_UPDATE_TIME", "NAME").
            sort_order: Sort direction ("ASC" or "DESC").
            include_tools: Whether to include the tools array in each server result.
            tool_limit: Maximum number of tools to include per server.
            page_size: Number of items per page.
            next_page_token: Token for pagination.

        Returns:
            Dict with MCP servers list and pagination info.
        """
        page_size_str = str(page_size) if page_size is not None else None
        order_by_enum: OrderByField | None = None
        if order_by:
            order_by_enum = OrderByField(order_by.upper())
        sort_order_enum: SortOrder | None = None
        if sort_order:
            sort_order_enum = SortOrder(sort_order.upper())
        response = self.mcp_api.find_mcp_servers(
            name=name,
            q=q,
            named_query=named_query,
            filter_query=filter_query,
            order_by=order_by_enum,
            sort_order=sort_order_enum,
            include_tools=include_tools,
            tool_limit=tool_limit,
            page_size=page_size_str,
            next_page_token=next_page_token,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_mcp_server(self, server_id: str, include_tools: bool | None = None) -> dict[str, Any]:
        """Get a specific MCP server by ID.

        Args:
            server_id: The MCP server identifier.
            include_tools: Whether to include the tools array in the result.

        Returns:
            Dict with MCP server details.
        """
        response = self.mcp_api.get_mcp_server(
            server_id=server_id,
            include_tools=include_tools,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_mcp_server_tools(
        self,
        server_id: str,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get tools for a specific MCP server.

        Args:
            server_id: The MCP server identifier.
            page_size: Number of items per page.
            next_page_token: Token for pagination.

        Returns:
            Dict with MCP server tools list and pagination info.
        """
        page_size_str = str(page_size) if page_size is not None else None
        response = self.mcp_api.find_mcp_server_tools(
            server_id=server_id,
            page_size=page_size_str,
            next_page_token=next_page_token,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_skills(
        self,
        name: str | None = None,
        q: str | None = None,
        source: str | list[str] | None = None,
        source_label: str | list[str] | None = None,
        filter_query: str | None = None,
        order_by: str | None = None,
        sort_order: str | None = None,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Get skills from the skill catalog.

        Args:
            name: Filter by skill name using SQL LIKE pattern matching.
            q: Free-form keyword search across name, description, and readme.
            source: Filter by source ID(s). Accepts a single ID or a list.
            source_label: Filter by source label(s). Accepts a single label or a
                list. Use "null" to match sources with no label.
            filter_query: SQL-like filter expression (e.g. "category='devops'").
            order_by: Field to order results by (e.g. "CREATE_TIME", "NAME").
            sort_order: Sort direction ("ASC" or "DESC").
            page_size: Number of items per page.
            next_page_token: Token for pagination.

        Returns:
            Dict with skills list and pagination info.
        """
        source_list = _as_str_list(source)
        source_label_list = _as_str_list(source_label)
        page_size_str = str(page_size) if page_size is not None else None

        order_by_enum: OrderByField | None = None
        if order_by:
            order_by_enum = OrderByField(order_by.upper())
        sort_order_enum: SortOrder | None = None
        if sort_order:
            sort_order_enum = SortOrder(sort_order.upper())

        response = self.skill_api.find_skills(
            name=name,
            q=q,
            source=source_list,
            source_label=source_label_list,
            filter_query=filter_query,
            order_by=order_by_enum,
            sort_order=sort_order_enum,
            page_size=page_size_str,
            next_page_token=next_page_token,
        )
        return response.to_dict()

    @_handle_api_errors
    def get_skill(self, skill_id: str) -> dict[str, Any]:
        """Get a specific skill by ID.

        Args:
            skill_id: The skill identifier.

        Returns:
            Dict with skill details.
        """
        response = self.skill_api.get_skill(id=_encode_path_param(skill_id))
        return response.to_dict()

    def get_named_queries(self, source: str | None = None) -> dict[str, Any]:
        """Get named queries.

        Args:
            source: Filter by source ID.

        Returns:
            Dict with named queries.

        Raises:
            NotImplementedError: This endpoint is not available in the current API.
        """
        msg = "Named queries endpoint is not implemented in the current API version"
        raise NotImplementedError(msg)

    def health_check(self) -> bool:
        """Check if service is healthy.

        Returns:
            True if service is responding, False otherwise.
        """
        try:
            self.get_sources()
            return True
        except CatalogError:
            return False

    @_handle_api_errors
    def preview_source(
        self,
        config_content: str,
        catalog_data: str | None = None,
        filter_status: str | None = None,
        page_size: int | None = None,
        next_page_token: str | None = None,
    ) -> dict[str, Any]:
        """Preview a source configuration.

        Args:
            config_content: YAML string containing the source configuration (required).
            catalog_data: Optional YAML string containing catalog data (models).
            filter_status: Filter response by status ('all', 'included', 'excluded').
            page_size: Number of items per page.
            next_page_token: Token for pagination.

        Returns:
            Dict with preview response including models and summary.

        Raises:
            ValueError: If config_content is empty or filter_status is invalid.
            CatalogConnectionError: If unable to connect to the service.
            CatalogValidationError: If the config is invalid (400/422).
            CatalogAPIError: For other API errors.
        """
        # Validate required parameters
        if not config_content or not isinstance(config_content, str):
            msg = "config_content must be a non-empty string"
            raise ValueError(msg)

        valid_filter_statuses = {"all", "included", "excluded", None}
        if filter_status not in valid_filter_statuses:
            msg = f"filter_status must be one of {valid_filter_statuses - {None}}, got: {filter_status}"
            raise ValueError(msg)

        # Convert strings to bytes for the generated client
        config_bytes = config_content.encode("utf-8")
        catalog_data_bytes = catalog_data.encode("utf-8") if catalog_data else None
        page_size_str = str(page_size) if page_size is not None else None

        response = self.catalog_api.preview_catalog_source(
            config=config_bytes,
            catalog_data=catalog_data_bytes,
            filter_status=filter_status,
            page_size=page_size_str,
            next_page_token=next_page_token,
        )
        return response.to_dict()

    # Convenience methods for extracting data

    def extract_model_names(self, response: dict[str, Any]) -> list[str]:
        """Extract model names from API response.

        Args:
            response: API response dict containing 'items' list.

        Returns:
            List of model names.
        """
        return [item["name"] for item in response.get("items", [])]

    def extract_artifact_properties(self, response: dict[str, Any], property_name: str) -> list[Any]:
        """Extract artifact property values from API response.

        Args:
            response: API response dict containing 'items' list.
            property_name: Name of the custom property to extract.

        Returns:
            List of property values found across all artifacts.
        """
        values = []
        for item in response.get("items", []):
            for artifact in item.get("artifacts", []):
                props = artifact.get("customProperties", {})
                if property_name in props:
                    values.append(props[property_name])
        return values

    def close(self) -> None:
        """Close the client and release all resources.

        Closes the underlying connection pool to free up sockets.
        After calling close(), the client should not be used.
        Logs warnings if cleanup fails but does not raise exceptions.
        """
        # Close the generated API client if it has a close method
        try:
            if hasattr(self.api_client, "close"):
                self.api_client.close()
        except Exception:
            logger.warning("Failed to close API client", exc_info=True)

        # Close the REST client's connection pool
        try:
            rest_client = getattr(self.api_client, "rest_client", None)
            if rest_client is not None:
                pool_manager = getattr(rest_client, "pool_manager", None)
                if pool_manager is not None:
                    pool_manager.clear()
        except Exception:
            logger.warning("Failed to clear connection pool", exc_info=True)

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit.

        Closes the client resources. Does not suppress exceptions.
        """
        self.close()
