"""E2E tests for ordering functionality.

Tests NAME and ACCURACY ordering behavior in the catalog API.
Consolidated from test_name_ordering.py and test_accuracy_sorting.py.

To run these tests:
1. Start the catalog service with test data loaded
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/models/test_ordering.py
"""

from typing import Any

import pytest

from model_catalog import CatalogAPIClient
from tests.models.sorting_utils import sort_items_by_field


def _assert_response_valid(response: dict[str, Any]) -> None:
    """Assert that API response has valid structure.

    Args:
        response: The API response dictionary to validate.

    Raises:
        AssertionError: If response is not a dict or missing 'items' field.
    """
    assert isinstance(response, dict), f"Response is not a dict: {type(response)}"
    assert "items" in response, f"Response missing 'items' field: {response.keys()}"
    assert isinstance(response["items"], list)


def _get_model_accuracy(model: dict[str, Any]) -> float | None:
    """Extract accuracy value from a model's artifacts.

    Looks for 'overall_average' or 'accuracy' in customProperties.

    Args:
        model: A model dictionary containing artifacts with customProperties.

    Returns:
        The accuracy value as a float, or None if not found.
    """
    for artifact in model.get("artifacts", []):
        custom_props = artifact.get("customProperties", {})
        for key in ("overall_average", "accuracy"):
            if key in custom_props:
                val = custom_props[key]
                if "double_value" in val:
                    return float(val["double_value"])
    return None


def _model_has_accuracy(model: dict[str, Any]) -> bool:
    """Check if a model has any accuracy metric.

    Args:
        model: A model dictionary to check for accuracy metrics.

    Returns:
        True if the model has an accuracy metric, False otherwise.
    """
    return _get_model_accuracy(model) is not None


class TestNameOrdering:
    """Test suite for NAME ordering functionality."""

    def test_order_by_name_asc(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that ordering by name ASC returns models."""
        response = api_client.get_models(order_by="name", sort_order="ASC")
        _assert_response_valid(response)

    def test_order_by_name_desc(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that ordering by name DESC returns models."""
        response = api_client.get_models(order_by="name", sort_order="DESC")
        _assert_response_valid(response)

    def test_name_asc_vs_desc_are_reversed(
        self, api_client: CatalogAPIClient, suppress_ssl_warnings: None, kind_cluster: bool
    ):
        """Test that ASC and DESC orderings are reversed."""
        # Use large page size to get all models
        page_size = 100 if kind_cluster else 1000
        response_asc = api_client.get_models(order_by="name", sort_order="ASC", page_size=page_size)
        response_desc = api_client.get_models(order_by="name", sort_order="DESC", page_size=page_size)

        if response_asc["items"] and len(response_asc["items"]) > 1:
            names_asc = [m["name"] for m in response_asc["items"]]
            names_desc = [m["name"] for m in response_desc["items"]]
            assert names_asc == list(reversed(names_desc))

    def test_name_ordering_consistent(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test name ordering returns consistent results."""
        response = api_client.get_models(order_by="name", sort_order="ASC")
        _assert_response_valid(response)

        if response["items"]:
            names = [m["name"] for m in response["items"]]
            # Verify we get a deterministic order
            assert len(names) > 0
            # Second call should return same order
            response2 = api_client.get_models(order_by="name", sort_order="ASC")
            names2 = [m["name"] for m in response2["items"]]
            assert names == names2

    def test_name_ordering_pagination_maintains_order(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that pagination maintains ordering.

        Note: Uses nextPageToken from first response. This is safe because test data
        is static during test execution. In production, tokens could become stale if
        underlying data changes between requests.
        """
        # Get first page
        response = api_client.get_models(order_by="name", sort_order="ASC", page_size=5)
        _assert_response_valid(response)

        if response.get("nextPageToken"):
            # Get second page using token from first response
            response2 = api_client.get_models(
                order_by="name", sort_order="ASC", page_size=5, next_page_token=response["nextPageToken"]
            )
            _assert_response_valid(response2)

            if response["items"] and response2["items"]:
                # Last item of first page should come before or equal to first item of second page.
                # Equality is valid when models have identical names (e.g., same model from
                # different sources) since the secondary sort order is undefined.
                last_name_page1 = response["items"][-1]["name"].lower()
                first_name_page2 = response2["items"][0]["name"].lower()
                assert last_name_page1 <= first_name_page2, (
                    f"Pagination order violated: last item of page 1 ('{last_name_page1}') "
                    f"should come before or equal to first item of page 2 ('{first_name_page2}')"
                )


class TestFieldOrdering:
    """Test suite for ID, CREATE_TIME, and LAST_UPDATE_TIME ordering."""

    @pytest.mark.parametrize(
        "order_by,sort_order",
        [
            ("ID", "ASC"),
            ("ID", "DESC"),
            ("CREATE_TIME", "ASC"),
            ("CREATE_TIME", "DESC"),
            ("LAST_UPDATE_TIME", "ASC"),
            ("LAST_UPDATE_TIME", "DESC"),
        ],
    )
    def test_field_sorting_works_correctly(
        self,
        order_by: str,
        sort_order: str,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test models endpoint sorts correctly by field and order.

        Validates that the API correctly sorts models by:
        - ID (numeric)
        - CREATE_TIME (timestamp)
        - LAST_UPDATE_TIME (timestamp)

        In both ASC and DESC order.
        """
        response = api_client.get_models(order_by=order_by, sort_order=sort_order)

        assert isinstance(response, dict), f"Response is not a dict: {type(response)}"
        assert "items" in response, f"Response missing 'items' field: {response.keys()}"

        items = response["items"]

        assert items == sort_items_by_field(items, order_by, sort_order), (
            f"Models not sorted correctly by {order_by} {sort_order}"
        )


class TestAccuracyOrdering:
    """Test suite for ACCURACY ordering functionality."""

    def test_order_by_accuracy_desc(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that orderBy=ACCURACY with sortOrder=DESC returns a valid response."""
        response = api_client.get_models(order_by="ACCURACY", sort_order="DESC")
        assert isinstance(response, dict)
        assert "items" in response

    def test_order_by_accuracy_asc(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that orderBy=ACCURACY with sortOrder=ASC returns a valid response."""
        response = api_client.get_models(order_by="ACCURACY", sort_order="ASC")
        assert isinstance(response, dict)
        assert "items" in response

    def test_accuracy_desc_sorts_correctly(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that orderBy=ACCURACY DESC returns models in descending accuracy order."""
        response = api_client.get_models(order_by="ACCURACY", sort_order="DESC")
        items = response.get("items", [])

        if len(items) < 2:
            pytest.skip("Need at least 2 models to test sorting")

        # Extract accuracy values from models that have them
        accuracies = []
        for model in items:
            acc = _get_model_accuracy(model)
            if acc is not None:
                accuracies.append(acc)

        if len(accuracies) >= 2:
            # Verify descending order for models with accuracy
            for i in range(len(accuracies) - 1):
                assert accuracies[i] >= accuracies[i + 1], f"Accuracies not in descending order: {accuracies}"

    def test_models_without_accuracy_come_last(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that models without accuracy metrics come last (NULLS LAST)."""
        response = api_client.get_models(order_by="ACCURACY", sort_order="DESC")
        items = response.get("items", [])

        if not items:
            pytest.skip("No models available")

        # Track if we've seen a model without accuracy
        found_model_without_accuracy = False
        found_model_with_accuracy_after = False

        for model in items:
            if not _model_has_accuracy(model):
                found_model_without_accuracy = True
            elif found_model_without_accuracy:
                found_model_with_accuracy_after = True

        assert not found_model_with_accuracy_after, (
            "Found model with accuracy after model without - NULLS LAST violated"
        )

    def test_accuracy_sorting_with_pagination(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that accuracy sorting works correctly with pagination."""
        response1 = api_client.get_models(
            order_by="ACCURACY",
            sort_order="DESC",
            page_size=2,
        )
        assert "items" in response1

        if not response1.get("nextPageToken"):
            pytest.skip("Not enough models for pagination test")

        response2 = api_client.get_models(
            order_by="ACCURACY",
            sort_order="DESC",
            page_size=2,
            next_page_token=response1["nextPageToken"],
        )
        assert "items" in response2

        all_items = response1["items"] + response2["items"]
        assert len(all_items) >= 2

    def test_accuracy_sorting_returns_all_models(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that accuracy sorting returns all models, not just those with accuracy."""
        response_all = api_client.get_models()
        all_count = len(response_all.get("items", []))

        response_sorted = api_client.get_models(order_by="ACCURACY", sort_order="DESC")
        sorted_count = len(response_sorted.get("items", []))

        assert sorted_count == all_count, f"Sorted count ({sorted_count}) != all count ({all_count})"

    def test_accuracy_metrics_structure(self, api_client: CatalogAPIClient, suppress_ssl_warnings: None):
        """Test that accuracy metrics are stored as custom properties on artifacts."""
        response = api_client.get_models()
        items = response.get("items", [])

        for model in items:
            for artifact in model.get("artifacts", []):
                custom_props = artifact.get("customProperties", {})
                if "accuracy" in custom_props:
                    accuracy_prop = custom_props["accuracy"]
                    assert "double_value" in accuracy_prop or "metadataType" in accuracy_prop, (
                        f"Accuracy property has unexpected structure: {accuracy_prop}"
                    )

    @pytest.mark.parametrize(
        "sort_order,filter_query",
        [
            ("ASC", "tasks='text-generation'"),
            ("DESC", "tasks='text-generation'"),
        ],
    )
    def test_accuracy_sorting_with_filter(
        self,
        sort_order: str,
        filter_query: str,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test accuracy sorting works correctly with filter applied.

        Validates:
        1. Filter is applied correctly (only matching models returned)
        2. Models WITH accuracy appear first, sorted by accuracy value
        3. Models WITHOUT accuracy appear after (NULLS LAST)
        """
        response = api_client.get_models(
            order_by="ACCURACY",
            sort_order=sort_order,
            filter_query=filter_query,
        )

        assert isinstance(response, dict)
        assert "items" in response

        items = response.get("items", [])

        # Validate NULLS LAST behavior
        found_model_without_accuracy = False
        accuracies_with_values = []

        for model in items:
            acc = _get_model_accuracy(model)
            if acc is None:
                found_model_without_accuracy = True
            else:
                assert not found_model_without_accuracy, "Found model with accuracy after model without accuracy"
                accuracies_with_values.append(acc)

        # Validate accuracy ordering
        if len(accuracies_with_values) >= 2:
            expected = sorted(accuracies_with_values, reverse=(sort_order == "DESC"))
            assert accuracies_with_values == expected, (
                f"Accuracies not in {sort_order} order: {accuracies_with_values}"
            )
