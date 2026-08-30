"""E2E tests for MCP server filtering and error handling.

Tests filterQuery and negative cases.

To run these tests:
1. Start the catalog service with MCP test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/mcp_servers/test_filtering.py
"""

import os

import pytest

from model_catalog import CatalogAPIClient, CatalogAPIError, CatalogNotFoundError

_KIND_CLUSTER = os.getenv("KIND_CLUSTER", "True").lower() in ("true", "1", "yes")


class TestMCPServerFiltering:
    """Test suite for MCP server filterQuery functionality."""

    @pytest.mark.parametrize(
        "filter_query, expected_count, expected_names",
        [
            pytest.param(
                "provider='Math Community'",
                1,
                {"calculator"},
                id="single_provider",
            ),
            pytest.param(
                "provider IN ('Weather Community','Math Community')",
                2,
                {"weather-api", "calculator"},
                id="multiple_providers_in",
            ),
            pytest.param(
                "verifiedSource=true AND sast=true",
                1,
                {"calculator"},
                id="boolean_flags",
                marks=pytest.mark.skipif(
                    not _KIND_CLUSTER, reason="Non-KinD environments have additional MCP sources"
                ),
            ),
            pytest.param(
                "license='MIT' AND (provider='Math Community' OR provider='Weather Community')",
                2,
                {"weather-api", "calculator"},
                id="complex_and_or",
            ),
        ],
    )
    def test_filter_query(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        filter_query: str,
        expected_count: int,
        expected_names: set[str],
    ):
        """Test filterQuery with various filter expressions."""
        response = api_client.get_mcp_servers(filter_query=filter_query)
        items = response.get("items", [])
        assert len(items) == expected_count, (
            f"Expected {expected_count} servers for '{filter_query}', got {len(items)}"
        )
        assert {s["name"] for s in items} == expected_names


class TestMCPServerNegative:
    """Test suite for MCP server error handling and negative cases."""

    def test_get_nonexistent_server_404(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test that requesting a non-existent MCP server returns 404."""
        with pytest.raises(CatalogNotFoundError):
            api_client.get_mcp_server(server_id="99999")

    @pytest.mark.parametrize(
        "invalid_filter_query",
        [
            pytest.param(
                "provider IN ('test','value'))",
                id="malformed_syntax_unbalanced_parentheses",
            ),
            pytest.param(
                "provider = 'test' AND",
                id="malformed_syntax_incomplete_expression",
            ),
        ],
    )
    def test_invalid_filter_syntax(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        invalid_filter_query: str,
    ):
        """Test that invalid filterQuery syntax raises an error."""
        with pytest.raises(CatalogAPIError, match="invalid filter query"):
            api_client.get_mcp_servers(filter_query=invalid_filter_query)
