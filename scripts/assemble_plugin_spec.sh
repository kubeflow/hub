#!/bin/bash

set -e

cd "$(dirname "$(readlink -f "$0")")/.."

if [ -z "$YQ" ]; then
  if [ -e "bin/yq" ]; then
    YQ="$(realpath "bin/yq")"
  else
    echo "Error: YQ is not set and bin/yq does not exist" >&2
    exit 1
  fi
fi

usage() {
    echo "Usage: $0 <plugin_name> <output_path>"
    echo ""
    echo "Assembles a standalone OpenAPI spec for a plugin by merging:"
    echo "  - Core catalog spec (api/openapi/src/catalog.yaml)"
    echo "  - Plugin spec (api/openapi/src/plugins/<name>.yaml)"
    echo "  - Shared libraries (api/openapi/src/lib/*.yaml)"
    echo ""
    echo "Example: $0 model /tmp/model_spec.yaml"
    exit 0
}

PLUGIN_NAME="${1:-}"
OUT_PATH="${2:-}"

if [[ -z "$PLUGIN_NAME" || -z "$OUT_PATH" ]]; then
    usage
fi

PLUGIN_FILE="api/openapi/src/plugins/$PLUGIN_NAME.yaml"
if [[ ! -f "$PLUGIN_FILE" ]]; then
    echo "Error: Plugin spec not found at $PLUGIN_FILE" >&2
    exit 1
fi

# Merge: core catalog + plugin spec + shared libs
# Core comes first (provides envelope, shared paths, shared schemas),
# then plugin spec (adds plugin-specific paths/schemas),
# then shared libs last (common.yaml provides base types, overrides on conflicts)
$YQ eval-all '. as $item ireduce ({}; . * $item)' \
    api/openapi/src/catalog.yaml \
    "$PLUGIN_FILE" \
    api/openapi/src/lib/*.yaml \
    >"$OUT_PATH"

# For non-model plugins, strip ModelCatalogService paths to avoid
# generating a model controller alongside the plugin controller
if [[ "$PLUGIN_NAME" != "model" ]]; then
    $YQ eval -i 'del(.paths[] | select(key | test("^/api/model_catalog/")))' "$OUT_PATH" 2>/dev/null || true
    # yq may not support that syntax; use explicit path deletion
    MODEL_PATHS=$($YQ eval '.paths | keys | .[] | select(test("^/api/model_catalog/"))' "$OUT_PATH" 2>/dev/null || echo "")
    if [[ -n "$MODEL_PATHS" ]]; then
        local_expr=""
        while IFS= read -r path; do
            [[ -z "$path" ]] && continue
            if [[ -n "$local_expr" ]]; then
                local_expr="$local_expr | "
            fi
            local_expr="${local_expr}del(.paths[\"$path\"])"
        done <<< "$MODEL_PATHS"
        if [[ -n "$local_expr" ]]; then
            $YQ eval -i "$local_expr" "$OUT_PATH"
        fi
    fi
fi

# Re-order keys for consistency
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
' "$OUT_PATH"
