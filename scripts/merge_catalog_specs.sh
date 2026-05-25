#!/bin/bash

set -e

cd "$(pwd)/$(dirname "$0")/.."

if [ -z "$YQ" ]; then
  if [ -e "bin/yq" ]; then
    YQ="$(realpath "bin/yq")"
  else
    echo "Error: YQ is not set and bin/yq does not exist" >&2
    exit 1
  fi
fi

TEMP_FILES=()

cleanup() {
    rm -f "${TEMP_FILES[@]}" 2>/dev/null || true
}
trap cleanup EXIT

register_temp() {
    TEMP_FILES+=("$1")
}

usage() {
    echo "Usage: $0 [--check] <basename.yaml>"
    echo "  --check: Check for differences in the generated merged catalog specification."
    echo ""
    echo "This script merges the core catalog API source with shared libraries and"
    echo "all plugin API specs to produce a unified OpenAPI specification."
    echo ""
    echo "Example: $0 catalog.yaml"
    exit 0
}

CHECK=false
BASENAME=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --check)
            CHECK=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            if [[ "${1#-}" != "$1" ]]; then
                echo "Unknown option: $1"
                usage
            fi
            if [[ "$BASENAME" != "" ]]; then
                usage
            fi

            BASENAME=$1
            shift
            ;;
    esac
done

if [[ "$BASENAME" == "" ]]; then
    usage
fi

BASENAME=$(basename "$BASENAME")
SOURCE_FILE="api/openapi/src/${BASENAME%.yaml}.yaml"
if [[ ! -f "$SOURCE_FILE" ]]; then
    echo "No source file at $SOURCE_FILE"
    exit 1
fi

OUT_FILE="api/openapi/$BASENAME"
if [[ "$CHECK" == "true" ]]; then
    OUT_FILE="$(mktemp -t modelregistry_catalog_spec_tempXXXXXX).yaml"
    register_temp "$OUT_FILE"
fi

# Step 1: Start with the core catalog source
cp "$SOURCE_FILE" "$OUT_FILE"

# Step 2: Discover and merge plugin specs (before shared libraries,
# so catalog-specific parameters appear before common.yaml parameters
# in the final output — preserving the original key order)
PLUGIN_FILES=()
while IFS= read -r f; do
    PLUGIN_FILES+=("$f")
done < <(find api/openapi/src/plugins -maxdepth 1 -name '*.yaml' -type f 2>/dev/null | sort || true)

for plugin_file in "${PLUGIN_FILES[@]}"; do
    temp_merged="$(mktemp -t merged_tempXXXXXX).yaml"
    register_temp "$temp_merged"
    $YQ eval-all '. as $item ireduce ({}; . * $item)' "$OUT_FILE" "$plugin_file" >"$temp_merged"
    mv "$temp_merged" "$OUT_FILE"
done

# Step 3: Merge shared libraries last (common.yaml provides base types
# and overrides any conflicting definitions)
temp_with_libs="$(mktemp -t merged_libs_XXXXXX).yaml"
register_temp "$temp_with_libs"
$YQ eval-all '. as $item ireduce ({}; . * $item)' "$OUT_FILE" api/openapi/src/lib/*.yaml >"$temp_with_libs"
mv "$temp_with_libs" "$OUT_FILE"

# Step 3: Re-order keys and sort for deterministic output
$YQ eval -i '
    {
        "openapi": .openapi,
        "info": .info,
        "servers": .servers,
        "paths": .paths,
        "components": .components,
        "security": .security,
        "tags": .tags
    } |
        sort_keys(.paths) |
        sort_keys(.components.schemas) |
        sort_keys(.components.responses)
' "$OUT_FILE"

if [[ "$CHECK" == "true" ]]; then
    exec diff -u "api/openapi/$BASENAME" "$OUT_FILE"
fi
