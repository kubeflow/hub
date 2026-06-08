# Panic Recipes: Filtering & Ordering (Panics 4–5)

These panics handle list query filtering and custom ordering/pagination.
Use the **Type Reference Table** in SKILL.md to select the correct schema types.

## Panic 4: `applyListFilters`

No-op — return query unchanged. Filtering can be added later.

```go
func applyXxxListFilters(query *gorm.DB, _ *models.XxxListOptions) *gorm.DB {
    return query
}
```

## Panic 5: `applyCustomOrdering`

This requires **three pieces** — all methods on the repository impl, not standalone functions.
`GenericRepository.List` treats a non-nil `ApplyCustomOrdering` callback as a *replacement*
for its built-in `ApplyStandardPagination`. If the callback doesn't apply pagination,
no LIMIT is applied and all rows are returned regardless of `pageSize`.

### 5a. OrderByColumns map

Defines which `orderBy` values the API accepts:

```go
var XxxOrderByColumns = map[string]string{
    "ID":               "id",
    "CREATE_TIME":      "create_time_since_epoch",
    "LAST_UPDATE_TIME": "last_update_time_since_epoch",
    "NAME":             "name",
    "id":               "id",
}
```

### 5b. `applyCustomOrdering`

Handle NAME ordering explicitly (it uses a special helper for cursor-based pagination
on the name column), fall back to standard for everything else:

```go
func (r *XxxRepositoryImpl) applyXxxCustomOrdering(query *gorm.DB, listOptions *models.XxxListOptions) *gorm.DB {
    db := r.GetConfig().DB
    TABLE := utils.GetTableName(db, &SCHEMA_TYPE{})
    orderBy := listOptions.GetOrderBy()

    if orderBy == "NAME" {
        return pagination.ApplyNameOrdering(query, TABLE, listOptions.GetSortOrder(), listOptions.GetNextPageToken(), listOptions.GetPageSize(), false)
    }

    return r.ApplyStandardPagination(query, listOptions, []models.Xxx{})
}
```

Replace TABLE lookup with the correct schema type from the reference table
(e.g., `&schema.Context{}` for context entities).

Add these imports if not already present:
```go
"github.com/kubeflow/hub/catalog/internal/db/pagination"
"github.com/kubeflow/hub/internal/platform/db/utils"
```

### 5c. `ApplyStandardPagination` override

Passes the OrderByColumns map to the GORM pagination scope:

```go
func (r *XxxRepositoryImpl) ApplyStandardPagination(query *gorm.DB, listOptions *models.XxxListOptions, entities any) *gorm.DB {
    pageSize := listOptions.GetPageSize()
    orderBy := listOptions.GetOrderBy()
    sortOrder := listOptions.GetSortOrder()
    nextPageToken := listOptions.GetNextPageToken()

    pag := &dbmodels.Pagination{
        PageSize:      &pageSize,
        OrderBy:       &orderBy,
        SortOrder:     &sortOrder,
        NextPageToken: &nextPageToken,
    }

    return query.Scopes(scopes.PaginateWithOptions(entities, pag, r.GetConfig().DB, "TABLE", XxxOrderByColumns))
}
```

Replace `"TABLE"` with the GORM table name: `"Context"` for context,
`"Artifact"` for artifact, `"Execution"` for execution.

Add this import if not already present:
```go
"github.com/kubeflow/hub/internal/platform/db/scopes"
```

### Update the constructor

Wire the method receiver in the constructor:
```go
ApplyCustomOrdering: r.applyXxxCustomOrdering,
```
