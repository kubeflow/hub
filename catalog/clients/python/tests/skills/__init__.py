"""Skill catalog E2E tests.

Test Structure:
    - test_skills.py: Skill listing, retrieval, pagination, and error handling

Note: unlike models and MCP servers, skill sources are git-backed
(type: git-skills-plugin) rather than static YAML fixtures, and the e2e
overlay (manifests/kustomize/options/catalog/overlays/e2e/) does not yet
register a skill_catalogs source. These tests are therefore structural
(they don't assert on specific seeded skill names/counts) and pass whether
or not any skills happen to be loaded — see kep-0005 SKC-601 for
connected e2e coverage against real skill data.
"""

__all__: list[str] = []
