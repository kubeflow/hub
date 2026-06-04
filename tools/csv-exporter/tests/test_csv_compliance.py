import csv
import os
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from model_catalog_export import FIXED_COLUMNS, write_csv


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
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert rows[0]["description"] == "Has commas, lots of them, really"

    def test_double_quotes_in_name(self):
        model = make_model('model "quoted"')
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert rows[0]["name"] == 'model "quoted"'

    def test_newlines_in_description(self):
        model = make_model("m1", description="Line 1\nLine 2\nLine 3")
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert rows[0]["description"] == "Line 1\nLine 2\nLine 3"

    def test_unicode_characters(self):
        model = make_model("modèle-français", description="日本語テスト données")
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert rows[0]["name"] == "modèle-français"
            assert "日本語テスト" in rows[0]["description"]

    def test_emoji_in_fields(self):
        model = make_model("model-1", description="Great model! 🚀🔥💯")
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert "🚀" in rows[0]["description"]

    def test_combined_hostile_characters(self):
        model = make_model(
            'model "A", v2',
            description='Line 1\nHas "quotes" and, commas\nAnd Unicode: café ☕',
        )
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert rows[0]["name"] == 'model "A", v2'
            assert "café ☕" in rows[0]["description"]

    def test_empty_fields(self):
        model = make_model("m1", description="")
        model["provider"] = None
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            rows, _ = read_csv(path)
            assert rows[0]["description"] == ""
            assert rows[0]["provider"] == ""


class TestBOM:
    def test_utf8_bom_present(self):
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([], [], path)
            with open(path, "rb") as f:
                bom = f.read(3)
            assert bom == b"\xef\xbb\xbf"

    def test_readable_with_utf8_sig(self):
        model = make_model("m1")
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
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
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], sorted(props.keys()), path)
            rows, fieldnames = read_csv(path)
            assert "accuracy" in fieldnames
            assert "framework" in fieldnames
            assert rows[0]["accuracy"] == "0.95"
            assert rows[0]["framework"] == "pytorch"

    def test_custom_keys_sorted_alphabetically(self):
        keys = ["zebra", "alpha", "middle"]
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([], keys, path)
            _, fieldnames = read_csv(path)
            custom_cols = fieldnames[len(FIXED_COLUMNS):]
            assert custom_cols == ["zebra", "alpha", "middle"]  # order as passed

    def test_sparse_custom_properties(self):
        m1 = make_model("m1", custom_props={
            "a": {"metadataType": "MetadataStringValue", "string_value": "x"},
        })
        m2 = make_model("m2", custom_props={
            "b": {"metadataType": "MetadataStringValue", "string_value": "y"},
        })
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([m1, m2], ["a", "b"], path)
            rows, _ = read_csv(path)
            assert rows[0]["a"] == "x"
            assert rows[0]["b"] == ""
            assert rows[1]["a"] == ""
            assert rows[1]["b"] == "y"


class TestExcludedFields:
    def test_readme_excluded(self):
        model = make_model("m1")
        model["readme"] = "x" * 50000
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            _, fieldnames = read_csv(path)
            assert "readme" not in fieldnames

    def test_logo_excluded(self):
        model = make_model("m1")
        model["logo"] = "data:image/png;base64,iVBOR..."
        with tempfile.TemporaryDirectory() as d:
            path = os.path.join(d, "out.csv")
            write_csv([model], [], path)
            _, fieldnames = read_csv(path)
            assert "logo" not in fieldnames
