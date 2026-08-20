#!/usr/bin/env bash

set -e

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}

PROJECT_ROOT=$(realpath "$(dirname "$0")/../../..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")

VERSIONS=("v1")
if [[ -n "$1" ]]; then
    VERSIONS=("$1")
fi

py-re-replace() {
  python3 -c "
import fileinput, re, sys
count, pattern, replacement, filepaths = int(sys.argv[1]), sys.argv[2], sys.argv[3], sys.argv[4:]
for filepath in filepaths:
    for line in fileinput.FileInput(filepath, inplace=True, backup=''):
        sys.stdout.write(re.sub(pattern, replacement, line, count=count))
" "$@"
}

TMPFILES=()
trap 'rm -rf "${TMPFILES[@]}"' EXIT

for VER in "${VERSIONS[@]}"; do
    echo "Generating skill plugin server stubs ($VER)"
    DST="$PROJECT_ROOT/internal/server/openapi/$VER"
    mkdir -p "$DST"

    SPEC=$(mktemp -t skill_plugin_spec_XXXXXX.yaml)
    GENDIR=$(mktemp -d -t skill_openapi_gen_XXXXXX)
    TMPFILES+=("$SPEC" "$GENDIR")

    "$REPO_ROOT/scripts/assemble_plugin_spec.sh" skill "$SPEC" "$VER"

    "$OPENAPI_GENERATOR" generate \
        -i "$SPEC" -g go-server -o "$GENDIR" --package-name "$VER" \
        --additional-properties=outputAsLibrary=true,enumClassPrefix=true,router=chi,sourceFolder=,onlyInterfaces=true,isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true \
        --template-dir "$REPO_ROOT/templates/go-server"

    # Fix package imports in temp files
    py-re-replace 1 'github\.com/kubeflow/hub/pkg/openapi' 'github.com/kubeflow/hub/catalog/pkg/openapi' \
        "$GENDIR/api_skill_catalog_service.go" \
        "$GENDIR/api.go"

    # Fix broken array-of-enum syntax in api.go interface
    py-re-replace 0 'model\.\[\]' '[]model.' "$GENDIR/api.go"

    # Copy this plugin's files to the shared output directory
    cp "$GENDIR/api_skill_catalog_service.go" "$DST/"
    cp "$GENDIR/api.go" "$DST/api_skill.go"

    # Copy shared infrastructure
    cp "$GENDIR"/impl.go "$GENDIR"/error.go "$GENDIR"/helpers.go "$GENDIR"/routers.go "$GENDIR"/logger.go "$DST/" 2>/dev/null || true

    # Copy model type files — needed by gen_type_asserts.sh
    cp "$GENDIR"/model_*.go "$DST/" 2>/dev/null || true

    "$REPO_ROOT/bin/goimports" -w "$DST/api_skill_catalog_service.go" "$DST/api_skill.go"

    echo "Skill plugin server stubs generated ($VER)"
done
