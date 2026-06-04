.PHONY: install test lint clean

install:
	pip install pytest ruff mypy

test:
	python -m pytest -s -x -rA tests/

lint:
	ruff check model_catalog_export.py tests/
	mypy model_catalog_export.py

clean:
	rm -rf __pycache__ tests/__pycache__ .mypy_cache .pytest_cache
