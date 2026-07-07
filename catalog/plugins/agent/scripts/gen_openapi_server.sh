#!/usr/bin/env bash

set -e

echo "Generating agent plugin server stubs"

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}

PROJECT_ROOT=$(realpath "$(dirname "$0")/../../..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")
DST="$PROJECT_ROOT/internal/server/openapi"

# Assemble standalone spec for agent plugin
SPEC=$(mktemp -t agent_plugin_spec_XXXXXX.yaml)
GENDIR=$(mktemp -d -t agent_openapi_gen_XXXXXX)
trap 'rm -rf "$SPEC" "$GENDIR"' EXIT

"$REPO_ROOT/scripts/assemble_plugin_spec.sh" agent "$SPEC"

# Generate into an isolated temp directory so we never touch other plugins' files.
# No .openapi-generator-ignore cross-plugin entries needed.
"$OPENAPI_GENERATOR" generate \
    -i "$SPEC" -g go-server -o "$GENDIR" --package-name openapi \
    --additional-properties=outputAsLibrary=true,enumClassPrefix=true,router=chi,sourceFolder=,onlyInterfaces=true,isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true,featureCORS=true \
    --template-dir "$REPO_ROOT/templates/go-server"

# Python-based regex replace function
py-re-replace() {
  python3 -c "
import fileinput, re, sys
count, pattern, replacement, filepaths = int(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4:]
for filepath in filepaths:
    for line in fileinput.FileInput(filepath, inplace=True, backup=''):
        sys.stdout.write(re.sub(pattern, replacement, line, count=count))
" "$@"
}

# Fix package imports in temp files before copying
py-re-replace 1 'github\.com/kubeflow/hub/pkg/openapi' 'github.com/kubeflow/hub/catalog/pkg/openapi' \
    "$GENDIR/api_agent_catalog_service.go" \
    "$GENDIR/api.go"

# Fix broken array-of-enum syntax in api.go interface (generator emits "model.[]Foo" instead of "[]model.Foo")
py-re-replace 0 'model\.\[\]' '[]model.' "$GENDIR/api.go"

# Copy this plugin's files to the shared output directory
cp "$GENDIR/api_agent_catalog_service.go" "$DST/"
cp "$GENDIR/api.go" "$DST/api_agent.go"

# Copy shared infrastructure (impl.go, error.go, etc.) so they stay in sync
cp "$GENDIR"/impl.go "$GENDIR"/error.go "$GENDIR"/helpers.go "$GENDIR"/routers.go "$GENDIR"/logger.go "$DST/" 2>/dev/null || true

# Copy model type files — needed by gen_type_asserts.sh (untracked, not committed)
cp "$GENDIR"/model_*.go "$DST/" 2>/dev/null || true

# Format
"$REPO_ROOT/bin/goimports" -w "$DST/api_agent_catalog_service.go" "$DST/api_agent.go"

echo "Agent plugin server stubs generated"
