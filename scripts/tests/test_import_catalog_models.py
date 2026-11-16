"""Exercise the importer CLI against local catalog and registry HTTP services."""

import json
import os
from pathlib import Path
import subprocess
import sys
import threading
import time
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlsplit


SCRIPT = Path(__file__).resolve().parents[1] / "import_catalog_models.py"
CATALOG = "/api/model_catalog/v1"
REGISTRY = "/api/model_registry/v1"


class ImportCatalogModelsTest(unittest.TestCase):
    def setUp(self):
        self.models = [{"name": "example", "sourceId": "source"}]
        self.artifacts = [
            {"artifactType": "model-artifact", "uri": "s3://bucket/model"}
        ]
        self.requests = []
        self.created = []
        self.existing = set()
        self.override = None
        self.truncate_path = None
        owner = self

        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *_args):
                pass

            def do_GET(self):
                self.respond()

            def do_POST(self):
                self.respond()

            def respond(self):
                url = urlsplit(self.path)
                query = parse_qs(url.query)
                body = None
                if self.command == "POST":
                    body = json.loads(
                        self.rfile.read(int(self.headers["Content-Length"]))
                    )
                owner.requests.append(
                    (self.command, url.path, query, body, dict(self.headers))
                )
                response = (
                    owner.override(self.command, url.path, query, body)
                    if owner.override
                    else None
                )
                if response is None:
                    response = owner.route(self.command, url.path, query, body)
                status, data = response
                encoded = json.dumps(data).encode()
                self.send_response(status)
                if status == 302:
                    self.send_header("Location", owner.url + "/redirected")
                self.send_header("Content-Type", "application/json")
                truncated = url.path == owner.truncate_path
                self.send_header(
                    "Content-Length", str(len(encoded) + (10 if truncated else 0))
                )
                if truncated:
                    self.close_connection = True
                self.end_headers()
                try:
                    self.wfile.write(encoded)
                except BrokenPipeError:
                    pass  # Expected when testing a client timeout.

        self.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.addCleanup(self.server.server_close)
        self.addCleanup(self.thread.join)
        self.addCleanup(self.server.shutdown)
        self.url = f"http://127.0.0.1:{self.server.server_port}"

    def route(self, method, path, query, body):
        if path == CATALOG + "/models":
            return 200, {"items": self.models}
        if path.startswith(CATALOG) and path.endswith("/artifacts"):
            return 200, {"items": self.artifacts}
        if path == REGISTRY + "/registered_model":
            if query["name"][0] in self.existing:
                return 200, {"id": "99", "name": query["name"][0]}
            return 404, {"message": "Model not found"}
        if method == "POST":
            result = {**body, "id": str(len(self.created) + 1)}
            self.created.append((path, result))
            if path == REGISTRY + "/registered_models":
                self.existing.add(body["name"])
            return 201, result
        return 500, {"message": "Unexpected route"}

    def run_cli(self, *args):
        return subprocess.run(
            [
                sys.executable,
                str(SCRIPT),
                "--catalog-url",
                self.url,
                "--registry-url",
                self.url,
                *args,
            ],
            capture_output=True,
            text=True,
            timeout=10,
            env={
                **os.environ,
                "CATALOG_TOKEN": "catalog-secret",
                "REGISTRY_TOKEN": "registry-secret",
            },
        )

    def test_copies_metadata_and_all_model_artifacts_into_one_version(self):
        self.models[0].update(
            {
                "description": "A model",
                "readme": "# Model",
                "provider": "Example",
                "license": "Apache-2.0",
                "language": ["en"],
                "tasks": ["text-generation"],
                "id": "catalog-id",
                "externalId": "external",
                "createTimeSinceEpoch": "123",
                "customProperties": {
                    "size": {"metadataType": "MetadataIntValue", "int_value": "7"}
                },
            }
        )
        self.artifacts = [
            {
                "artifactType": "model-artifact",
                "uri": "s3://one",
                "name": "shared-name",
                "description": "Weights",
                "id": "old",
                "externalId": "old-external",
                "customProperties": {
                    "accuracy": {
                        "metadataType": "MetadataDoubleValue",
                        "double_value": 0.9,
                    }
                },
            },
            {"artifactType": "metrics-artifact", "uri": "s3://metrics"},
            {"artifactType": "model-artifact", "uri": "oci://two"},
        ]
        result = self.run_cli()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.created), 4)
        model = self.created[0][1]
        self.assertEqual(model["readme"], "# Model")
        self.assertEqual(model["tasks"], ["text-generation"])
        self.assertEqual(model["customProperties"]["size"]["int_value"], "7")
        self.assertNotIn("externalId", model)
        self.assertNotIn("createTimeSinceEpoch", model)
        self.assertEqual(self.created[1][1]["name"], "1")
        self.assertEqual(self.created[1][1]["registeredModelId"], "1")
        self.assertEqual(self.created[2][0], REGISTRY + "/model_versions/2/artifacts")
        self.assertEqual(self.created[3][0], REGISTRY + "/model_versions/2/artifacts")
        self.assertEqual(self.created[2][1]["uri"], "s3://one")
        self.assertEqual(self.created[3][1]["uri"], "oci://two")
        self.assertNotIn("name", self.created[2][1])
        self.assertNotIn("externalId", self.created[2][1])
        self.assertEqual(
            self.created[2][1]["customProperties"]["accuracy"]["double_value"], 0.9
        )

    def test_paginates_models_and_artifacts_and_encodes_path_segments(self):
        self.models = [{"name": "org/model + x", "sourceId": "source/one"}]

        def pages(method, path, query, body):
            if path == CATALOG + "/models":
                if "nextPageToken" in query:
                    self.assertEqual(query["nextPageToken"], ["models+next"])
                    return 200, {"items": [{"name": "second", "sourceId": "source"}]}
                return 200, {"items": self.models, "nextPageToken": "models+next"}
            if path.endswith("/artifacts") and method == "GET":
                if "nextPageToken" in query:
                    return 200, {
                        "items": [{"artifactType": "model-artifact", "uri": "s3://two"}]
                    }
                return 200, {"items": self.artifacts, "nextPageToken": "artifacts-next"}
            return None

        self.override = pages
        result = self.run_cli()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.created), 8)
        self.assertTrue(
            any(
                "source%2Fone/models/org%2Fmodel%20%2B%20x/artifacts" in r[1]
                for r in self.requests
            )
        )

    def test_skips_existing_models_and_cross_source_name_collisions(self):
        self.models = [
            {"name": "existing", "sourceId": "a"},
            {"name": "new", "sourceId": "a"},
            {"name": "new", "sourceId": "b"},
        ]
        self.existing.add("existing")
        result = self.run_cli()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.created), 3)
        self.assertIn("skipped=2", result.stdout)

    def test_selection_prefix_version_and_author(self):
        self.models.append({"name": "second", "sourceId": "source"})
        result = self.run_cli(
            "--source",
            "source",
            "--limit",
            "1",
            "--name-prefix",
            "test-",
            "--version",
            "v2",
            "--author",
            "Tester",
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created[0][1]["name"], "test-example")
        self.assertEqual(self.created[1][1]["name"], "v2")
        self.assertEqual(self.created[1][1]["author"], "Tester")
        self.assertEqual(self.requests[0][2]["source"], ["source"])
        self.assertEqual(len(self.created), 3)

    def test_skips_models_without_uris_and_omits_empty_artifacts(self):
        self.artifacts = [{"artifactType": "model-artifact", "uri": " "}]
        result = self.run_cli()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created, [])
        self.assertIn("skipped=1", result.stdout)
        self.artifacts.append({"artifactType": "model-artifact", "uri": "s3://valid"})
        result = self.run_cli()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(self.created), 3)

    def test_dry_run_does_not_write_and_counts_duplicate_names(self):
        self.models.append({"name": "example", "sourceId": "other"})
        result = self.run_cli("--dry-run")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.created, [])
        self.assertFalse(any(r[0] == "POST" for r in self.requests))
        self.assertIn("skipped=1", result.stdout)
        self.assertIn("Would import", result.stdout)

    def test_sends_separate_tokens(self):
        result = self.run_cli()
        self.assertEqual(result.returncode, 0, result.stderr)
        for _, path, _, _, headers in self.requests:
            expected = (
                "Bearer catalog-secret"
                if path.startswith(CATALOG)
                else "Bearer registry-secret"
            )
            self.assertEqual(headers["Authorization"], expected)

    def test_authentication_failure_stops_import(self):
        self.models.append({"name": "second", "sourceId": "source"})
        self.override = (
            lambda method, path, query, body: (401, {"message": "Unauthorized"})
            if path.startswith(REGISTRY)
            else None
        )
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.created, [])
        self.assertEqual(
            len([r for r in self.requests if r[1].startswith(REGISTRY)]), 1
        )
        self.assertNotIn("registry-secret", result.stderr)

    def test_partial_creation_is_reported_without_retry_and_other_models_continue(self):
        self.models.append({"name": "second", "sourceId": "source"})
        self.override = (
            lambda method, path, query, body: (500, {"message": "Failed"})
            if path == REGISTRY + "/model_versions/2/artifacts"
            else None
        )
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("registered_model_id=1", result.stderr)
        self.assertIn("model_version_id=2", result.stderr)
        self.assertEqual(
            len(
                [
                    r
                    for r in self.requests
                    if r[1] == REGISTRY + "/model_versions/2/artifacts"
                ]
            ),
            1,
        )
        self.assertIn("imported=1", result.stdout)
        self.assertIn("failed=1", result.stdout)

    def test_catalog_pagination_failure_stops_import(self):
        def fail_second_page(method, path, query, body):
            if path == CATALOG + "/models":
                if "nextPageToken" in query:
                    return 500, {"message": "Failed"}
                return 200, {"items": self.models, "nextPageToken": "next"}
            return None

        self.override = fail_second_page
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("imported=1", result.stdout)

    def test_request_timeout_exits_with_failure(self):
        def slow_response(method, path, query, body):
            time.sleep(0.15)
            return 200, {"items": []}

        self.override = slow_response
        result = self.run_cli("--timeout", "0.05")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("timed out", result.stderr)
        self.assertEqual(self.created, [])

    def test_artifact_pagination_failure_prevents_creating_partial_model(self):
        def fail_artifacts(method, path, query, body):
            if method == "GET" and path.endswith("/artifacts"):
                if "nextPageToken" in query:
                    return 500, {"message": "Failed"}
                return 200, {"items": self.artifacts, "nextPageToken": "next"}
            return None

        self.override = fail_artifacts
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.created, [])

    def test_repeated_pagination_token_stops_instead_of_looping(self):
        self.override = lambda method, path, query, body: (
            (200, {"items": [], "nextPageToken": "repeated"})
            if path == CATALOG + "/models"
            else None
        )
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(self.requests), 2)

    def test_invalid_options_fail_before_any_requests(self):
        for args in [
            ("--limit", "0"),
            ("--timeout", "nan"),
            ("--timeout", "-1"),
            ("--version", " "),
            ("--catalog-url", "ftp://example.com"),
        ]:
            with self.subTest(args=args):
                result = self.run_cli(*args)
                self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.requests, [])

    def test_missing_ca_file_reports_configuration_error(self):
        result = self.run_cli(
            "--registry-ca", str(SCRIPT.parent / "nonexistent-ca.pem")
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Configuration error", result.stderr)
        self.assertEqual(self.requests, [])

    def test_redirect_is_not_followed_with_service_credentials(self):
        # A redirect must not forward a bearer token to another endpoint.
        self.override = lambda method, path, query, body: (302, {"message": "Moved"})
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(self.requests), 1)

    def test_truncated_write_response_reports_partial_ids_and_continues(self):
        self.models.append({"name": "second", "sourceId": "source"})
        self.truncate_path = REGISTRY + "/model_versions/2/artifacts"
        result = self.run_cli()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("registered_model_id=1", result.stderr)
        self.assertIn("model_version_id=2", result.stderr)
        self.assertIn("imported=1", result.stdout)
        self.assertIn("failed=1", result.stdout)
        self.assertNotIn("Traceback", result.stderr)


if __name__ == "__main__":
    unittest.main()
