#!/usr/bin/env bash

set -e

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}

PROJECT_ROOT=$(realpath "$(dirname "$0")/../../..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")

VERSIONS=("v1alpha1" "v1")
if [[ -n "$1" ]]; then
    VERSIONS=("$1")
fi

MCP_MODEL_MAPPINGS="MCPArtifact=MCPArtifact,MCPConfigMapKey=MCPConfigMapKey,MCPConfigMapRequirement=MCPConfigMapRequirement,MCPEndpoints=MCPEndpoints,MCPEnvVarMetadata=MCPEnvVarMetadata,MCPPrerequisites=MCPPrerequisites,MCPResourceRecommendation=MCPResourceRecommendation,MCPResourceRecommendation_high=MCPResourceRecommendationHigh,MCPResourceRecommendation_minimal=MCPResourceRecommendationMinimal,MCPResourceRecommendation_recommended=MCPResourceRecommendationRecommended,MCPRuntimeMetadata=MCPRuntimeMetadata,MCPRuntimeMetadata_capabilities=MCPRuntimeMetadataCapabilities,MCPRuntimeMetadata_healthEndpoints=MCPRuntimeMetadataHealthEndpoints,MCPSecretKey=MCPSecretKey,MCPSecretRequirement=MCPSecretRequirement,MCPSecurityIndicator=MCPSecurityIndicator,MCPServer=MCPServer,MCPServerList=MCPServerList,MCPServiceAccountRequirement=MCPServiceAccountRequirement,MCPStorageMount=MCPStorageMount,MCPStorageSource=MCPStorageSource,MCPTool=MCPTool,MCPToolParameter=MCPToolParameter,MCPToolWithServer=MCPToolWithServer,MCPToolsList=MCPToolsList"

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
    echo "Generating MCP plugin server stubs ($VER)"
    DST="$PROJECT_ROOT/internal/server/openapi/$VER"
    mkdir -p "$DST"

    SPEC=$(mktemp -t mcp_plugin_spec_XXXXXX.yaml)
    GENDIR=$(mktemp -d -t mcp_openapi_gen_XXXXXX)
    TMPFILES+=("$SPEC" "$GENDIR")

    "$REPO_ROOT/scripts/assemble_plugin_spec.sh" mcp "$SPEC" "$VER"

    "$OPENAPI_GENERATOR" generate \
        -i "$SPEC" -g go-server -o "$GENDIR" --package-name "$VER" \
        --additional-properties=outputAsLibrary=true,enumClassPrefix=true,router=chi,sourceFolder=,onlyInterfaces=true,isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true \
        --model-name-mappings="$MCP_MODEL_MAPPINGS" \
        --template-dir "$REPO_ROOT/templates/go-server"

    # Fix package imports in temp files
    py-re-replace 1 'github\.com/kubeflow/hub/pkg/openapi' 'github.com/kubeflow/hub/catalog/pkg/openapi' \
        "$GENDIR/api_mcp_catalog_service.go" \
        "$GENDIR/api.go"

    # Copy this plugin's files to the shared output directory
    cp "$GENDIR/api_mcp_catalog_service.go" "$DST/"
    cp "$GENDIR/api.go" "$DST/api_mcp.go"

    # Copy shared infrastructure
    cp "$GENDIR"/impl.go "$GENDIR"/error.go "$GENDIR"/helpers.go "$GENDIR"/routers.go "$GENDIR"/logger.go "$DST/" 2>/dev/null || true

    # Copy model type files — needed by gen_type_asserts.sh
    cp "$GENDIR"/model_*.go "$DST/" 2>/dev/null || true

    "$REPO_ROOT/bin/goimports" -w "$DST/api_mcp_catalog_service.go" "$DST/api_mcp.go"

    echo "MCP plugin server stubs generated ($VER)"
done
