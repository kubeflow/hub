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
PLUGIN_DIRS=()
while IFS= read -r dir; do
    PLUGIN_DIRS+=("$dir")
done < <(find api/openapi/src/plugins/* -maxdepth 0 -type d 2>/dev/null | sort || true)

for plugin_dir in "${PLUGIN_DIRS[@]}"; do
    plugin_name=$(basename "$plugin_dir")

    if [[ -z "$plugin_name" ]]; then
        continue
    fi

    # Collect plugin spec files (openapi.yaml + components.yaml)
    plugin_files=()
    for f in "$plugin_dir/openapi.yaml" "$plugin_dir/components.yaml"; do
        if [[ -f "$f" ]]; then
            plugin_files+=("$f")
        fi
    done

    if [[ ${#plugin_files[@]} -eq 0 ]]; then
        continue
    fi

    # Merge plugin files into a single temporary spec
    temp_plugin="$(mktemp -t "plugin_${plugin_name}_XXXXXX").yaml"
    register_temp "$temp_plugin"

    if [[ ${#plugin_files[@]} -eq 1 ]]; then
        cp "${plugin_files[0]}" "$temp_plugin"
    else
        $YQ eval-all '. as $item ireduce ({}; . * $item)' "${plugin_files[@]}" >"$temp_plugin"
    fi

    # Deep-merge plugin spec into the main output
    temp_merged="$(mktemp -t merged_tempXXXXXX).yaml"
    register_temp "$temp_merged"
    $YQ eval-all '. as $item ireduce ({}; . * $item)' "$OUT_FILE" "$temp_plugin" >"$temp_merged"
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
