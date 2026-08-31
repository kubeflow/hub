"""Pytest configuration and fixtures for catalog tests.

This module follows the model-registry Python client pattern:
- Assumes catalog service is already running (K8s, local, etc.)
- Uses environment variables for configuration
"""

import logging
import os
import time
from collections.abc import Generator
from pathlib import Path

import pytest
import requests
import urllib3
import yaml

from model_catalog import CatalogAPIClient

from .constants import (
    API_BASE_PATH,
    CATALOG_URL,
    CLIENT_TIMEOUT,
    MAX_BACKOFF,
    MAX_POLL_TIME,
    POLL_INTERVAL,
    get_verify_ssl,
)

# Configure logging
logging.basicConfig(
    format="%(asctime)s.%(msecs)03d - %(name)s:%(levelname)s: %(message)s",
    datefmt="%H:%M:%S",
    level=logging.WARNING,
)

logger = logging.getLogger("model-catalog")


def pytest_addoption(parser):
    """Add custom command line options."""
    parser.addoption("--e2e", action="store_true", help="run end-to-end tests")
    parser.addoption("--fuzz", action="store_true", help="run fuzzing tests")


def pytest_configure(config):
    """Register custom markers."""
    config.addinivalue_line("markers", "e2e: mark test as end-to-end test")
    config.addinivalue_line("markers", "fuzz: mark test as fuzzing test")
    config.addinivalue_line("markers", "huggingface: mark test as requiring HuggingFace API")


def _auto_mark_test(item) -> None:
    """Auto-mark tests based on their location."""
    path = str(item.fspath)
    if "fuzz_api" in path:
        item.add_marker(pytest.mark.fuzz)
    elif "tests" in path:
        item.add_marker(pytest.mark.e2e)


def pytest_collection_modifyitems(config, items):
    """Modify test collection based on markers and options."""
    e2e = config.getoption("--e2e")
    fuzz = config.getoption("--fuzz")

    selected = []
    deselected = []

    for item in items:
        _auto_mark_test(item)

        if e2e:
            if "e2e" in item.keywords:
                selected.append(item)
            else:
                deselected.append(item)
        elif fuzz:
            if "fuzz" in item.keywords:
                selected.append(item)
            else:
                deselected.append(item)
        else:
            # Default run (no flags): only collect unit tests, deselect e2e/fuzz
            if "e2e" in item.keywords or "fuzz" in item.keywords:
                deselected.append(item)
            else:
                selected.append(item)

    if deselected:
        config.hook.pytest_deselected(items=deselected)
        items[:] = selected


def pytest_report_teststatus(report, config):
    """Custom test status reporting."""
    if config.getoption("--quiet", default=False):
        return

    test_name = report.head_line
    if report.passed:
        if report.when == "call":
            print(f"\nTEST: {test_name} STATUS: \033[0;32mPASSED\033[0m")
    elif report.skipped:
        print(f"\nTEST: {test_name} STATUS: \033[1;33mSKIPPED\033[0m")
    elif report.failed:
        if report.when != "call":
            print(f"\nTEST: {test_name} [{report.when}] STATUS: \033[0;31mERROR\033[0m")
        else:
            print(f"\nTEST: {test_name} STATUS: \033[0;31mFAILED\033[0m")


# Maximum directory levels to traverse when searching for repo root
_MAX_PARENT_LEVELS = 10


@pytest.fixture(scope="session")
def root(request) -> Path:
    """Get repository root directory.

    Navigates up from catalog/clients/python to find the repo root.
    The repo root is identified by the presence of a .git directory.

    Raises:
        RuntimeError: If the repository root cannot be found.
    """

    current = request.config.rootpath
    # Walk up looking for .git directory (repo root marker)
    for _ in range(_MAX_PARENT_LEVELS):
        if (current / ".git").exists():
            return current
        current = current.parent
    # Fail explicitly if repo root not found
    msg = (
        f"Could not find repository root (.git directory) starting from "
        f"{request.config.rootpath}. Searched {_MAX_PARENT_LEVELS} levels up."
    )
    raise RuntimeError(msg)


@pytest.fixture(scope="session")
def user_token() -> str | None:
    """Get user token from environment."""
    return os.getenv("AUTH_TOKEN")


@pytest.fixture(scope="session")
def request_headers(user_token: str | None) -> dict[str, str]:
    """Get request headers including authorization if token is set."""
    headers = {"Content-Type": "application/json"}
    if user_token:
        headers["Authorization"] = f"Bearer {user_token}"
    return headers


@pytest.fixture(scope="session")
def verify_ssl() -> bool:
    """Get SSL verification setting from environment."""
    return get_verify_ssl(logger)


@pytest.fixture(scope="function")
def suppress_ssl_warnings(verify_ssl: bool) -> None:
    """Suppress urllib3 SSL warnings when SSL verification is disabled."""
    if not verify_ssl:
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)


@pytest.fixture(scope="session")
def kind_cluster() -> bool:
    """Get KIND_CLUSTER environment variable value.

    Returns:
        True if KIND_CLUSTER is not set or set to 'True', False if set to 'False'.
    """
    kind_cluster_env = os.getenv("KIND_CLUSTER", "True")
    return kind_cluster_env.lower() in ("true", "1", "yes")


def poll_for_ready(user_token: str | None, verify_ssl: bool) -> None:
    """Wait for catalog service to be ready using exponential backoff.

    Args:
        user_token: Optional auth token.
        verify_ssl: Whether to verify SSL certificates.
    """
    url = f"{CATALOG_URL}{API_BASE_PATH}/sources"
    params = {
        "url": url,
        "headers": {
            "Authorization": f"Bearer {user_token}",
        }
        if user_token
        else None,
        "verify": verify_ssl,
    }
    # Exponential backoff: start at POLL_INTERVAL, double each time, cap at MAX_BACKOFF
    backoff = POLL_INTERVAL
    poll_start = time.time()

    while True:
        elapsed_time = time.time() - poll_start
        if elapsed_time >= MAX_POLL_TIME:
            msg = f"Catalog service not ready after {int(elapsed_time)}s at {url}"
            logger.error(msg)
            raise TimeoutError(msg)
        logger.info("Attempting to connect to server %s", url)
        try:
            response = requests.get(**params, timeout=MAX_BACKOFF)
            if response.status_code < 500:  # Accept any non-5xx response
                logger.info("Server is up!")
                return
        except requests.exceptions.ConnectionError:
            pass

        time.sleep(backoff)
        backoff = min(backoff * 2, MAX_BACKOFF)  # Exponential backoff with cap


def poll_for_filter_options(user_token: str | None, verify_ssl: bool) -> None:
    """Wait until filter options are populated (materialized views refreshed).

    The catalog HTTP server becomes reachable before leader setup completes
    (migrations → model writes → OnLeaderReady → REFRESH MATERIALIZED VIEW).
    This guard ensures tests only start after the views are populated.

    Args:
        user_token: Optional auth token.
        verify_ssl: Whether to verify SSL certificates.
    """
    url = f"{CATALOG_URL}{API_BASE_PATH}/models/filter_options"
    params = {
        "url": url,
        "headers": {
            "Authorization": f"Bearer {user_token}",
        }
        if user_token
        else None,
        "verify": verify_ssl,
    }
    backoff = POLL_INTERVAL
    poll_start = time.time()

    while True:
        elapsed_time = time.time() - poll_start
        if elapsed_time >= MAX_POLL_TIME:
            logger.warning("Filter options still empty after %ds; proceeding anyway", int(elapsed_time))
            return
        logger.info("Waiting for filter options to be populated...")
        try:
            response = requests.get(**params, timeout=MAX_BACKOFF)
            if response.status_code == 200:
                try:
                    filters = response.json().get("filters", {})
                    if filters:
                        logger.info("Filter options ready (%d fields)", len(filters))
                        return
                except Exception:
                    pass
        except requests.exceptions.ConnectionError:
            pass

        time.sleep(backoff)
        backoff = min(backoff * 2, MAX_BACKOFF)


@pytest.fixture(scope="session")
def api_client(user_token: str | None, verify_ssl: bool) -> Generator[CatalogAPIClient, None, None]:
    """Create API client for the catalog service.

    This is a session-scoped fixture that connects to the already-running
    catalog service specified by CATALOG_URL environment variable.

    Timeout is configurable via CATALOG_CLIENT_TIMEOUT env var (default 30s).
    """
    if not verify_ssl:
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

    poll_for_ready(user_token=user_token, verify_ssl=verify_ssl)
    poll_for_filter_options(user_token=user_token, verify_ssl=verify_ssl)
    with CatalogAPIClient(
        CATALOG_URL, timeout=CLIENT_TIMEOUT, verify_ssl=verify_ssl, access_token=user_token
    ) as client:
        yield client


@pytest.fixture(scope="session")
def model_with_artifacts(
    request: pytest.FixtureRequest, api_client: CatalogAPIClient, verify_ssl: bool, kind_cluster: bool
) -> tuple[str, str]:
    """Get a model that has artifacts for testing.

    The param value controls the minimum number of artifacts required.
    Default is 1. Override via indirect parametrization for tests needing more.

    Returns:
        Tuple of (source_id, model_name) for a model with artifacts.

    Raises:
        pytest.fail: If no models are available or no model has enough artifacts.
    """
    min_artifacts = getattr(request, "param", 1)
    if not verify_ssl:
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

    models = api_client.get_models(page_size=1000 if not kind_cluster else None)
    if not models.get("items"):
        pytest.fail("No models available - test data may not be loaded")

    for model in models["items"]:
        source_id = model.get("sourceId")
        model_name = model.get("name")
        if not source_id or not model_name:
            continue

        artifacts = api_client.get_artifacts(source_id=source_id, model_name=model_name)
        items = artifacts.get("items", [])
        if len(items) >= min_artifacts:
            return source_id, model_name

    pytest.fail(f"No model with at least {min_artifacts} artifact(s) found")


@pytest.fixture(scope="session")
def testdata_dir(root) -> Path:
    """Get path to testdata directory."""
    return root / "test" / "testdata"


@pytest.fixture(scope="session")
def local_testdata_dir() -> Path:
    """Get path to local testdata directory (in tests/)."""
    return Path(__file__).parent / "testdata"


@pytest.fixture(scope="session")
def test_catalog_data(pytestconfig: pytest.Config) -> dict:
    """Load test catalog data used by E2E tests.

    Returns:
        Dictionary containing the test catalog YAML data.
    """
    test_catalog_path = (
        Path(pytestconfig.rootpath)
        / "../../.."
        / "manifests"
        / "kustomize"
        / "options"
        / "catalog"
        / "overlays"
        / "e2e"
        / "test-catalog.yaml"
    )
    with open(test_catalog_path) as f:
        return yaml.safe_load(f)


@pytest.fixture(scope="session")
def test_mcp_catalog_data(pytestconfig: pytest.Config) -> dict:
    """Load MCP server test catalog data used by E2E tests.

    Returns:
        Dictionary containing the MCP server test YAML data.
    """
    mcp_catalog_path = (
        Path(pytestconfig.rootpath)
        / "../../.."
        / "manifests"
        / "kustomize"
        / "options"
        / "catalog"
        / "overlays"
        / "e2e"
        / "test-mcp-servers.yaml"
    )
    with open(mcp_catalog_path) as f:
        return yaml.safe_load(f)


@pytest.fixture(scope="session")
def sources_config(pytestconfig: pytest.Config) -> dict:
    """Load the sources.yaml configuration used by E2E tests.

    Returns:
        Dictionary containing the sources YAML data (catalogs + mcp_catalogs).
    """
    sources_path = (
        Path(pytestconfig.rootpath)
        / "../../.."
        / "manifests"
        / "kustomize"
        / "options"
        / "catalog"
        / "overlays"
        / "e2e"
        / "sources.yaml"
    )
    with open(sources_path) as f:
        return yaml.safe_load(f)
