import csv
import os
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from model_catalog_export import COLUMN_ORDER, EXCLUDED_FIELDS, discover_columns, write_csv


def make_model(name, description="", custom_props=None, **overrides):
    model = {
        "id": f"id-{name}",
        "name": name,
        "description": description,
        "provider": "TestProvider",
        "maturity": "Generally Available",
        "license": "apache-2.0",
        "licenseLink": "",
        "libraryName": "transformers",
        "source_id": "test-source",
        "externalId": "",
        "createTimeSinceEpoch": "1609459200000",
        "lastUpdateTimeSinceEpoch": "1609459200000",
        "language": ["en"],
        "tasks": ["text-generation"],
        "validatedTasks": [],
    }
    if custom_props:
        model["customProperties"] = custom_props
    model.update(overrides)
    return model


def read_csv(path):
    with open(path, encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        return list(reader), reader.fieldnames


class TestRFC4180Compliance:
    def test_commas_in_description(self):
        model = make_model("m1", description="Has commas, lots of them, really")
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert rows[0]["description"] == "Has commas, lots of them, really"

    def test_double_quotes_in_name(self):
        model = make_model('model "quoted"')
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert rows[0]["name"] == 'model "quoted"'

    def test_newlines_in_description(self):
        model = make_model("m1", description="Line 1\nLine 2\nLine 3")
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert rows[0]["description"] == "Line 1\nLine 2\nLine 3"

    def test_unicode_characters(self):
        model = make_model("modèle-français", description="日本語テスト données")
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert rows[0]["name"] == "modèle-français"
            assert "日本語テスト" in rows[0]["description"]

    def test_emoji_in_fields(self):
        model = make_model("model-1", description="Great model! \U0001f680\U0001f525\U0001f4af")
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert "\U0001f680" in rows[0]["description"]

    def test_combined_hostile_characters(self):
        model = make_model(
            'model "A", v2',
            description='Line 1\nHas "quotes" and, commas\nAnd Unicode: café ☕',
        )
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert rows[0]["name"] == 'model "A", v2'
            assert "café ☕" in rows[0]["description"]

    def test_empty_fields(self):
        model = make_model("m1", description="")
        model["provider"] = None
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            rows, _ = read_csv(path)
            assert rows[0]["description"] == ""
            assert rows[0]["provider"] == ""


class TestBOM:
    def test_utf8_bom_present(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([], COLUMN_ORDER, [], path)
            with open(path, "rb") as f:
                bom = f.read(3)
            assert bom == b"\xef\xbb\xbf"

    def test_readable_with_utf8_sig(self):
        model = make_model("m1")
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, [], path)
            with open(path, encoding="utf-8-sig") as f:
                reader = csv.DictReader(f)
                rows = list(reader)
            assert rows[0]["name"] == "m1"
            assert reader.fieldnames[0] == "id"


class TestCustomPropertyColumns:
    def test_custom_keys_appear_as_columns(self):
        props = {
            "framework": {"metadataType": "MetadataStringValue", "string_value": "pytorch"},
            "accuracy": {"metadataType": "MetadataDoubleValue", "double_value": 0.95},
        }
        model = make_model("m1", custom_props=props)
        columns = discover_columns([model])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], columns, sorted(props.keys()), path)
            rows, fieldnames = read_csv(path)
            assert "accuracy" in fieldnames
            assert "framework" in fieldnames
            assert rows[0]["accuracy"] == "0.95"
            assert rows[0]["framework"] == "pytorch"

    def test_custom_keys_sorted_alphabetically(self):
        keys = ["zebra", "alpha", "middle"]
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([], COLUMN_ORDER, keys, path)
            _, fieldnames = read_csv(path)
            custom_cols = fieldnames[len(COLUMN_ORDER):]
            assert custom_cols == ["zebra", "alpha", "middle"]  # order as passed

    def test_sparse_custom_properties(self):
        m1 = make_model("m1", custom_props={
            "a": {"metadataType": "MetadataStringValue", "string_value": "x"},
        })
        m2 = make_model("m2", custom_props={
            "b": {"metadataType": "MetadataStringValue", "string_value": "y"},
        })
        columns = discover_columns([m1, m2])
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([m1, m2], columns, ["a", "b"], path)
            rows, _ = read_csv(path)
            assert rows[0]["a"] == "x"
            assert rows[0]["b"] == ""
            assert rows[1]["a"] == ""
            assert rows[1]["b"] == "y"


class TestDiscoverColumns:
    def test_empty_models_returns_column_order(self):
        assert discover_columns([]) == list(COLUMN_ORDER)

    def test_excludes_readme(self):
        model = make_model("m1")
        model["readme"] = "x" * 50000
        columns = discover_columns([model])
        assert "readme" not in columns

    def test_excludes_logo(self):
        model = make_model("m1")
        model["logo"] = "data:image/png;base64,iVBOR..."
        columns = discover_columns([model])
        assert "logo" not in columns

    def test_excludes_serving_config(self):
        model = make_model("m1")
        model["servingConfig"] = {"toolCalling": {}}
        columns = discover_columns([model])
        assert "servingConfig" not in columns

    def test_excludes_custom_properties(self):
        model = make_model("m1", custom_props={"k": {"metadataType": "MetadataStringValue", "string_value": "v"}})
        columns = discover_columns([model])
        assert "customProperties" not in columns

    def test_preserves_column_order(self):
        model = make_model("m1")
        columns = discover_columns([model])
        order_indices = [columns.index(c) for c in COLUMN_ORDER if c in columns]
        assert order_indices == sorted(order_indices)

    def test_new_api_fields_appended_alphabetically(self):
        model = make_model("m1")
        model["newFieldB"] = "val"
        model["newFieldA"] = "val"
        columns = discover_columns([model])
        known_end = len([c for c in COLUMN_ORDER if c in columns])
        new_cols = columns[known_end:]
        assert new_cols == ["newFieldA", "newFieldB"]

    def test_all_excluded_fields_filtered(self):
        model = make_model("m1")
        for field in EXCLUDED_FIELDS:
            model[field] = "value"
        columns = discover_columns([model])
        for field in EXCLUDED_FIELDS:
            assert field not in columns
