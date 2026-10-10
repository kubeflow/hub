import pytest
import schemathesis
from hypothesis import settings

from tests.fuzz_api.model_registry.test_mr_stateless import call_and_validate_with_null_byte_handling

schema = schemathesis.pytest.from_fixture("generated_schema")

@pytest.mark.parametrize("generated_schema", ["catalog-v1.yaml"], indirect=True)
@schema.parametrize()
@settings(
    deadline=None,
)
@pytest.mark.fuzz
def test_catalog_api_stateless(auth_headers: dict, case: schemathesis.Case, verify_ssl: bool) -> None:
    """Test the Model Catalog API endpoints.

    This test uses schemathesis to generate and validate API requests
    """
    call_and_validate_with_null_byte_handling(case, auth_headers, verify_ssl)
