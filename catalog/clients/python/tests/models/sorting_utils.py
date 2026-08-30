"""Utility functions for sorting tests."""

from typing import Any


def get_field_value(item: dict[str, Any], field: str) -> Any:
    """Extract field value from an item for sorting comparison.

    Args:
        item: Model/item dictionary from API response
        field: Field name (ID, CREATE_TIME, LAST_UPDATE_TIME)

    Returns:
        The field value, converted appropriately for comparison

    Raises:
        ValueError: If field is invalid or missing from item
    """
    field_mapping = {
        "ID": "id",
        "CREATE_TIME": "createTimeSinceEpoch",
        "LAST_UPDATE_TIME": "lastUpdateTimeSinceEpoch",
    }

    if field not in field_mapping:
        raise ValueError(f"Invalid field: {field}")

    api_field = field_mapping[field]
    value = item.get(api_field)

    if value is None:
        raise ValueError(f"Field {field} ({api_field}) is missing from item: {item}")

    # Convert ID for proper numeric comparison
    if field == "ID":
        try:
            return int(value)
        except ValueError:
            return str(value)

    return value


def sort_items_by_field(items: list[dict], field: str, order: str) -> list[dict]:
    """Sort items by the specified field and order.

    Args:
        items: List of items to sort
        field: Field name to sort on (ID, CREATE_TIME, LAST_UPDATE_TIME)
        order: Sort order (ASC or DESC)

    Returns:
        A new list of items sorted by the specified field and order.

    Raises:
        ValueError: If field or order is invalid
    """
    if order in {"ASC", "DESC"}:
        return sorted(items, key=lambda item: get_field_value(item, field), reverse=(order == "DESC"))
    raise ValueError(f"Invalid sort order: {order}")


def sort_items_by_custom_property(items: list[dict], property_field: str, sort_order: str) -> list[dict]:
    """Sort items by a custom property value with fallback behavior.

    Expected behavior:
    1. Items WITH the property appear first, sorted by value (ASC/DESC)
    2. Items WITHOUT the property appear after, sorted by ID ASC (fallback)

    Args:
        items: List of artifact items from API response
        property_field: Property field path (e.g., "accuracy.double_value")
        sort_order: Sort order (ASC or DESC)

    Returns:
        A new list of items sorted by the custom property with fallback to ID ASC.

    Raises:
        ValueError: If sort_order is invalid
    """
    if sort_order not in ("ASC", "DESC"):
        raise ValueError(f"Invalid sort order: {sort_order}")

    property_name, value_type = property_field.rsplit(".", 1)

    items_with = []
    items_without = []

    for item in items:
        custom_props = item.get("customProperties", {})
        if property_name in custom_props:
            value = custom_props[property_name].get(value_type)
            if value is not None:
                items_with.append((item, value))
            else:
                items_without.append(item)
        else:
            items_without.append(item)

    sorted_with = sorted(items_with, key=lambda x: x[1], reverse=(sort_order == "DESC"))
    sorted_without = sorted(items_without, key=lambda item: int(item["id"]))

    return [item for item, _ in sorted_with] + sorted_without
