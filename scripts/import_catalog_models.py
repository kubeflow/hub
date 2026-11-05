#!/usr/bin/env python3
r"""Seed a model registry with metadata and artifact URIs from a model catalog.

Requires Python 3.10+; no packages need to be installed. Model files are never
downloaded. Each imported model has one version containing all model artifacts;
catalog metrics artifacts are excluded. Existing registry models are skipped.

Examples (run from the repository root)::

    python3 scripts/import_catalog_models.py \
        --catalog-url http://localhost:8081 --registry-url http://localhost:8080 \
        --dry-run

    python3 scripts/import_catalog_models.py \
        --catalog-url http://localhost:8081 --registry-url http://localhost:8080 \
        --source my-source --limit 20 --name-prefix test- --version 1

For authenticated services, set CATALOG_TOKEN and REGISTRY_TOKEN independently
in the environment. HTTPS verifies certificates; use --catalog-ca/--registry-ca
for private CA files. URLs are service base URLs, without API paths.

Imports are not transactional. Failed writes are never retried automatically.
On failure, created record IDs are reported; inspect partial records manually.
Reruns skip those existing models too. Use a new --name-prefix for a fresh set.
If sources share a model name, the first imported entry wins. --limit counts
catalog models considered, including skips and failures. Dry runs still read
both services but perform no writes.
"""

import argparse
from http.client import HTTPException
from itertools import islice
import json
import math
import os
import ssl
import sys
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlencode, urlsplit
from urllib.request import HTTPRedirectHandler, HTTPSHandler, Request, build_opener


CATALOG = "/api/model_catalog/v1"
REGISTRY = "/api/model_registry/v1"
MODEL_FIELDS = (
    "description",
    "readme",
    "maturity",
    "language",
    "tasks",
    "provider",
    "logo",
    "license",
    "licenseLink",
    "libraryName",
    "customProperties",
)
ARTIFACT_FIELDS = ("uri", "description", "customProperties")


class APIError(Exception):
    """An HTTP request or response could not be completed."""

    def __init__(self, message, status=None):
        super().__init__(message)
        self.status = status


class PaginationError(APIError):
    """Catalog traversal failed; stop rather than silently omit data."""


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        # Keep service credentials on the configured endpoint and never replay writes.
        return None


class API:
    def __init__(self, base_url, token, ca_file, timeout):
        self.base_url = base_url.rstrip("/")
        self.token = token
        context = ssl.create_default_context(cafile=ca_file)
        self.opener = build_opener(HTTPSHandler(context=context), NoRedirect())
        self.timeout = timeout

    def request(self, method, path, *, params=None, body=None):
        url = self.base_url + path
        if params:
            url += "?" + urlencode(params)
        headers = {"Accept": "application/json"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        data = None
        if body is not None:
            headers["Content-Type"] = "application/json"
            data = json.dumps(body).encode("utf-8")
        request = Request(url, data=data, headers=headers, method=method)
        try:
            with self.opener.open(request, timeout=self.timeout) as response:
                result = json.load(response)
        except HTTPError as error:
            # Do not print response bodies: an upstream service might echo credentials.
            status = error.code
            error.close()
            raise APIError(f"{method} {path}: HTTP {status}", status) from error
        except (URLError, OSError, ValueError, HTTPException) as error:
            raise APIError(f"{method} {path}: {error}") from error
        if not isinstance(result, dict):
            raise APIError(f"{method} {path}: expected a JSON object")
        return result

    def items(self, path, **params):
        params = {"pageSize": "100", **params}
        seen_tokens = set()
        while True:
            try:
                page = self.request("GET", path, params=params)
                items = page.get("items", [])
                if not isinstance(items, list) or any(
                    not isinstance(item, dict) for item in items
                ):
                    raise APIError(f"GET {path}: expected a list of objects in items")
                token = page.get("nextPageToken")
                if token and (not isinstance(token, str) or token in seen_tokens):
                    raise APIError(f"GET {path}: invalid or repeated pagination token")
            except APIError as error:
                raise PaginationError(str(error), error.status) from error
            yield from items
            if not token:
                return
            seen_tokens.add(token)
            params["nextPageToken"] = token


def copy_fields(resource, fields):
    return {key: resource[key] for key in fields if resource.get(key) is not None}


def created_id(response):
    value = response.get("id")
    if not isinstance(value, str) or not value:
        raise APIError(
            "Creation response has no record ID; inspect the registry for partial data"
        )
    return value


def import_models(args, catalog, registry):
    counts = dict(imported=0, skipped=0, failed=0, artifacts=0, planned=0)
    seen_names = set()
    params = {"source": args.source} if args.source else {}
    models = catalog.items(CATALOG + "/models", **params)
    if args.limit is not None:
        models = islice(models, args.limit)
    try:
        for model in models:
            name = args.name_prefix + model.get("name", "")
            record_ids = []
            try:
                if not model.get("name") or not model.get("sourceId"):
                    raise APIError("Catalog model is missing name or sourceId")
                if name in seen_names:
                    counts["skipped"] += 1
                    print(f"Skipped {name!r}: name already encountered")
                    continue
                try:
                    registry.request(
                        "GET", REGISTRY + "/registered_model", params={"name": name}
                    )
                except APIError as error:
                    if error.status != 404:
                        raise
                else:
                    seen_names.add(name)
                    counts["skipped"] += 1
                    print(f"Skipped {name!r}: registry model already exists")
                    continue

                path = (
                    f"{CATALOG}/sources/{quote(model['sourceId'], safe='')}/models/"
                    f"{quote(model['name'], safe='')}/artifacts"
                )
                artifacts = []
                for artifact in catalog.items(path, artifactType="model-artifact"):
                    if artifact.get("artifactType") != "model-artifact":
                        continue
                    uri = artifact.get("uri")
                    if not isinstance(uri, str) or not uri.strip():
                        print(
                            f"Omitted artifact for {name!r}: no usable URI",
                            file=sys.stderr,
                        )
                        continue
                    artifacts.append(
                        {
                            "artifactType": "model-artifact",
                            **copy_fields(artifact, ARTIFACT_FIELDS),
                        }
                    )
                if not artifacts:
                    counts["skipped"] += 1
                    print(f"Skipped {name!r}: no usable model artifact URIs")
                    continue
                if args.dry_run:
                    seen_names.add(name)
                    counts["planned"] += 1
                    print(
                        f"Would import {name!r}: version={args.version!r}, artifacts={len(artifacts)}"
                    )
                    continue

                response = registry.request(
                    "POST",
                    REGISTRY + "/registered_models",
                    body={
                        "name": name,
                        **copy_fields(model, MODEL_FIELDS),
                    },
                )
                model_id = created_id(response)
                record_ids.append(f"registered_model_id={model_id}")
                seen_names.add(name)
                response = registry.request(
                    "POST",
                    REGISTRY + "/model_versions",
                    body={
                        "name": args.version,
                        "author": args.author,
                        "registeredModelId": model_id,
                    },
                )
                version_id = created_id(response)
                record_ids.append(f"model_version_id={version_id}")
                for artifact in artifacts:
                    response = registry.request(
                        "POST",
                        f"{REGISTRY}/model_versions/{quote(version_id, safe='')}/artifacts",
                        body=artifact,
                    )
                    artifact_id = created_id(response)
                    record_ids.append(f"artifact_id={artifact_id}")
                    counts["artifacts"] += 1
                counts["imported"] += 1
                print(f"Imported {name!r}: {', '.join(record_ids)}")
            except APIError as error:
                counts["failed"] += 1
                partial = f"; created: {', '.join(record_ids)}" if record_ids else ""
                print(f"Failed {name!r}: {error}{partial}", file=sys.stderr)
                if isinstance(error, PaginationError) or error.status in (401, 403):
                    return 1
    except APIError as error:
        counts["failed"] += 1
        print(f"Stopped: {error}", file=sys.stderr)
    finally:
        print(
            "Summary: " + ", ".join(f"{key}={value}" for key, value in counts.items())
        )
    return 1 if counts["failed"] else 0


def service_url(value):
    url = urlsplit(value)
    if (
        url.scheme not in ("http", "https")
        or not url.hostname
        or url.query
        or url.fragment
        or url.username
        or url.password
    ):
        raise argparse.ArgumentTypeError(
            "expected an HTTP(S) base URL without credentials, query, or fragment"
        )
    return value.rstrip("/")


def positive_int(value):
    number = int(value)
    if number <= 0:
        raise argparse.ArgumentTypeError("must be a positive integer")
    return number


def positive_timeout(value):
    number = float(value)
    if not math.isfinite(number) or number <= 0:
        raise argparse.ArgumentTypeError("must be a positive finite number of seconds")
    return number


def main():
    parser = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("--catalog-url", required=True, type=service_url)
    parser.add_argument("--registry-url", required=True, type=service_url)
    parser.add_argument("--source", help="Import only this catalog source ID")
    parser.add_argument(
        "--limit",
        type=positive_int,
        help="Maximum catalog models considered (default: all)",
    )
    parser.add_argument(
        "--name-prefix", default="", help="Prefix for destination model names"
    )
    parser.add_argument(
        "--version", default="1", help="Destination version name (default: 1)"
    )
    parser.add_argument(
        "--author",
        default="catalog-import",
        help="Version author (default: catalog-import)",
    )
    parser.add_argument("--catalog-ca", help="PEM CA file for the catalog")
    parser.add_argument("--registry-ca", help="PEM CA file for the registry")
    parser.add_argument(
        "--timeout",
        type=positive_timeout,
        default=30,
        help="Per-request timeout in seconds (default: 30)",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show intended imports without writing to the registry",
    )
    args = parser.parse_args()
    if not args.version.strip():
        parser.error("--version must not be empty")
    try:
        catalog = API(
            args.catalog_url,
            os.environ.get("CATALOG_TOKEN"),
            args.catalog_ca,
            args.timeout,
        )
        registry = API(
            args.registry_url,
            os.environ.get("REGISTRY_TOKEN"),
            args.registry_ca,
            args.timeout,
        )
        return import_models(args, catalog, registry)
    except (OSError, ValueError) as error:
        print(f"Configuration error: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
