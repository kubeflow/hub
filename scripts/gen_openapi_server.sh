#!/usr/bin/env bash

set -e

SPEC_FILE="$1"
OUTPUT_DIR="$2"
PACKAGE_NAME="$3"

if [ -z "$SPEC_FILE" ] || [ -z "$OUTPUT_DIR" ] || [ -z "$PACKAGE_NAME" ]; then
    echo "Usage: $0 <spec-file> <output-dir> <package-name>"
    echo "  paths are relative to the project root"
    exit 1
fi

echo "Generating the OpenAPI server ($PACKAGE_NAME)"

PROJECT_ROOT=$(realpath "$(dirname "$0")"/..)
OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-"$PROJECT_ROOT"/bin/openapi-generator-cli}

$OPENAPI_GENERATOR generate \
    -i "$PROJECT_ROOT"/"$SPEC_FILE" -g go-server -o "$PROJECT_ROOT"/"$OUTPUT_DIR" --package-name "$PACKAGE_NAME" \
    --ignore-file-override "$PROJECT_ROOT"/.openapi-generator-ignore --additional-properties=outputAsLibrary=true,enumClassPrefix=true,router=chi,sourceFolder=,onlyInterfaces=true,isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true \
    --template-dir "$PROJECT_ROOT"/templates/go-server

echo "Assembling type_assert Go file"
"$PROJECT_ROOT"/scripts/gen_type_asserts.sh "$OUTPUT_DIR"

gofmt -w "$PROJECT_ROOT"/"$OUTPUT_DIR"

echo "OpenAPI server generation completed"
