"""E2E tests for artifact functionality.

Tests artifact filtering and ordering behavior in the catalog API.
Consolidated from test_artifact_filtering.py and test_artifacts_ordering.py.

To run these tests:
1. Start the catalog service with test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/models/test_artifacts.py
"""

import pytest

from model_catalog import CatalogAPIClient, CatalogValidationError
from tests.models.sorting_utils import sort_items_by_custom_property, sort_items_by_field


class TestArtifacts:
    """Test suite for artifact functionality (filtering and ordering).

    Uses the model_with_artifacts fixture from conftest.py for tests
    that need a model with artifacts.
    """

    # === Basic Tests ===

    def test_get_artifacts_returns_response(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test that getting artifacts returns a response."""
        source_id, model_name = model_with_artifacts
        response = api_client.get_artifacts(source_id=source_id, model_name=model_name)
        assert isinstance(response, dict)
        assert "items" in response

    def test_artifacts_have_expected_structure(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test that artifacts have expected structure."""
        source_id, model_name = model_with_artifacts
        response = api_client.get_artifacts(source_id=source_id, model_name=model_name)

        for item in response.get("items", []):
            assert isinstance(item, dict)
            # Should have at least id or uri
            assert "id" in item or "uri" in item

    # === Filtering Tests ===

    def test_filter_artifacts_by_model(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test filtering artifacts by model source and name."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(source_id=source_id, model_name=model_name)
        assert isinstance(response, dict)
        assert "items" in response

        # Verify we got artifacts
        artifacts = response.get("items", [])
        assert len(artifacts) > 0, f"Expected artifacts for {model_name}"

    def test_filter_artifacts_by_query(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test filtering artifacts using filter query."""
        source_id, model_name = model_with_artifacts

        # Get all artifacts first
        all_artifacts = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
        )

        assert all_artifacts.get("items"), "No artifacts available for filtering test"

        # Test filter by framework_type property.
        # Note: This filter may return empty results if test data doesn't have
        # artifacts with framework_type="pytorch". The test validates the filter
        # mechanism works, not that specific data exists.
        filter_query = 'framework_type.string_value = "pytorch"'
        filtered = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            filter_query=filter_query,
        )

        assert isinstance(filtered, dict)
        assert "items" in filtered

        # Verify any filtered results actually match the filter criteria
        for artifact in filtered.get("items", []):
            props = artifact.get("customProperties", {})
            if "framework_type" in props:
                assert props["framework_type"].get("string_value") == "pytorch"

    def test_filter_artifacts_by_numeric_property(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test filtering artifacts by numeric custom property."""
        source_id, model_name = model_with_artifacts

        # Test filter by accuracy > 0.9
        filter_query = "accuracy.double_value > 0.9"
        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            filter_query=filter_query,
        )

        assert isinstance(response, dict)
        assert "items" in response

        # Verify all returned artifacts have accuracy > 0.9
        for artifact in response.get("items", []):
            props = artifact.get("customProperties", {})
            if "accuracy" in props:
                acc = props["accuracy"].get("double_value")
                assert acc is not None, "Accuracy value is None"
                assert acc > 0.9, f"Expected accuracy > 0.9, got {acc}"

    def test_filter_artifacts_with_and_logic(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], kind_cluster: bool
    ) -> None:
        """Test filtering artifacts with AND logic combining multiple conditions."""
        if not kind_cluster:
            pytest.skip("This test would be skipped on non-kind cluster")
        source_id, model_name = model_with_artifacts

        # Test combining filters with AND
        filter_query = '(framework_type.string_value = "pytorch") AND (accuracy.double_value > 0.5)'
        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            filter_query=filter_query,
        )

        assert isinstance(response, dict)
        assert "items" in response

        # Ensure at least one artifact matches (test data should have pytorch with accuracy > 0.5)
        items = response.get("items", [])
        assert len(items) > 0, "Expected at least one artifact matching AND filter"

        # Verify all returned artifacts match both conditions
        for artifact in items:
            props = artifact.get("customProperties", {})

            framework = props["framework_type"].get("string_value")
            assert framework == "pytorch", f"Expected framework_type='pytorch', got '{framework}'"

            acc = props["accuracy"].get("double_value")
            assert acc is not None and acc > 0.5, f"Expected accuracy > 0.5, got {acc}"

    def test_filter_artifacts_with_or_logic(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], kind_cluster: bool
    ) -> None:
        """Test filtering artifacts with OR logic combining multiple conditions."""
        if not kind_cluster:
            pytest.skip("This test would be skipped on non-kind cluster")
        source_id, model_name = model_with_artifacts

        # Test combining filters with OR using existing frameworks in test data
        filter_query = '(framework_type.string_value = "pytorch") OR (framework_type.string_value = "onnx")'
        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            filter_query=filter_query,
        )

        assert isinstance(response, dict)
        assert "items" in response

        # Ensure at least one artifact matches (test data should have pytorch or onnx)
        items = response.get("items", [])
        assert len(items) > 0, "Expected at least one artifact matching OR filter"

        # Verify all returned artifacts match at least one condition
        for artifact in items:
            props = artifact.get("customProperties", {})

            framework = props["framework_type"].get("string_value")
            assert framework in ["pytorch", "onnx"], (
                f"Expected framework_type to be 'pytorch' or 'onnx', got '{framework}'"
            )

    def test_filter_artifacts_returns_empty_for_no_matches(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test that a valid filter query with no matching artifacts returns empty results."""
        source_id, model_name = model_with_artifacts

        # Use a valid query that should return no results
        filter_query = 'framework_type.string_value = "nonexistent_framework_xyz"'
        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            filter_query=filter_query,
        )

        assert isinstance(response, dict)
        assert "items" in response
        assert response["items"] == [], "Expected empty results for non-matching filter"
        assert response.get("size", 0) == 0, "Expected size to be 0 for non-matching filter"

    def test_multiple_models_have_different_artifacts(
        self, api_client: CatalogAPIClient, suppress_ssl_warnings: None
    ) -> None:
        """Test that different models have their own artifacts."""
        models = api_client.get_models()
        if not models.get("items") or len(models["items"]) < 2:
            pytest.skip("Need at least 2 models for this test")

        model1 = models["items"][0]
        model2 = models["items"][1]

        artifacts1 = api_client.get_artifacts(
            source_id=model1["source_id"],
            model_name=model1["name"],
        )

        artifacts2 = api_client.get_artifacts(
            source_id=model2["source_id"],
            model_name=model2["name"],
        )

        # Verify both responses are valid
        assert isinstance(artifacts1, dict)
        assert isinstance(artifacts2, dict)
        assert "items" in artifacts1
        assert "items" in artifacts2

    # === Ordering Tests ===

    def test_artifacts_have_ordering_fields(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test artifacts have timestamp fields for ordering."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
        )

        artifacts = response.get("items", [])
        if len(artifacts) < 1:
            pytest.skip("Need at least 1 artifact to test")

        # Check that artifacts have some form of timestamp or id for ordering
        for artifact in artifacts:
            has_ordering_field = (
                artifact.get("id") is not None
                or artifact.get("createTimeSinceEpoch") is not None
                or artifact.get("lastUpdateTimeSinceEpoch") is not None
            )
            assert has_ordering_field, "Artifact missing ordering fields"

    def test_artifacts_have_identifiers(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test artifacts can be identified by name/id."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
        )

        artifacts = response.get("items", [])

        # Extract artifact identifiers
        identifiers = []
        for artifact in artifacts:
            name = artifact.get("name") or artifact.get("artifactType") or artifact.get("id")
            if name:
                identifiers.append(name)

        # We should have identifiers for artifacts
        assert len(identifiers) > 0, "No artifact identifiers found"

    def test_artifacts_pagination(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test artifacts support pagination."""
        source_id, model_name = model_with_artifacts

        # Request small page size
        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            page_size=1,
        )

        assert isinstance(response, dict)
        assert "items" in response

        # Check pagination fields
        assert "pageSize" in response or "size" in response

    def test_artifacts_custom_properties(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test that artifacts with custom properties are valid."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
        )

        artifacts = response.get("items", [])

        # Extract accuracy values to verify they exist
        accuracy_values = []
        for artifact in artifacts:
            props = artifact.get("customProperties", {})
            if "accuracy" in props:
                acc = props["accuracy"].get("double_value")
                if acc is not None:
                    accuracy_values.append(acc)

        # Test data should have artifacts with accuracy
        if accuracy_values:
            # Verify values are valid numbers
            for acc in accuracy_values:
                assert isinstance(acc, (int, float))
                assert 0 <= acc <= 1, f"Accuracy {acc} should be between 0 and 1"

    @pytest.mark.parametrize(
        "artifact_type",
        [
            pytest.param("model-artifact", id="single_model_artifact"),
            pytest.param("metrics-artifact", id="single_metrics_artifact"),
            pytest.param(["model-artifact", "metrics-artifact"], id="multiple_artifact_types"),
        ],
    )
    def test_filter_artifacts_by_artifact_type(
        self,
        api_client: CatalogAPIClient,
        model_with_artifacts: tuple[str, str],
        suppress_ssl_warnings: None,
        artifact_type: str | list[str],
    ) -> None:
        """Test filtering artifacts by single or multiple artifact types."""
        source_id, model_name = model_with_artifacts

        # Get all artifacts first
        all_artifacts_response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
        )
        all_artifacts = all_artifacts_response["items"]

        # Filter by artifact type
        filtered_response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            artifact_type=artifact_type,
        )

        filtered_artifacts = filtered_response["items"]

        # Convert to list for validation
        expected_types = [artifact_type] if isinstance(artifact_type, str) else artifact_type

        # Verify all returned artifacts match the requested type(s)
        for artifact in filtered_artifacts:
            assert artifact["artifactType"] in expected_types, (
                f"Expected artifactType to be one of {expected_types}, got '{artifact['artifactType']}'"
            )

        # Verify the filter didn't miss any artifacts of the requested type(s)
        expected_artifacts = [a for a in all_artifacts if a.get("artifactType") in expected_types]
        assert len(filtered_artifacts) == len(expected_artifacts), (
            f"Filter returned {len(filtered_artifacts)} artifacts, "
            f"but expected {len(expected_artifacts)} artifacts of type(s) {expected_types}"
        )


class TestNegativeArtifacts:
    """Test suite for negative artifact functionality."""

    @pytest.mark.parametrize(
        "invalid_filter_query",
        [
            pytest.param(
                "fake IN ('test', 'fake'))",
                id="malformed_syntax_unbalanced_parentheses",
            ),
            pytest.param(
                "ttft_p90.double_value < abc",
                id="type_mismatch_string_in_numeric_comparison",
            ),
            pytest.param(
                "hardware_type.string_value = 5.0",
                id="type_mismatch_number_in_string_equality",
            ),
            pytest.param(
                "field.string_value IN (unclosed",
                id="malformed_syntax_unclosed_list",
            ),
            pytest.param(
                "field.string_value AND",
                id="malformed_syntax_incomplete_expression",
            ),
        ],
    )
    def test_search_artifacts_by_invalid_filter_query(
        self,
        api_client: CatalogAPIClient,
        model_with_artifacts: tuple[str, str],
        invalid_filter_query: str,
        suppress_ssl_warnings: None,
    ) -> None:
        """Test that search artifacts by invalid filter query raises a validation error."""
        source_id, model_name = model_with_artifacts
        with pytest.raises(CatalogValidationError, match="invalid filter query"):
            api_client.get_artifacts(
                source_id=source_id,
                model_name=model_name,
                filter_query=invalid_filter_query,
            )

    def test_filter_artifacts_by_invalid_artifact_type(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str]
    ) -> None:
        """Test that filtering by an invalid artifact type raises a validation error."""
        source_id, model_name = model_with_artifacts

        invalid_artifact_type = "invalid-artifact-type"

        with pytest.raises(CatalogValidationError, match="Input should be 'model-artifact' or 'metrics-artifact'"):
            api_client.get_artifacts(
                source_id=source_id,
                model_name=model_name,
                artifact_type=invalid_artifact_type,
            )


@pytest.mark.parametrize("model_with_artifacts", [2], indirect=True)
class TestArtifactsSorting:
    """Basic sorting coverage for artifact endpoints. Thorough testing is done downstream."""

    @pytest.mark.parametrize(
        "order_by,sort_order",
        [
            pytest.param("ID", "ASC", id="id_asc"),
            pytest.param("ID", "DESC", id="id_desc"),
        ],
    )
    def test_artifacts_sorting_by_id(
        self,
        api_client: CatalogAPIClient,
        model_with_artifacts: tuple[str, str],
        suppress_ssl_warnings: None,
        order_by: str,
        sort_order: str,
    ) -> None:
        """Test artifacts endpoint sorts correctly by ID."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            order_by=order_by,
            sort_order=sort_order,
        )

        items = response["items"]
        assert len(items) >= 2, "Need at least 2 artifacts to validate sorting"
        assert items == sort_items_by_field(items, order_by, sort_order)

    @pytest.mark.parametrize(
        "order_by,sort_order",
        [
            pytest.param("accuracy.double_value", "ASC", id="accuracy_asc"),
            pytest.param("accuracy.double_value", "DESC", id="accuracy_desc"),
        ],
    )
    def test_artifacts_sorting_by_custom_property(
        self,
        api_client: CatalogAPIClient,
        model_with_artifacts: tuple[str, str],
        suppress_ssl_warnings: None,
        order_by: str,
        sort_order: str,
    ) -> None:
        """Test artifacts endpoint sorts correctly by custom property values."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            order_by=order_by,
            sort_order=sort_order,
        )

        items = response["items"]
        assert len(items) >= 2, "Need at least 2 artifacts to validate sorting"
        assert items == sort_items_by_custom_property(items, order_by, sort_order)

    def test_sorting_by_non_existing_property_falls_back_to_id(
        self, api_client: CatalogAPIClient, model_with_artifacts: tuple[str, str], suppress_ssl_warnings: None
    ) -> None:
        """Test that sorting by a non-existing property falls back to ID ASC."""
        source_id, model_name = model_with_artifacts

        response = api_client.get_artifacts(
            source_id=source_id,
            model_name=model_name,
            order_by="non_existing_property.double_value",
            sort_order="DESC",
        )

        items = response["items"]
        assert len(items) >= 2, "Need at least 2 artifacts to validate sorting"

        # No artifacts have this property, so all should fall back to ID ASC
        assert items == sort_items_by_field(items, "ID", "ASC")
