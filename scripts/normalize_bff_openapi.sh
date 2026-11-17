#!/bin/bash
set -euo pipefail

# Run from clients/ui/bff. Reuse the registry contract for custom JSON unions.
# Script normalizes Artifact and MetadataValue types into the OpenAPI generated spec
YQ="${YQ:-../../../bin/yq}"
"$YQ" -e -i '
  load("../../../api/openapi/model-registry.yaml").components.schemas as $registry |
  .components.schemas += (
    $registry | with_entries(
      select(.key == "Artifact" or .key == "MetadataValue" or
        (.key | test("^Metadata(Bool|Double|Int|Proto|String|Struct)Value$"))) |
      .key = "openapi." + .key
    ) |
    (.. | select(tag == "!!str")) |= sub("^#/components/schemas/", "#/components/schemas/openapi.")
  ) |
  with(.components.schemas."openapi.MetadataValue";
    .type = "object" | .required = ["metadataType"]) |
  with(.components.schemas."openapi.Artifact";
    .type = "object" | .required = ["artifactType"]) |
  . as $doc |
  (.components.schemas."openapi.Artifact".discriminator.mapping | to_entries[]) as $variant ireduce ($doc;
    .components.schemas[($variant.value | sub("^#/components/schemas/", ""))].properties.artifactType = {
      "type": "string", "enum": [$variant.key], "default": $variant.key
    }
  )
' openapi/swagger.yaml

# Swag omits explode=false, but form parameters default to explode=true in OpenAPI.
"$YQ" -e -i '
  (.paths."/api/v1/model_catalog/sources/{source_id}/artifacts/{model_name}".get.parameters[] |
    select(.name == "artifactType")).explode = false
' openapi/swagger.yaml
"$YQ" -o=json -I=4 '.' openapi/swagger.yaml >openapi/swagger.json
