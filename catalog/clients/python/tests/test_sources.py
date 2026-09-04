"""E2E tests for source functionality.

Tests source status, paths, merge, and required fields.
Consolidated from test_source_status.py, test_source_paths.py,
test_source_merge.py, and test_sources_required_items.py.

To run these tests:
1. Start the catalog service with test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/test_sources.py
"""

import pytest

from model_catalog import CatalogAPIClient


class TestSourceBasics:
    """Test suite for basic source functionality."""

    def test_get_sources_returns_response(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that getting sources returns a response with items."""
        response = api_client.get_sources()
        assert isinstance(response, dict)
        assert "items" in response
        assert isinstance(response["items"], list)

    def test_sources_exist(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that at least one source is configured."""
        response = api_client.get_sources()
        assert len(response["items"]) > 0, "Expected at least one source"

    def test_source_required_fields(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that sources have all required fields."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"

        for source in response["items"]:
            # Required: identifier
            assert "id" in source or "name" in source, "Source missing identifier"
            # Required: status indicator
            assert "enabled" in source or "status" in source, f"Source {source.get('id')} missing status indicator"

    def test_source_ids_are_unique(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that sources have unique IDs."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"

        source_ids = [s.get("id") for s in response["items"]]
        unique_ids = set(source_ids)

        assert len(source_ids) == len(unique_ids), f"Duplicate source IDs found: {source_ids}"

    def test_source_pagination_fields(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that sources response includes pagination fields."""
        response = api_client.get_sources()
        # Response should have pagination info (size or pageSize)
        has_pagination = "size" in response or "pageSize" in response
        assert has_pagination, f"Response missing pagination fields. Keys: {response.keys()}"


class TestSourceStatus:
    """Test suite for source status functionality."""

    def test_sources_have_status_field(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that sources have a status field."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"

        for source in response["items"]:
            assert isinstance(source, dict)
            assert "status" in source or "enabled" in source

    def test_enabled_source_returns_models(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that enabled sources return models."""
        sources = api_client.get_sources()
        assert sources.get("items"), "No sources found"

        # Find an enabled source that is available (not in error state)
        enabled_source = next(
            (s for s in sources["items"] if s.get("enabled") is True and s.get("status") != "error"),
            None,
        )

        if not enabled_source:
            enabled_source = next(
                (s for s in sources["items"] if s.get("status") == "available"),
                None,
            )

        assert enabled_source, "No enabled source found"

        source_id = enabled_source.get("id") or enabled_source.get("name")
        assert source_id, "Source has no identifier"

        models = api_client.get_models(source=source_id)
        assert isinstance(models, dict)
        assert "items" in models
        assert len(models.get("items", [])) > 0, f"Expected models from enabled source {source_id}"

    def test_disabled_source_excluded_from_models(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that disabled sources are excluded from model results."""
        sources = api_client.get_sources()
        if not sources.get("items"):
            pytest.skip("No sources found in test data")

        # Find a disabled source
        disabled_source = next(
            (s for s in sources["items"] if s.get("enabled") is False),
            None,
        )

        if not disabled_source:
            disabled_source = next(
                (s for s in sources["items"] if s.get("status") == "disabled"),
                None,
            )

        if not disabled_source:
            pytest.skip("Test requires a disabled source in test data")

        disabled_source_id = disabled_source.get("id") or disabled_source.get("name")
        assert disabled_source_id, "Disabled source has no identifier"

        # Get all models
        all_models = api_client.get_models()
        assert isinstance(all_models, dict)

        # None of the models should be from the disabled source
        for model in all_models.get("items", []):
            model_source = model.get("sourceId")
            assert model_source != disabled_source_id, (
                f"Model {model.get('name')} from disabled source {disabled_source_id} should not appear in results"
            )

    def test_source_status_values(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that source status has expected values."""
        sources = api_client.get_sources()
        assert sources.get("items"), "No sources found"

        for source in sources["items"]:
            status = source.get("status")
            enabled = source.get("enabled")

            assert status is not None or enabled is not None

            if status:
                valid_statuses = {"available", "partially-available", "disabled", "error", "loading", "pending"}
                assert status in valid_statuses, f"Unexpected status: {status}"

    def test_enabled_and_status_consistency(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that enabled flag and status are consistent."""
        sources = api_client.get_sources()
        assert sources.get("items"), "No sources found"

        for source in sources["items"]:
            enabled = source.get("enabled")
            status = source.get("status")

            if enabled is not None and status is not None:
                if enabled is True:
                    assert status != "disabled", f"Source {source.get('id')} has enabled=True but status=disabled"
                if enabled is False:
                    assert status == "disabled", f"Source {source.get('id')} has enabled=False but status={status}"

    def test_disabled_sources_in_response(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that disabled sources appear in response with correct status."""
        response = api_client.get_sources()
        if not response.get("items"):
            pytest.skip("No sources found in test data")

        disabled_sources = [s for s in response["items"] if s.get("enabled") is False or s.get("status") == "disabled"]

        if not disabled_sources:
            pytest.skip("Test requires at least one disabled source in test data")

        for source in disabled_sources:
            assert source.get("id") or source.get("name"), "Disabled source missing identifier"
            if "status" in source:
                assert source["status"] == "disabled"
            if "enabled" in source:
                assert source["enabled"] is False


class TestSourcePaths:
    """Test suite for source path configurations."""

    def test_sources_are_loaded(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that sources configured with paths are properly loaded."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"
        assert len(response["items"]) > 0

    def test_source_status_indicates_path_resolution(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that source status indicates whether paths were resolved correctly."""
        response = api_client.get_sources()

        for source in response.get("items", []):
            status = source.get("status")
            error = source.get("error")

            # If status is error, there should be an error message
            if status == "error":
                assert error is not None, f"Source {source.get('id')} has error status but no error message"

    def test_available_sources_exist(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that available sources exist (paths resolved correctly)."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"

        available_sources = [s for s in response["items"] if s.get("status") == "available"]
        assert len(available_sources) > 0, "No available sources found"

    def test_models_from_path_configured_sources(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that models are loaded from path-configured sources."""
        sources = api_client.get_sources()
        models = api_client.get_models()

        available_sources = [s for s in sources.get("items", []) if s.get("status") == "available"]

        if available_sources:
            assert models.get("items"), "Expected models from available sources"

    def test_source_error_messages_are_helpful(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that source error messages indicate path issues."""
        response = api_client.get_sources()

        for source in response.get("items", []):
            if source.get("status") == "error":
                error = source.get("error", "")
                assert error, f"Source {source.get('id')} has error status but empty error"
                assert isinstance(error, str)
                assert len(error) > 0

    def test_source_error_field_consistency(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that error field is consistent with source status."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"

        for source in response["items"]:
            status = source.get("status")
            error = source.get("error")
            source_id = source.get("id") or source.get("name")

            # Available and disabled sources should not have errors
            if status in ["available", "disabled"]:
                assert error is None or error == "", (
                    f"Source '{source_id}' with status '{status}' should not have error, got: {error}"
                )

            # Error status sources must have an error message
            if status == "error":
                assert error is not None and error != "", (
                    f"Source '{source_id}' has status 'error' but no error message"
                )


class TestSourceMerge:
    """Test suite for source merge functionality."""

    def test_source_merge_priority(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that source merge respects priority ordering."""
        response = api_client.get_sources()
        assert response.get("items"), "No sources found"

        for source in response["items"]:
            enabled = source.get("enabled")
            status = source.get("status")

            if enabled is True:
                assert status in ["available", "error", None], (
                    f"Source {source.get('id')} has enabled=True but status={status}"
                )
            if enabled is False:
                assert status in ["disabled", None], f"Source {source.get('id')} has enabled=False but status={status}"

    def test_source_merge_labels(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that source merge correctly handles labels."""
        response = api_client.get_sources()

        for source in response.get("items", []):
            labels = source.get("labels", [])
            assert isinstance(labels, list)
            for label in labels:
                assert isinstance(label, str)

    def test_enabled_sources_return_models(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that enabled merged sources return models."""
        sources = api_client.get_sources()
        assert sources.get("items"), "No sources found"

        enabled_sources = [s for s in sources["items"] if s.get("enabled") is True or s.get("status") == "available"]

        if not enabled_sources:
            pytest.skip("No enabled sources found in test data")

        models = api_client.get_models()
        assert models.get("items"), "Expected models from enabled sources"


class TestSourceAssetTypeFilter:
    """Test suite for /sources endpoint assetType query parameter filtering."""

    @pytest.mark.parametrize(
        "asset_type,config_key",
        [
            (None, "catalogs"),
            ("models", "catalogs"),
            ("mcp_servers", "mcp_catalogs"),
            ("invalid_value", None),
        ],
        ids=["default-models", "explicit-models", "mcp-servers", "invalid-empty"],
    )
    def test_asset_type_filters_sources(
        self,
        asset_type: str | None,
        config_key: str | None,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        sources_config: dict,
        kind_cluster: bool,
    ):
        """Test that assetType filter returns the correct sources."""
        response = api_client.get_sources(asset_type=asset_type)
        assert isinstance(response.get("items"), list)

        if config_key is None:
            assert response["items"] == []
            return

        if kind_cluster:
            expected_by_id = {s["id"]: s for s in sources_config.get(config_key, [])}
            actual_by_id = {s["id"]: s for s in response["items"]}
            assert set(actual_by_id) == set(expected_by_id)
            for source_id, actual in actual_by_id.items():
                assert actual["name"] == expected_by_id[source_id]["name"]


class TestGetSourceStatusEndpoint:
    """Test suite for GET /sources/{source_id}/status."""

    def _find_source_with_status(self, api_client: CatalogAPIClient, status: str) -> dict | None:
        sources = api_client.get_sources()
        return next((s for s in sources.get("items", []) if s.get("status") == status), None)

    def test_get_status_available_source(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that an available source reports status 'available' with no error."""
        src = self._find_source_with_status(api_client, "available")
        if not src:
            pytest.skip("No available source in test data")

        status = api_client.get_source_status(src["id"])
        assert status["status"] == "available"
        assert not status.get("error")

    def test_get_status_error_source(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that an errored source reports status 'error' with an error message."""
        src = self._find_source_with_status(api_client, "error")
        if not src:
            pytest.skip("No error source in test data")

        status = api_client.get_source_status(src["id"])
        assert status["status"] == "error"
        assert status.get("error"), "error source should include an error message"

    def test_get_status_unknown_source_returns_empty(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that an unknown source returns an empty status (200), not a 404."""
        status = api_client.get_source_status("does_not_exist_xyz")
        assert status.get("status") is None
        assert status.get("error") is None

    def test_get_status_agrees_with_listing(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that the status endpoint agrees with the status embedded in /sources."""
        sources = api_client.get_sources()
        assert sources.get("items"), "No sources found"

        for src in sources["items"]:
            status = api_client.get_source_status(src["id"])
            assert status.get("status") == src.get("status")


class TestClearSourceStatusEndpoint:
    """Test suite for DELETE /sources/{source_id}/status.

    Defined last in this module: clearing a status mutates what /sources
    reports for that source, so these tests run after every status-reading
    assertion above to avoid polluting them.
    """

    def test_clear_unknown_source_is_noop(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that clearing an unknown source's status does not raise."""
        assert api_client.clear_source_status("does_not_exist_xyz") is None

    def test_clear_then_get_returns_empty(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that clearing a source's status makes it read back empty."""
        sources = api_client.get_sources()
        src = next((s for s in sources.get("items", []) if s.get("status") == "error"), None)
        if not src:
            pytest.skip("No error source to clear in test data")

        # Pre-condition: it has a persisted status.
        assert api_client.get_source_status(src["id"]).get("status") == "error"

        api_client.clear_source_status(src["id"])

        after = api_client.get_source_status(src["id"])
        assert after.get("status") is None
        assert after.get("error") is None
