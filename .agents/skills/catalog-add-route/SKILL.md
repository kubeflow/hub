---
name: catalog-add-route
description: >
  Add an endpoint or query parameter to a catalog plugin's OpenAPI spec.
  Guides the user through the spec edit, then recommends running /sync-catalog
  to propagate changes. Does not run sync automatically — the user may want to
  batch multiple route additions before syncing.
user-invocable: true
---

# Catalog Add Route

Add a new endpoint or query parameter to a catalog plugin's OpenAPI spec.

## Phase 1: Identify Plugin

Ask which plugin. Validate that `api/openapi/src/plugins/<name>.yaml` exists.
Read the current spec to understand existing endpoints.

## Phase 2: What to Add

Ask the user what they want. Common options:

1. **Add query parameter** to the existing list endpoint
2. **Add sub-resource endpoint** (e.g., `GET /agents/{id}/tools`)
3. **Add custom endpoint** (user describes what they need)

## Phase 3: Gather Details + Edit Spec

### Option 1: Add query parameter

Ask:
- Parameter name (snake_case, e.g., `provider`, `tags`, `verified`)
- Type: `string`, `boolean`, `array` (of strings), `integer`
- Description (one line)
- Required? (default: false)

Read the plugin spec's list endpoint. Add the parameter before the `$ref` pagination
parameters, following the existing pattern:

```yaml
- name: <param_name>
  description: <description>
  schema:
    type: <type>          # or for arrays:
    # type: array
    # items:
    #   type: string
  in: query
  required: false
```

For a `filterQuery` parameter (advanced SQL-like filtering), use the shared ref:
```yaml
- $ref: "#/components/parameters/filterQuery"
```
This is defined in `api/openapi/src/lib/common.yaml`.

### Option 2: Add sub-resource endpoint

Ask:
- Sub-resource name (e.g., `tools`, `artifacts`, `metrics`)
- Parent path parameter name (e.g., `id`, `agent_id`)
- Sub-resource fields (name + type pairs, like the entity fields)
- Should there be a list endpoint, get-by-id endpoint, or both?

Add to the spec following the model plugin's artifact pattern
(reference: `api/openapi/src/plugins/model.yaml`, the artifacts endpoints):

**List sub-resource:**
```yaml
  /api/<name>_catalog/v1alpha1/<entity>s/{<parent_id>}/<sub_resource>s:
    get:
      summary: List <sub_resource>s for a <entity>.
      tags:
        - <PascalName>CatalogService
      parameters:
        - name: <parent_id>
          description: The parent entity ID.
          schema:
            type: string
          in: path
          required: true
        - $ref: "#/components/parameters/pageSize"
        - $ref: "#/components/parameters/orderBy"
        - $ref: "#/components/parameters/sortOrder"
        - $ref: "#/components/parameters/nextPageToken"
      responses:
        "200":
          $ref: "#/components/responses/<SubResource>ListResponse"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "500":
          $ref: "#/components/responses/InternalServerError"
      operationId: getAll<Entity><SubResource>s
```

**Get sub-resource:**
```yaml
  /api/<name>_catalog/v1alpha1/<entity>s/{<parent_id>}/<sub_resource>s/{<sub_id>}:
    get:
      summary: Get a <sub_resource> by ID.
      tags:
        - <PascalName>CatalogService
      parameters:
        - name: <parent_id>
          ...
        - name: <sub_id>
          ...
      responses:
        "200":
          $ref: "#/components/responses/<SubResource>Response"
        ...
      operationId: get<Entity><SubResource>
```

Also add the sub-resource schema and response definitions under `components`:

```yaml
components:
  schemas:
    <SubResource>:
      type: object
      properties:
        id:
          type: string
        name:
          type: string
        # ... sub-resource fields

    <SubResource>List:
      type: object
      properties:
        items:
          type: array
          items:
            $ref: "#/components/schemas/<SubResource>"
        nextPageToken:
          type: string
        pageSize:
          type: integer
        size:
          type: integer

  responses:
    <SubResource>ListResponse:
      description: A list of <sub_resource> entities.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/<SubResource>List"
    <SubResource>Response:
      description: A single <sub_resource> entity.
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/<SubResource>"
```

### Option 3: Custom endpoint

Ask the user to describe the endpoint (path, method, purpose). Generate the YAML
following the conventions above. Use the tag `<PascalName>CatalogService` and
derive the operationId from the path + method.

### Naming conventions

Follow these patterns consistently:
- **OperationId**: `find<Entity>s` (list), `get<Entity>` (get), `getAll<Parent><Child>s` (sub-list)
- **Tags**: Always `<PascalName>CatalogService`
- **Response refs**: `<Entity>ListResponse`, `<Entity>Response`
- **Path segments**: snake_case plural (e.g., `/agents`, `/tools`)

## Phase 4: Report

Print what was added to the spec and suggest next steps:

```
Added to api/openapi/src/plugins/<name>.yaml:
  - <description of what was added>

Next: run /sync-catalog to regenerate server stubs and wire the new endpoint.
If you want to add more routes first, run /catalog-add-route again.
```
