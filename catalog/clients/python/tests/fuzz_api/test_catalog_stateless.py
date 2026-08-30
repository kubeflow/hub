"""Stateless fuzzing tests for Catalog API using Schemathesis."""

import pytest
import schemathesis
from hypothesis import settings

schema = schemathesis.pytest.from_fixture("generated_schema")


@pytest.mark.parametrize("generated_schema", ["catalog.yaml"], indirect=True)
@schema.parametrize()
@settings(
    deadline=None,
)
@pytest.mark.fuzz
def test_catalog_api_stateless(
    auth_headers: dict, case: schemathesis.Case, verify_ssl: bool, suppress_ssl_warnings: None
) -> None:
    """Test the Model Catalog API endpoints.

    This test uses schemathesis to generate and validate API requests
    against the OpenAPI schema. It performs stateless testing, meaning
    each request is independent of others.

    Args:
        auth_headers: Authorization headers for API requests
        case: Schemathesis test case with generated request data
        verify_ssl: Whether to verify SSL certificates
    """
    case.call_and_validate(headers=auth_headers, verify=verify_ssl)
