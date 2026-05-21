#!/usr/bin/env bash

set -e

echo "Generating the OpenAPI server (per-plugin)"

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}
export OPENAPI_GENERATOR

PROJECT_ROOT=$(realpath "$(dirname "$0")"/..)
DST="$PROJECT_ROOT/${1:-internal/server/openapi}"

# Step 1: Run model plugin generator (also produces shared infrastructure)
"$PROJECT_ROOT/plugins/model/scripts/gen_openapi_server.sh"

# Step 2: Run MCP plugin generator
"$PROJECT_ROOT/plugins/mcp/scripts/gen_openapi_server.sh"

# Step 3: Generate type assertions from combined model_*.go files
echo "Assembling type_assert Go file"
./scripts/gen_type_asserts.sh "$DST"

# Step 4: Final formatting
"$PROJECT_ROOT/../bin/goimports" -w "$DST"

echo "OpenAPI server generation completed"
