"""E2E tests for MCP server catalog functionality.

Tests MCP server listing, retrieval by ID, tools, and custom properties.

To run these tests:
1. Start the catalog service with MCP test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/mcp_servers/test_servers.py
"""

import random

from model_catalog import CatalogAPIClient
from tests.constants import MCP_SERVER_REQUIRED_FIELDS


class TestMCPServerBasics:
    """Test suite for basic MCP server listing and retrieval."""

    def test_mcp_servers_loaded(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        test_mcp_catalog_data: dict,
        kind_cluster: bool,
    ):
        """Test that all expected MCP servers are loaded with required fields."""
        response = api_client.get_mcp_servers(page_size=None if kind_cluster else 100)
        items = response.get("items", [])
        actual_names = {server["name"] for server in items}
        expected_names = {server["name"] for server in test_mcp_catalog_data["mcp_servers"]}
        if kind_cluster:
            assert actual_names == expected_names
        else:
            assert set(expected_names).issubset(set(actual_names))
        for server in items:
            missing = MCP_SERVER_REQUIRED_FIELDS - server.keys()
            assert not missing, f"Server '{server.get('name')}' missing fields: {missing}"

    def test_mcp_server_providers(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        test_mcp_catalog_data: dict,
        kind_cluster: bool,
    ):
        """Test that MCP server providers match expected values."""
        response = api_client.get_mcp_servers(page_size=None if kind_cluster else 100)
        actual_providers = {server["name"]: server["provider"] for server in response["items"]}
        expected_providers = {server["name"]: server["provider"] for server in test_mcp_catalog_data["mcp_servers"]}
        if kind_cluster:
            assert actual_providers == expected_providers
        else:
            assert set(expected_providers.items()).issubset(set(actual_providers.items()))

    def test_mcp_server_get_by_id(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        kind_cluster: bool,
    ):
        """Test that an MCP server can be retrieved by ID with all required fields."""
        response = api_client.get_mcp_servers(page_size=None if kind_cluster else 100)
        assert response.get("items"), "No MCP servers found"
        server = random.choice(response["items"])
        single = api_client.get_mcp_server(server_id=server["id"])
        assert single["name"] == server["name"]

        missing = MCP_SERVER_REQUIRED_FIELDS - single.keys()
        assert not missing, f"Server '{single.get('name')}' missing fields: {missing}"

    def test_mcp_server_pagination(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        test_mcp_catalog_data: dict,
        kind_cluster: bool,
    ):
        """Test pagination by iterating all servers with pageSize=1."""
        expected_names = {server["name"] for server in test_mcp_catalog_data["mcp_servers"]}
        collected_names: set[str] = set()
        next_token = None

        # Use a safe upper bound: on non-KinD environments there may be more servers
        max_pages = len(expected_names) + 1 if kind_cluster else 100
        for _ in range(max_pages):
            response = api_client.get_mcp_servers(page_size=1, next_page_token=next_token)
            items = response.get("items", [])
            assert len(items) == 1, f"Expected exactly 1 item per page, got {len(items)}"

            name = items[0]["name"]
            assert name not in collected_names, f"Duplicate server: {name}"
            collected_names.add(name)

            next_token = response.get("nextPageToken")
            if not next_token:
                break

        if kind_cluster:
            assert collected_names == expected_names
        else:
            assert expected_names.issubset(collected_names)

    def test_filter_by_name(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test filtering MCP servers by exact name."""
        target_name = "calculator"
        response = api_client.get_mcp_servers(name=target_name)
        items = response.get("items", [])
        assert len(items) == 1, f"Expected 1 server named '{target_name}', got {len(items)}"
        assert items[0]["name"] == target_name


class TestMCPServerTools:
    """Test suite for MCP server tools functionality."""

    def test_tool_count_without_include(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        test_mcp_catalog_data: dict,
        kind_cluster: bool,
    ):
        """Test that toolCount reflects actual tools even without includeTools."""
        response = api_client.get_mcp_servers(page_size=None if kind_cluster else 100)
        expected_counts = {s["name"]: len(s.get("tools", [])) for s in test_mcp_catalog_data["mcp_servers"]}
        for server in response.get("items", []):
            name = server["name"]
            if name in expected_counts:
                assert server.get("toolCount", 0) == expected_counts[name], (
                    f"Server '{name}': expected toolCount {expected_counts[name]}, got {server.get('toolCount', 0)}"
                )

    def test_tools_included_when_requested(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        test_mcp_catalog_data: dict,
        kind_cluster: bool,
    ):
        """Test that tools are returned when include_tools=True."""
        response = api_client.get_mcp_servers(include_tools=True, page_size=None if kind_cluster else 100)
        expected_tools = {
            s["name"]: [t["name"] for t in s.get("tools", [])] for s in test_mcp_catalog_data["mcp_servers"]
        }
        for server in response.get("items", []):
            name = server["name"]
            if name in expected_tools:
                actual_tool_names = [t["name"] for t in server.get("tools", [])]
                assert sorted(actual_tool_names) == sorted(expected_tools[name])


class TestMCPServerCustomProperties:
    """Test suite for MCP server custom properties."""

    def test_custom_properties_loaded(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        test_mcp_catalog_data: dict,
        kind_cluster: bool,
    ):
        """Test that customProperties are correctly loaded from YAML."""
        response = api_client.get_mcp_servers(page_size=None if kind_cluster else 100)
        servers_by_name = {s["name"]: s for s in response.get("items", [])}

        for yaml_server in test_mcp_catalog_data["mcp_servers"]:
            name = yaml_server["name"]
            expected_props = yaml_server.get("customProperties")
            if expected_props:
                assert name in servers_by_name
                assert servers_by_name[name].get("customProperties") == expected_props
