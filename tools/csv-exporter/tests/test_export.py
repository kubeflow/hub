import csv
import io
import json
import os
import sys
import tempfile
import urllib.error
from unittest import mock

import pytest

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from model_catalog_export import (
    FIXED_COLUMNS,
    build_url,
    collect_models_and_keys,
    extract_custom_property_value,
    format_field,
    main,
    model_to_row,
    paginate_models,
    parse_args,
    write_csv,
)


def make_api_response(items, next_page_token=""):
    return {
        "items": items,
        "size": len(items),
        "pageSize": 100,
        "nextPageToken": next_page_token,
    }


def make_model(name, custom_props=None, **overrides):
    model = {
        "id": f"id-{name}",
        "name": name,
        "description": f"Description of {name}",
        "provider": "TestProvider",
        "maturity": "Generally Available",
        "license": "apache-2.0",
        "licenseLink": "https://www.apache.org/licenses/LICENSE-2.0",
        "libraryName": "transformers",
        "source_id": "test-source",
        "externalId": f"ext-{name}",
        "createTimeSinceEpoch": "1609459200000",
        "lastUpdateTimeSinceEpoch": "1609459200000",
        "language": ["en"],
        "tasks": ["text-generation"],
        "validatedTasks": ["text-generation"],
    }
    if custom_props:
        model["customProperties"] = custom_props
    model.update(overrides)
    return model


def mock_urlopen(responses):
    call_count = {"n": 0}

    def _urlopen(req):
        idx = call_count["n"]
        call_count["n"] += 1
        data = json.dumps(responses[idx]).encode("utf-8")
        resp = mock.MagicMock()
        resp.status = 200
        resp.read.return_value = data
        resp.__enter__ = mock.MagicMock(return_value=resp)
        resp.__exit__ = mock.MagicMock(return_value=False)
        return resp

    return _urlopen


class TestBuildUrl:
    def test_basic(self):
        url = build_url("http://localhost:8080", 50)
        assert url == "http://localhost:8080/api/model_catalog/v1alpha1/models?pageSize=50"

    def test_with_trailing_slash(self):
        url = build_url("http://localhost:8080/", 50)
        assert url == "http://localhost:8080/api/model_catalog/v1alpha1/models?pageSize=50"

    def test_with_next_page_token(self):
        url = build_url("http://localhost:8080", 50, next_page_token="abc123")
        assert "pageSize=50" in url
        assert "nextPageToken=abc123" in url

    def test_with_sources(self):
        url = build_url("http://localhost:8080", 50, sources=["src1", "src2"])
        assert "source=src1" in url
        assert "source=src2" in url


class TestExtractCustomPropertyValue:
    def test_string_value(self):
        val = {"metadataType": "MetadataStringValue", "string_value": "pytorch"}
        assert extract_custom_property_value(val) == "pytorch"

    def test_double_value(self):
        val = {"metadataType": "MetadataDoubleValue", "double_value": 0.95}
        assert extract_custom_property_value(val) == 0.95

    def test_int_value(self):
        val = {"metadataType": "MetadataIntValue", "int_value": "42"}
        assert extract_custom_property_value(val) == "42"

    def test_bool_value(self):
        val = {"metadataType": "MetadataBoolValue", "bool_value": True}
        assert extract_custom_property_value(val) is True

    def test_proto_value(self):
        val = {"metadataType": "MetadataProtoValue", "type": "some.type", "proto_value": "abc"}
        result = extract_custom_property_value(val)
        parsed = json.loads(result)
        assert parsed["metadataType"] == "MetadataProtoValue"

    def test_struct_value(self):
        val = {"metadataType": "MetadataStructValue", "struct_value": "encoded"}
        result = extract_custom_property_value(val)
        parsed = json.loads(result)
        assert parsed["metadataType"] == "MetadataStructValue"

    def test_plain_string_fallback(self):
        assert extract_custom_property_value("raw") == "raw"


class TestFormatField:
    def test_none(self):
        assert format_field(None) == ""

    def test_string(self):
        assert format_field("hello") == "hello"

    def test_number(self):
        assert format_field(42) == "42"

    def test_bool(self):
        assert format_field(True) == "true"
        assert format_field(False) == "false"

    def test_list(self):
        result = format_field(["en", "es"])
        assert json.loads(result) == ["en", "es"]

    def test_dict(self):
        result = format_field({"key": "val"})
        assert json.loads(result) == {"key": "val"}


class TestPagination:
    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_single_page(self, mock_open):
        models = [make_model("m1"), make_model("m2")]
        mock_open.side_effect = mock_urlopen([make_api_response(models)])
        result = list(paginate_models("http://localhost:8080", 100))
        assert len(result) == 2
        assert result[0]["name"] == "m1"

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_multi_page(self, mock_open):
        page1 = make_api_response([make_model("m1")], next_page_token="page2")
        page2 = make_api_response([make_model("m2")], next_page_token="page3")
        page3 = make_api_response([make_model("m3")])
        mock_open.side_effect = mock_urlopen([page1, page2, page3])
        result = list(paginate_models("http://localhost:8080", 100))
        assert len(result) == 3

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_empty_catalog(self, mock_open):
        mock_open.side_effect = mock_urlopen([make_api_response([])])
        result = list(paginate_models("http://localhost:8080", 100))
        assert len(result) == 0

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_limit(self, mock_open):
        page1 = make_api_response([make_model("m1"), make_model("m2")], next_page_token="p2")
        page2 = make_api_response([make_model("m3")])
        mock_open.side_effect = mock_urlopen([page1, page2])
        result = list(paginate_models("http://localhost:8080", 100, limit=2))
        assert len(result) == 2

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_limit_across_pages(self, mock_open):
        page1 = make_api_response([make_model("m1")], next_page_token="p2")
        page2 = make_api_response([make_model("m2"), make_model("m3")])
        mock_open.side_effect = mock_urlopen([page1, page2])
        result = list(paginate_models("http://localhost:8080", 100, limit=2))
        assert len(result) == 2

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_source_filter(self, mock_open):
        mock_open.side_effect = mock_urlopen([make_api_response([])])
        list(paginate_models("http://localhost:8080", 100, sources=["my-source"]))
        req = mock_open.call_args[0][0]
        assert "source=my-source" in req.full_url


class TestCollectModelsAndKeys:
    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_discovers_custom_keys(self, mock_open):
        m1 = make_model("m1", {"framework": {"metadataType": "MetadataStringValue", "string_value": "pt"}})
        m2 = make_model("m2", {"accuracy": {"metadataType": "MetadataDoubleValue", "double_value": 0.9}})
        mock_open.side_effect = mock_urlopen([make_api_response([m1, m2])])
        models, keys = collect_models_and_keys("http://localhost:8080", 100, None, None, None)
        assert len(models) == 2
        assert keys == ["accuracy", "framework"]


class TestModelToRow:
    def test_fixed_columns(self):
        model = make_model("test")
        row = model_to_row(model, [])
        assert len(row) == len(FIXED_COLUMNS)
        assert row[0] == "id-test"
        assert row[1] == "test"

    def test_custom_columns(self):
        props = {
            "framework": {"metadataType": "MetadataStringValue", "string_value": "pytorch"},
            "accuracy": {"metadataType": "MetadataDoubleValue", "double_value": 0.95},
        }
        model = make_model("test", custom_props=props)
        row = model_to_row(model, ["accuracy", "framework"])
        assert row[len(FIXED_COLUMNS)] == "0.95"
        assert row[len(FIXED_COLUMNS) + 1] == "pytorch"

    def test_missing_custom_key(self):
        model = make_model("test", {"a": {"metadataType": "MetadataStringValue", "string_value": "x"}})
        row = model_to_row(model, ["a", "b"])
        assert row[len(FIXED_COLUMNS)] == "x"
        assert row[len(FIXED_COLUMNS) + 1] == ""


class TestWriteCsv:
    def test_creates_file(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([make_model("m1")], [], path)
            assert os.path.exists(path)

    def test_header_only_for_empty(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([], [], path)
            with open(path, encoding="utf-8-sig") as f:
                lines = f.readlines()
            assert len(lines) == 1

    def test_bom_present(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([], [], path)
            with open(path, "rb") as f:
                assert f.read(3) == b"\xef\xbb\xbf"

    def test_row_count(self):
        models = [make_model(f"m{i}") for i in range(5)]
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv(models, [], path)
            with open(path, encoding="utf-8-sig") as f:
                reader = csv.reader(f)
                rows = list(reader)
            assert len(rows) == 6  # header + 5 models

    def test_no_partial_on_error(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")

            class BrokenModel:
                def get(self, _key, _default=None):
                    raise RuntimeError("disk full")

            with pytest.raises(RuntimeError):
                write_csv([BrokenModel()], [], path)
            assert not os.path.exists(path)

    def test_creates_output_directory(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "subdir", "nested", "out.csv")
            write_csv([make_model("m1")], [], path)
            assert os.path.exists(path)


class TestErrorHandling:
    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_http_error_exits_nonzero(self, mock_open):
        error_body = b'{"error": "unauthorized"}'
        mock_open.side_effect = urllib.error.HTTPError(
            "http://localhost:8080", 401, "Unauthorized", {}, io.BytesIO(error_body)
        )
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            with pytest.raises(SystemExit) as exc_info:
                main(["--url", "http://localhost:8080", "--output", path])
            assert exc_info.value.code != 0
            assert not os.path.exists(path)

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_connection_error_exits_nonzero(self, mock_open):
        mock_open.side_effect = urllib.error.URLError("Connection refused")
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            with pytest.raises(SystemExit) as exc_info:
                main(["--url", "http://localhost:8080", "--output", path])
            assert exc_info.value.code != 0
            assert not os.path.exists(path)


class TestParseArgs:
    def test_required_args(self):
        args = parse_args(["--url", "http://localhost", "--output", "out.csv"])
        assert args.url == "http://localhost"
        assert args.output == "out.csv"

    def test_optional_args(self):
        args = parse_args([
            "--url", "http://localhost", "--output", "out.csv",
            "--source", "s1", "--source", "s2",
            "--limit", "10",
            "--page-size", "50",
            "--header", "Authorization: Bearer tok123",
        ])
        assert args.source == ["s1", "s2"]
        assert args.limit == 10
        assert args.page_size == 50
        assert args.headers == [("Authorization", "Bearer tok123")]

    def test_defaults(self):
        args = parse_args(["--url", "http://localhost", "--output", "out.csv"])
        assert args.source is None
        assert args.limit is None
        assert args.page_size == 100
        assert args.headers is None


class TestMainIntegration:
    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_full_export(self, mock_open):
        props = {"framework": {"metadataType": "MetadataStringValue", "string_value": "pytorch"}}
        models = [make_model("m1", props), make_model("m2")]
        mock_open.side_effect = mock_urlopen([make_api_response(models)])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            main(["--url", "http://localhost:8080", "--output", path])
            with open(path, encoding="utf-8-sig") as f:
                reader = csv.DictReader(f)
                rows = list(reader)
            assert len(rows) == 2
            assert "framework" in reader.fieldnames
            assert rows[0]["framework"] == "pytorch"
            assert rows[1]["framework"] == ""

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_empty_catalog_produces_header_only(self, mock_open):
        mock_open.side_effect = mock_urlopen([make_api_response([])])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            main(["--url", "http://localhost:8080", "--output", path])
            with open(path, encoding="utf-8-sig") as f:
                lines = f.readlines()
            assert len(lines) == 1

    @mock.patch("model_catalog_export.urllib.request.urlopen")
    def test_with_headers(self, mock_open):
        mock_open.side_effect = mock_urlopen([make_api_response([])])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            main([
                "--url", "http://localhost:8080",
                "--output", path,
                "--header", "Authorization: Bearer mytoken",
            ])
            req = mock_open.call_args[0][0]
            assert req.get_header("Authorization") == "Bearer mytoken"
