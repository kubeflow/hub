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
    echo "  - Plugin paths and schemas (catalog/plugins/<name>/api/openapi/)"
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

PLUGIN_DIR="catalog/plugins/$PLUGIN_NAME/api/openapi"
if [[ ! -d "$PLUGIN_DIR" ]]; then
    echo "Error: Plugin directory not found at $PLUGIN_DIR" >&2
    exit 1
fi

# Collect plugin spec files
PLUGIN_FILES=()
for f in "$PLUGIN_DIR/openapi.yaml" "$PLUGIN_DIR/components.yaml"; do
    if [[ -f "$f" ]]; then
        PLUGIN_FILES+=("$f")
    fi
done

if [[ ${#PLUGIN_FILES[@]} -eq 0 ]]; then
    echo "Error: No spec files found in $PLUGIN_DIR" >&2
    exit 1
fi

# Merge: core catalog + plugin specs + shared libs
# Core comes first (provides envelope, shared paths, shared schemas),
# then plugin specs (add plugin-specific paths/schemas),
# then shared libs last (common.yaml provides base types, overrides on conflicts)
$YQ eval-all '. as $item ireduce ({}; . * $item)' \
    api/openapi/src/catalog.yaml \
    "${PLUGIN_FILES[@]}" \
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
