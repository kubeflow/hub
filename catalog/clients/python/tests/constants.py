"""Test constants and configuration for catalog E2E tests."""

import logging
import os

# Catalog service URL (from environment or default)
CATALOG_URL = os.environ.get("CATALOG_URL", "http://localhost:8081")

# API base path - keep in sync with server if API version changes
API_BASE_PATH = "/api/model_catalog/v1"

# Client timeout for E2E tests (default 30s is more generous than library default of 10s)
CLIENT_TIMEOUT = int(os.environ.get("CATALOG_CLIENT_TIMEOUT", "30"))

# Polling configuration (for waiting on service readiness)
# These can be overridden via environment variables for slow/CI environments
MAX_POLL_TIME = int(os.environ.get("CATALOG_POLL_TIMEOUT", "60"))  # seconds
POLL_INTERVAL = int(os.environ.get("CATALOG_POLL_INTERVAL", "1"))  # seconds (initial backoff)
MAX_BACKOFF = int(os.environ.get("CATALOG_MAX_BACKOFF", "10"))  # seconds (max backoff)


# Required fields that every MCP server response must contain
MCP_SERVER_REQUIRED_FIELDS: set[str] = {"id", "name", "provider", "description"}


def get_verify_ssl(logger: logging.Logger | None = None) -> bool:
    """Get SSL verification setting from environment.

    Args:
        logger: Optional logger for warning messages.

    Returns:
        True if SSL should be verified (default), False only if VERIFY_SSL=false.
    """
    verify_ssl_env = os.environ.get("VERIFY_SSL")
    if verify_ssl_env is None:
        return True
    # Only disable SSL when explicitly set to common "falsy" values
    # Any other value (including "true", "yes", "1") keeps SSL enabled
    verify = verify_ssl_env.lower() not in ("false", "0", "no", "off")
    if not verify and logger:
        logger.warning("SSL verification is DISABLED (VERIFY_SSL=false). This is insecure!")
    return verify
