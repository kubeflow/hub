#!/usr/bin/env bash

set -e

echo "Generating model plugin server stubs"

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}

PROJECT_ROOT=$(realpath "$(dirname "$0")/../../..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")
DST="$PROJECT_ROOT/internal/server/openapi"
PLUGIN_DIR="$PROJECT_ROOT/plugins/model"

# Assemble standalone spec for model plugin
SPEC=$(mktemp -t model_plugin_spec_XXXXXX.yaml)
trap 'rm -f "$SPEC"' EXIT

"$REPO_ROOT/scripts/assemble_plugin_spec.sh" model "$SPEC"

"$OPENAPI_GENERATOR" generate \
    -i "$SPEC" -g go-server -o "$DST" --package-name openapi \
    --ignore-file-override "$PLUGIN_DIR"/.openapi-generator-ignore \
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

# Fix array type references in api.go
py-re-replace 0 'model\.\[\]ArtifactTypeQueryParam' '[]model.ArtifactTypeQueryParam' "$DST/api.go"
py-re-replace 0 'model\.\[\]ArtifactType2QueryParam' '[]model.ArtifactTypeQueryParam' "$DST/api.go"

# Fix package imports
py-re-replace 1 'github\.com/kubeflow/hub/pkg/openapi' 'github.com/kubeflow/hub/catalog/pkg/openapi' \
    "$DST/api_model_catalog_service.go" \
    "$DST/api.go"

# Fix wildcard path placeholder
py-re-replace 1 '\{model_name\+\}|model_name\+' '*' "$DST/api_model_catalog_service.go"

# Rename api.go to api_model.go (avoid collision with MCP gen)
mv "$DST/api.go" "$DST/api_model.go"

# Remove generator boilerplate that shouldn't be tracked
rm -rf "$DST/README.md" "$DST/api/openapi.yaml" "$DST/.openapi-generator-ignore"

# Format generated files
"$REPO_ROOT/bin/goimports" -w "$DST/api_model_catalog_service.go" "$DST/api_model.go"

# Apply route delegation patch
echo "Applying model plugin patches"
(
    cd "$REPO_ROOT"
    ./bin/goimports -w "$DST/api_model_catalog_service.go"
    git apply patches/api_model_catalog_service.patch
)

echo "Model plugin server stubs generated"
