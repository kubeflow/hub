#!/usr/bin/env bash

set -e

PROJECT_ROOT=$(realpath "$(dirname "$0")/..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")

export OPENAPI_GENERATOR="${OPENAPI_GENERATOR:-$REPO_ROOT/bin/openapi-generator-cli}"

for script in "$PROJECT_ROOT"/plugins/*/scripts/gen_openapi_server.sh; do
    [ -x "$script" ] || continue
    "$script"
done

"$PROJECT_ROOT/scripts/gen_type_asserts.sh" "$PROJECT_ROOT/internal/server/openapi"
"$REPO_ROOT/bin/goimports" -w "$PROJECT_ROOT/internal/server/openapi"
