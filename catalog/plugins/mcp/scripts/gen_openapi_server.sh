#!/usr/bin/env bash

set -e

echo "Generating MCP plugin server stubs"

OPENAPI_GENERATOR=${OPENAPI_GENERATOR:-openapi-generator-cli}

PROJECT_ROOT=$(realpath "$(dirname "$0")/../../..")
REPO_ROOT=$(realpath "$PROJECT_ROOT/..")
DST="$PROJECT_ROOT/internal/server/openapi"
PLUGIN_DIR="$PROJECT_ROOT/plugins/mcp"

# Assemble standalone spec for MCP plugin
SPEC=$(mktemp -t mcp_plugin_spec_XXXXXX.yaml)
trap 'rm -f "$SPEC"' EXIT

"$REPO_ROOT/scripts/assemble_plugin_spec.sh" mcp "$SPEC"

# Model name mappings to preserve Go acronym casing conventions
MCP_MODEL_MAPPINGS="MCPArtifact=MCPArtifact,MCPConfigMapKey=MCPConfigMapKey,MCPConfigMapRequirement=MCPConfigMapRequirement,MCPEndpoints=MCPEndpoints,MCPEnvVarMetadata=MCPEnvVarMetadata,MCPPrerequisites=MCPPrerequisites,MCPResourceRecommendation=MCPResourceRecommendation,MCPResourceRecommendation_high=MCPResourceRecommendationHigh,MCPResourceRecommendation_minimal=MCPResourceRecommendationMinimal,MCPResourceRecommendation_recommended=MCPResourceRecommendationRecommended,MCPRuntimeMetadata=MCPRuntimeMetadata,MCPRuntimeMetadata_capabilities=MCPRuntimeMetadataCapabilities,MCPRuntimeMetadata_healthEndpoints=MCPRuntimeMetadataHealthEndpoints,MCPSecretKey=MCPSecretKey,MCPSecretRequirement=MCPSecretRequirement,MCPSecurityIndicator=MCPSecurityIndicator,MCPServer=MCPServer,MCPServerList=MCPServerList,MCPServiceAccountRequirement=MCPServiceAccountRequirement,MCPTool=MCPTool,MCPToolParameter=MCPToolParameter,MCPToolWithServer=MCPToolWithServer,MCPToolsList=MCPToolsList"

"$OPENAPI_GENERATOR" generate \
    -i "$SPEC" -g go-server -o "$DST" --package-name openapi \
    --ignore-file-override "$PLUGIN_DIR"/.openapi-generator-ignore \
    --additional-properties=outputAsLibrary=true,enumClassPrefix=true,router=chi,sourceFolder=,onlyInterfaces=true,isGoSubmodule=true,enumClassPrefix=true,useOneOfDiscriminatorLookup=true,featureCORS=true \
    --model-name-mappings="$MCP_MODEL_MAPPINGS" \
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

# Fix package imports
py-re-replace 1 'github\.com/kubeflow/hub/pkg/openapi' 'github.com/kubeflow/hub/catalog/pkg/openapi' \
    "$DST/api_mcp_catalog_service.go" \
    "$DST/api.go"

# Rename api.go to api_mcp.go (avoid collision with model gen)
mv "$DST/api.go" "$DST/api_mcp.go"

# Remove generator boilerplate that shouldn't be tracked
rm -rf "$DST/README.md" "$DST/api/openapi.yaml" "$DST/.openapi-generator-ignore"

# Format generated files
"$REPO_ROOT/bin/goimports" -w "$DST/api_mcp_catalog_service.go" "$DST/api_mcp.go"

echo "MCP plugin server stubs generated"
