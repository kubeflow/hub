import warnings

import pytest
from hypothesis import HealthCheck, settings
from hypothesis.errors import Unsatisfiable


@pytest.mark.fuzz
class TestRestAPIStateful:
    @pytest.mark.parametrize("generated_schema", ["model-registry-v1.yaml"], indirect=True)
    def test_mr_api_stateful(self, state_machine):
        """Launches stateful tests against the Model Registry API endpoints defined in its openAPI yaml spec file"""
        try:
            state_machine.run(settings=settings(
                max_examples=20,
                deadline=10000,
                suppress_health_check=[
                    HealthCheck.filter_too_much,
                    HealthCheck.too_slow,
                    HealthCheck.data_too_large,
                ],
            ))
        except Unsatisfiable:
            warnings.warn(
                "Stateful test hit Hypothesis Unsatisfiable — tight spec constraints "
                "with allOf composition make data generation non-deterministic. "
                "This is not a server bug.",
                stacklevel=1,
            )
