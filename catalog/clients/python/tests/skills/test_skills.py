"""E2E tests for skill catalog functionality.

Tests skill listing, retrieval by ID, pagination, and error handling.

Unlike models and MCP servers, skill sources are git-backed
(type: git-skills-plugin) and the e2e overlay does not register one, so
most of these tests are structural rather than data-driven: they assert on
response shape and error behavior, not on specific seeded skill names or
counts. See kep-0005 SKC-601 for connected e2e coverage against real
skill data.

TestSkillsClientParamTranslation is the exception: it mocks the generated
skill_api and needs no running catalog, giving real parameter-plumbing
regression coverage independent of any seeded data.

To run these tests:
1. Start the catalog service (skill catalog is enabled by default)
2. Set CATALOG_URL environment variable (default: http://localhost:8081)
3. Run: pytest --e2e tests/skills/test_skills.py
"""

from unittest.mock import MagicMock

import pytest

from model_catalog import CatalogAPIClient, CatalogNotFoundError


class TestSkillsBasics:
    """Test suite for basic skill listing and retrieval."""

    def test_get_skills_returns_paginated_list(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test that get_skills returns a well-formed paginated response."""
        response = api_client.get_skills(page_size=10)
        assert "items" in response
        assert isinstance(response["items"], list)
        assert "pageSize" in response

    def test_get_skills_pagination_params_accepted(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test that page_size=1 is honored (at most one item per page)."""
        response = api_client.get_skills(page_size=1)
        assert len(response.get("items", [])) <= 1

    def test_get_skill_by_id_round_trip(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test that a skill returned by get_skills can be fetched by ID."""
        response = api_client.get_skills(page_size=1)
        items = response.get("items", [])
        if not items:
            pytest.skip("No skills loaded in this environment")

        skill = items[0]
        single = api_client.get_skill(skill_id=skill["id"])
        assert single["name"] == skill["name"]


class TestSkillsQueryParams:
    """Test suite for skill list query parameters — accepted without error.

    These only confirm the server doesn't reject the params (e.g. with a 400).
    Since no skill source is registered in the e2e overlay, get_skills()
    returns an empty list regardless of what's passed here, so an empty
    response is NOT evidence the param was actually applied — see
    TestSkillsClientParamTranslation below for that coverage.
    """

    @pytest.mark.parametrize(
        "kwargs",
        [
            pytest.param({"name": "%nonexistent-skill-xyz-12345%"}, id="name_like_pattern"),
            pytest.param({"q": "nonexistent_search_term_xyz_98765"}, id="q_keyword_search"),
            pytest.param({"source": ["nonexistent-source-id"]}, id="source_filter"),
            pytest.param({"source_label": ["nonexistent-label"]}, id="source_label_filter"),
            pytest.param({"filter_query": "category = 'nonexistent-category-xyz'"}, id="filter_query"),
        ],
    )
    def test_query_params_accepted_without_error(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
        kwargs: dict,
    ):
        """Test that each list query param is accepted by the server (no 400)."""
        api_client.get_skills(**kwargs)


class TestSkillsClientParamTranslation:
    """Test suite verifying CatalogAPIClient.get_skills/get_skill correctly
    translate Python kwargs into generated-client call arguments.

    Unlike the rest of this module, these don't need a running catalog: they
    construct a client directly and patch its generated skill_api, so they
    give real regression coverage for parameter plumbing (e.g. a future
    change that swaps source/source_label, or breaks empty-string handling)
    independent of whether any skill data is loaded anywhere.
    """

    @pytest.fixture
    def client_with_mocked_skill_api(self) -> CatalogAPIClient:
        """A CatalogAPIClient whose skill_api.find_skills/get_skill are mocked.

        __init__ never makes a network call, so this needs no live server.
        """
        client = CatalogAPIClient("http://localhost:1", timeout=1)
        client.skill_api = MagicMock()
        client.skill_api.find_skills.return_value.to_dict.return_value = {"items": [], "pageSize": 10}
        client.skill_api.get_skill.return_value.to_dict.return_value = {"id": "1", "name": "example"}
        return client

    @pytest.mark.parametrize(
        ("kwargs", "expected_call_kwargs"),
        [
            pytest.param({"name": "deploy%"}, {"name": "deploy%"}, id="name"),
            pytest.param({"q": "kubectl"}, {"q": "kubectl"}, id="q"),
            pytest.param({"source": "src-a"}, {"source": ["src-a"]}, id="source_single_string"),
            pytest.param({"source": ["src-a", "src-b"]}, {"source": ["src-a", "src-b"]}, id="source_list"),
            pytest.param({"source": ""}, {"source": None}, id="source_empty_string_is_no_filter"),
            pytest.param({"source_label": "official"}, {"source_label": ["official"]}, id="source_label_single_string"),
            pytest.param({"source_label": ""}, {"source_label": None}, id="source_label_empty_string_is_no_filter"),
            pytest.param({"filter_query": "category='devops'"}, {"filter_query": "category='devops'"}, id="filter_query"),
        ],
    )
    def test_get_skills_translates_kwargs(
        self,
        client_with_mocked_skill_api: CatalogAPIClient,
        kwargs: dict,
        expected_call_kwargs: dict,
    ):
        """Test that each get_skills kwarg reaches find_skills with the
        expected translated value, and nothing else."""
        client_with_mocked_skill_api.get_skills(**kwargs)

        call_kwargs = client_with_mocked_skill_api.skill_api.find_skills.call_args.kwargs
        for key, expected_value in expected_call_kwargs.items():
            assert call_kwargs[key] == expected_value, f"find_skills({key}=...) got {call_kwargs[key]!r}, expected {expected_value!r}"

    def test_get_skill_passes_id(self, client_with_mocked_skill_api: CatalogAPIClient):
        """Test that get_skill forwards the skill_id to the generated client."""
        client_with_mocked_skill_api.get_skill(skill_id="42")

        call_kwargs = client_with_mocked_skill_api.skill_api.get_skill.call_args.kwargs
        assert call_kwargs["id"] == "42"


class TestSkillsNegative:
    """Test suite for skill error handling and negative cases."""

    def test_get_nonexistent_skill_404(
        self,
        api_client: CatalogAPIClient,
        suppress_ssl_warnings: None,
    ):
        """Test that requesting a non-existent skill returns 404."""
        with pytest.raises(CatalogNotFoundError):
            api_client.get_skill(skill_id="99999")
