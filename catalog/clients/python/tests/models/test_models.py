"""E2E tests for model functionality.

Tests model filtering, pagination, and basic operations.

To run these tests:
1. Start the catalog service with test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/models/test_models.py
"""

import pytest

from model_catalog import CatalogAPIClient


class TestModels:
    """Test suite for model functionality."""

    def test_get_models_returns_items(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that getting all models returns a list of items."""
        response = api_client.get_models()

        assert "items" in response
        assert isinstance(response["items"], list)

    def test_models_have_required_fields(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that models have required fields."""
        response = api_client.get_models()
        print(response)
        assert response.get("items"), "No models found"

        for model in response["items"]:
            assert "name" in model, "Model missing name"
            assert "source_id" in model, f"Model {model.get('name')} missing source_id"

    def test_filter_models_by_source(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test filtering models by source."""
        sources = api_client.get_sources()
        if sources.get("items"):
            source_id = sources["items"][0].get("id") or sources["items"][0].get("name")
            if source_id:
                response = api_client.get_models(source=source_id)
                assert "items" in response

                # All returned models should be from the specified source
                for model in response.get("items", []):
                    assert model.get("source_id") == source_id

    def test_get_models_with_pagination(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test model pagination."""
        response = api_client.get_models(page_size=5)
        assert "items" in response
        assert len(response["items"]) <= 5

    def test_pagination_next_page(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test getting next page of models."""
        response = api_client.get_models(page_size=3)

        if response.get("nextPageToken"):
            response2 = api_client.get_models(
                page_size=3,
                next_page_token=response["nextPageToken"],
            )
            assert "items" in response2

            # Different pages should have different models
            names1 = {m["name"] for m in response["items"]}
            names2 = {m["name"] for m in response2["items"]}
            # No overlap between pages
            assert not names1.intersection(names2)

    def test_models_reference_valid_sources(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that all models reference valid enabled sources."""
        sources = api_client.get_sources(page_size=100)
        enabled_source_ids = {
            s.get("id") or s.get("name")
            for s in sources.get("items", [])
            if s.get("enabled") is True or s.get("status") == "available"
        }

        models = api_client.get_models()

        for model in models.get("items", []):
            source_id = model.get("source_id")
            assert source_id in enabled_source_ids, f"Model {model.get('name')} has invalid source_id: {source_id}"

    def test_models_from_disabled_source_excluded(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that models from disabled sources don't appear in results."""
        sources = api_client.get_sources()
        assert sources.get("items"), "No sources found"

        disabled_source_ids = [
            s.get("id") or s.get("name")
            for s in sources["items"]
            if s.get("enabled") is False or s.get("status") == "disabled"
        ]

        if not disabled_source_ids:
            pytest.skip("No disabled sources found in test data")

        models = api_client.get_models()
        model_source_ids = [m.get("source_id") for m in models.get("items", [])]

        for disabled_id in disabled_source_ids:
            assert disabled_id not in model_source_ids, f"Model from disabled source {disabled_id} found in results"

    def test_model_count_consistency(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that model counts per source are consistent."""
        sources = api_client.get_sources()
        models = api_client.get_models()

        model_counts: dict[str, int] = {}
        for model in models.get("items", []):
            source_id = model.get("source_id")
            model_counts[source_id] = model_counts.get(source_id, 0) + 1

        for source_id in model_counts:
            source = next(
                (s for s in sources["items"] if s.get("id") == source_id),
                None,
            )
            if source:
                assert source.get("enabled") is True or source.get("status") == "available", (
                    f"Source {source_id} has {model_counts[source_id]} models but is not enabled"
                )

    def test_models_custom_properties_has_valid_structure(
        self, api_client: CatalogAPIClient, suppress_ssl_warnings: None, kind_cluster: bool
    ):
        """Test that models with custom properties have valid MetadataStringValue structure."""
        models = api_client.get_models()
        models_with_props = [m for m in models.get("items", []) if m.get("customProperties")]

        if not models_with_props:
            pytest.skip("No models with custom properties found in test data")

        all_errors = []
        for model in models_with_props:
            model_name = model.get("name")
            custom_properties = model.get("customProperties")
            if errors := _validate_custom_property_structure(
                custom_properties=custom_properties, kind_cluster=kind_cluster
            ):
                err_lines = "\n".join(errors)
                all_errors.append(f"Model '{model_name}': {err_lines}")
        assert not all_errors, "\n".join(all_errors)


def _validate_custom_property_structure(custom_properties: dict, kind_cluster: bool) -> list[str]:
    """Validate the structure of a custom property."""
    expected_keys = ["size", "tensor_type", "variant_group_id"] if kind_cluster else ["model_type"]
    errors = []
    for key in expected_keys:
        prop = custom_properties[key]

        if not isinstance(prop, dict):
            errors.append(f"Custom property '{key}' is not a dictionary")

        if "metadataType" not in prop:
            errors.append(f"Custom property '{key}' missing 'metadataType' field")
        if prop.get("metadataType") != "MetadataStringValue":
            errors.append(f"Custom property '{key}' has unexpected metadataType: {prop.get('metadataType')} ")

        if "string_value" not in prop:
            errors.append(f"Custom property '{key}' missing 'string_value' field")
        if not isinstance(prop.get("string_value"), str):
            errors.append(f"Custom property '{key}' string_value is not a string: {type(prop.get('string_value'))}")

    return errors
