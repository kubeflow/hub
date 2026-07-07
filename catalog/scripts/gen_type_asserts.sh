#!/usr/bin/env bash

set -e

ASSERT_FILE_PATH="$1/type_asserts.go"

PROJECT_ROOT=$(realpath "$(dirname "$0")"/..)

# These files generate with incorrect logic (polymorphic oneOf types):
rm -f "$1/model_metadata_value.go" \
      "$1/model_catalog_artifact.go" \
      "$1/model_agent_artifact.go" \
      "$1/model_filter_option.go" \
      "$1/model_preview_catalog_source_response.go"

python3 "${PROJECT_ROOT}/scripts/gen_type_asserts.py" $1 >"$ASSERT_FILE_PATH"

gofmt -w "$ASSERT_FILE_PATH"
