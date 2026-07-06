import re
from pathlib import Path


def test_documented_extras_exist_in_package_metadata():
    package_dir = Path(__file__).resolve().parents[1]
    pyproject = (package_dir / "pyproject.toml").read_text()
    extras_section = re.search(r"^\[tool\.poetry\.extras\]\n(?P<body>(?:.+\n)+?)(?:\n\[|\Z)", pyproject, re.MULTILINE)
    assert extras_section, "Missing [tool.poetry.extras] section in pyproject.toml"

    extras = {
        line.split("=", 1)[0].strip()
        for line in extras_section.group("body").splitlines()
        if line.strip() and not line.lstrip().startswith("#")
    }
    readme = (package_dir / "README.md").read_text()

    documented = set(re.findall(r"model-registry\[([a-z0-9_-]+)\]", readme))

    assert documented
    assert documented <= extras, f"Documented extras missing from pyproject.toml: {sorted(documented - extras)}"
