#!/usr/bin/env python3
"""Export Model Catalog metadata to CSV.

Standalone script — requires only Python 3.10+ standard library.
Run: python model_catalog_export.py --url <base-url> --output models.csv
"""

import argparse
import contextlib
import csv
import json
import os
import sys
import tempfile
import urllib.error
import urllib.parse
import urllib.request

EXCLUDED_FIELDS = {"readme", "logo", "servingConfig", "customProperties"}

COLUMN_ORDER = [
    "id",
    "name",
    "description",
    "provider",
    "maturity",
    "license",
    "licenseLink",
    "libraryName",
    "source_id",
    "externalId",
    "createTimeSinceEpoch",
    "lastUpdateTimeSinceEpoch",
    "language",
    "tasks",
    "validatedTasks",
]

API_PATH = "/api/model_catalog/v1alpha1/models"
SOURCES_API_PATH = "/api/model_catalog/v1alpha1/sources"

UTF8_BOM = "\ufeff"


def build_url(base_url, page_size, next_page_token=None, sources=None):
    url = base_url.rstrip("/") + API_PATH
    params = {"pageSize": str(page_size)}
    if next_page_token:
        params["nextPageToken"] = next_page_token
    if sources:
        params["source"] = sources
    return f"{url}?{urllib.parse.urlencode(params, doseq=True)}"


def fetch_page(base_url, page_size, next_page_token=None, sources=None, headers=None):
    url = build_url(base_url, page_size, next_page_token, sources)
    req = urllib.request.Request(url)
    req.add_header("Accept", "application/json")
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = ""
        with contextlib.suppress(Exception):
            body = e.read().decode("utf-8", errors="replace")[:500]
        msg = f"Error: HTTP {e.code} from {url}\n{body}"
        raise SystemExit(msg) from None
    except urllib.error.URLError as e:
        msg = f"Error: cannot connect to {url}: {e.reason}"
        raise SystemExit(msg) from None


def paginate_models(base_url, page_size, sources=None, limit=None, headers=None):
    next_page_token = None
    count = 0
    while True:
        data = fetch_page(base_url, page_size, next_page_token, sources, headers)
        items = data.get("items") or []
        for model in items:
            yield model
            count += 1
            if limit and count >= limit:
                return
        next_page_token = data.get("nextPageToken", "")
        if not next_page_token:
            return


def extract_custom_property_value(metadata_value):
    if not isinstance(metadata_value, dict):
        return str(metadata_value)
    meta_type = metadata_value.get("metadataType", "")
    if meta_type == "MetadataStringValue":
        return metadata_value.get("string_value", "")
    if meta_type == "MetadataDoubleValue":
        return metadata_value.get("double_value", "")
    if meta_type == "MetadataIntValue":
        return metadata_value.get("int_value", "")
    if meta_type == "MetadataBoolValue":
        return metadata_value.get("bool_value", "")
    return json.dumps(metadata_value, ensure_ascii=False)


def format_field(value):
    if value is None:
        return ""
    if isinstance(value, list):
        return json.dumps(value, ensure_ascii=False)
    if isinstance(value, dict):
        return json.dumps(value, ensure_ascii=False)
    if isinstance(value, bool):
        return str(value).lower()
    return str(value)


def discover_columns(models):
    if not models:
        return list(COLUMN_ORDER)
    seen: set[str] = set()
    for model in models:
        seen.update(k for k in model if k not in EXCLUDED_FIELDS)
    ordered = [k for k in COLUMN_ORDER if k in seen]
    ordered += sorted(seen - set(COLUMN_ORDER))
    return ordered


def collect_models_and_keys(base_url, page_size, sources, limit, headers):
    models = []
    custom_keys: set[str] = set()
    for model in paginate_models(base_url, page_size, sources, limit, headers):
        models.append(model)
        props = model.get("customProperties") or {}
        custom_keys.update(props.keys())
    return models, discover_columns(models), sorted(custom_keys)


def model_to_row(model, columns, custom_keys):
    row = []
    for col in columns:
        row.append(format_field(model.get(col)))
    props = model.get("customProperties") or {}
    for key in custom_keys:
        if key in props:
            row.append(format_field(extract_custom_property_value(props[key])))
        else:
            row.append("")
    return row


def write_csv(models, columns, custom_keys, output_path):
    output_dir = os.path.dirname(os.path.abspath(output_path))
    os.makedirs(output_dir, exist_ok=True)
    tmp_fd, tmp_path = tempfile.mkstemp(suffix=".csv", dir=output_dir)
    try:
        with os.fdopen(tmp_fd, "w", newline="", encoding="utf-8") as f:
            f.write(UTF8_BOM)
            writer = csv.writer(f)
            header = list(columns) + custom_keys
            writer.writerow(header)
            for model in models:
                writer.writerow(model_to_row(model, columns, custom_keys))
        os.replace(tmp_path, os.path.abspath(output_path))
    except BaseException:
        with contextlib.suppress(OSError):
            os.unlink(tmp_path)
        raise


def fetch_sources(base_url, headers=None):
    url = base_url.rstrip("/") + SOURCES_API_PATH
    req = urllib.request.Request(url)
    req.add_header("Accept", "application/json")
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    try:
        with urllib.request.urlopen(req) as resp:
            data = json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body = ""
        with contextlib.suppress(Exception):
            body = e.read().decode("utf-8", errors="replace")[:500]
        msg = f"Error: HTTP {e.code} from {url}\n{body}"
        raise SystemExit(msg) from None
    except urllib.error.URLError as e:
        msg = f"Error: cannot connect to {url}: {e.reason}"
        raise SystemExit(msg) from None
    return data.get("items") or []


def print_sources(sources):
    if not sources:
        print("No sources found.", file=sys.stderr)
        return
    id_width = max(len("ID"), max(len(s.get("id", "")) for s in sources))
    name_width = max(len("NAME"), max(len(s.get("name", "")) for s in sources))
    print(f"{'ID':<{id_width}}  {'NAME':<{name_width}}  STATUS", file=sys.stderr)
    for s in sources:
        print(
            f"{s.get('id', ''):<{id_width}}  "
            f"{s.get('name', ''):<{name_width}}  "
            f"{s.get('status', '')}",
            file=sys.stderr,
        )


def parse_header(value):
    if ":" not in value:
        msg = f"Invalid header format: '{value}'. Expected 'Name: Value'."
        raise argparse.ArgumentTypeError(msg)
    key, _, val = value.partition(":")
    return key.strip(), val.strip()


def parse_args(argv=None):
    parser = argparse.ArgumentParser(
        prog="model-catalog-export",
        description="Export Model Catalog metadata to CSV.",
    )
    parser.add_argument(
        "--url",
        required=True,
        help="Catalog API base URL (e.g., http://localhost:8080)",
    )
    parser.add_argument(
        "--output",
        default=None,
        help="Output CSV file path",
    )
    parser.add_argument(
        "--source",
        action="append",
        default=None,
        help="Filter by source ID (repeatable)",
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=None,
        help="Maximum number of models to export (must be >= 1)",
    )
    parser.add_argument(
        "--page-size",
        type=int,
        default=100,
        help="Number of models per API page (default: 100, must be >= 1)",
    )
    parser.add_argument(
        "--header",
        action="append",
        type=parse_header,
        default=None,
        dest="headers",
        help='HTTP header as "Name: Value" (repeatable)',
    )
    parser.add_argument(
        "--list-sources",
        action="store_true",
        default=False,
        help="List available sources and exit",
    )
    return parser.parse_args(argv)


def validate_url_scheme(url):
    if not url.startswith(("http://", "https://")):
        msg = "Error: --url must use http:// or https:// scheme"
        raise SystemExit(msg)


def main(argv=None):
    args = parse_args(argv)
    validate_url_scheme(args.url)
    if args.limit is not None and args.limit < 1:
        msg = "Error: --limit must be >= 1"
        raise SystemExit(msg)
    if args.page_size < 1:
        msg = "Error: --page-size must be >= 1"
        raise SystemExit(msg)
    headers = dict(args.headers) if args.headers else None

    if args.list_sources:
        try:
            sources = fetch_sources(args.url, headers)
        except SystemExit:
            raise
        except Exception as e:
            print(f"Error: {e}", file=sys.stderr)
            raise SystemExit(1) from None
        print_sources(sources)
        return

    if not args.output:
        print("Error: --output is required (unless using --list-sources)", file=sys.stderr)
        raise SystemExit(2)

    try:
        models, columns, custom_keys = collect_models_and_keys(
            base_url=args.url,
            page_size=args.page_size,
            sources=args.source,
            limit=args.limit,
            headers=headers,
        )
        write_csv(models, columns, custom_keys, args.output)
    except SystemExit:
        raise
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        raise SystemExit(1) from None

    print(
        f"Exported {len(models)} models "
        f"({len(columns) + len(custom_keys)} columns) "
        f"to {args.output}",
        file=sys.stderr,
    )


if __name__ == "__main__":
    main()
