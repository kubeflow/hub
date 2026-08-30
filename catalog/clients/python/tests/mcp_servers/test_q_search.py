"""E2E tests for MCP server keyword search functionality.

Tests the 'q' search parameter for text-based MCP server search
across name, description, and provider fields.

To run these tests:
1. Start the catalog service with MCP test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/mcp_servers/test_q_search.py
"""

import pytest

from model_catalog import CatalogAPIClient

SEARCHABLE_FIELDS = ("name", "description", "provider")


class TestMCPServerSearch:
    """TC-API-012: Tests for MCP server keyword search via q parameter."""

    @pytest.mark.parametrize(
        "search_term, expected_names",
        [
            pytest.param("calculator", {"calculator"}, id="match_by_name"),
            pytest.param("File system management", {"file-manager"}, id="match_by_description"),
            pytest.param("CloudMCP", {"remote-http-server"}, id="match_by_provider"),
            pytest.param("calc", {"calculator"}, id="partial_prefix"),
            pytest.param("manager", {"file-manager"}, id="partial_suffix"),
            pytest.param("system", {"file-manager"}, id="partial_middle"),
            pytest.param("CALCULATOR", {"calculator"}, id="case_upper"),
            pytest.param("Calculator", {"calculator"}, id="case_mixed"),
            pytest.param("nonexistent_keyword_xyz_99999", set(), id="no_results"),
        ],
    )
    def test_keyword_search(
        self,
        search_term: str,
        expected_names: set[str],
        api_client: CatalogAPIClient,
        test_mcp_catalog_data: dict,
        suppress_ssl_warnings: None,
        kind_cluster: bool,
    ):
        """Test keyword search across name, description, provider fields.

        Validates: single-field matching, partial matching (prefix/suffix/middle),
        case-insensitive matching, and non-matching query returns empty.
        """
        if not kind_cluster:
            pytest.skip("Test data validation requires Kind cluster")

        response = api_client.get_mcp_servers(q=search_term)
        items = response.get("items", [])
        actual_names = {s["name"] for s in items}

        expected_from_data = _get_expected_server_names(test_mcp_catalog_data, search_term)
        assert actual_names == expected_from_data, (
            f"q='{search_term}': expected {sorted(expected_from_data)}, got {sorted(actual_names)}"
        )

    @pytest.mark.parametrize("search_term", ["", None])
    def test_keyword_search_empty_query(
        self,
        search_term: str | None,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        kind_cluster: bool,
    ):
        """Test that empty or None q parameter returns all servers."""
        response_q = api_client.get_mcp_servers(q=search_term, page_size=None if kind_cluster else 100)
        response_all = api_client.get_mcp_servers(page_size=None if kind_cluster else 100)

        names_q = {s["name"] for s in response_q.get("items", [])}
        names_all = {s["name"] for s in response_all.get("items", [])}
        assert names_q == names_all, f"q={search_term!r} returned {sorted(names_q)}, expected all: {sorted(names_all)}"

    @pytest.mark.parametrize(
        "search_term",
        [
            pytest.param("test' OR '1'='1", id="sql_injection_single_quote"),
            pytest.param("test; DROP TABLE", id="sql_injection_drop"),
            pytest.param("test<script>alert(1)</script>", id="xss"),
            pytest.param("test@#$%^&*()", id="special_chars"),
        ],
    )
    def test_keyword_search_special_characters(
        self,
        search_term: str,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """TC-ERROR-007: Test that special characters in search query are handled safely."""
        response = api_client.get_mcp_servers(q=search_term)
        assert "items" in response


def _get_expected_server_names(catalog_data: dict, search_term: str) -> set[str]:
    """Get MCP server names that should match the search term from test data."""
    term_lower = search_term.lower()
    return {
        server["name"]
        for server in catalog_data.get("mcp_servers", [])
        if any(term_lower in server.get(field, "").lower() for field in SEARCHABLE_FIELDS)
    }
