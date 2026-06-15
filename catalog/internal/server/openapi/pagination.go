package openapi

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
)

// parsePaginationParams validates and parses pageSize and nextPageToken for DB-backed endpoints.
// It returns the page size as int32, or an error if either parameter is invalid.
func parsePaginationParams(pageSize string, nextPageToken string) (int32, error) {
	pageSizeInt := int32(10)
	if pageSize != "" {
		parsed, err := strconv.ParseInt(pageSize, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid pageSize: %w", err)
		}
		if parsed < 1 {
			return 0, fmt.Errorf("pageSize must be at least 1, got %d", parsed)
		}
		pageSizeInt = int32(parsed)
	}
	if nextPageToken != "" {
		if _, err := scopes.DecodeCursor(nextPageToken); err != nil {
			return 0, fmt.Errorf("invalid nextPageToken: %w", err)
		}
	}
	return pageSizeInt, nil
}

// parsePageSize validates and parses only the pageSize parameter, without validating nextPageToken.
// Used by endpoints that handle their own pagination token format (e.g., in-memory offset-based pagination).
func parsePageSize(pageSize string) (int32, error) {
	pageSizeInt := int32(10)
	if pageSize != "" {
		parsed, err := strconv.ParseInt(pageSize, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid pageSize: %w", err)
		}
		if parsed < 1 {
			return 0, fmt.Errorf("pageSize must be at least 1, got %d", parsed)
		}
		pageSizeInt = int32(parsed)
	}
	return pageSizeInt, nil
}

type paginator[T model.Sortable] struct {
	PageSize  int32
	OrderBy   model.OrderByField
	SortOrder model.SortOrder
	cursor    *stringCursor
}

func newPaginator[T model.Sortable](pageSize string, orderBy model.OrderByField, sortOrder model.SortOrder, nextPageToken string) (*paginator[T], error) {
	if orderBy != "" && !orderBy.IsValid() {
		return nil, fmt.Errorf("unsupported order by field: %s", orderBy)
	}
	if sortOrder != "" && !sortOrder.IsValid() {
		return nil, fmt.Errorf("unsupported sort order field: %s", sortOrder)
	}

	p := &paginator[T]{
		PageSize:  10, // Default page size
		OrderBy:   orderBy,
		SortOrder: sortOrder,
	}

	if pageSize != "" {
		pageSize64, err := strconv.ParseInt(pageSize, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error converting page size to int32: %w", err)
		}
		if pageSize64 < 1 {
			return nil, fmt.Errorf("pageSize must be at least 1, got %d", pageSize64)
		}
		p.PageSize = int32(pageSize64)
	}

	if nextPageToken != "" {
		cursor, err := decodeStringCursor(nextPageToken)
		if err != nil {
			return nil, fmt.Errorf("invalid nextPageToken: %w", err)
		}
		p.cursor = cursor
	}

	return p, nil
}

func (p *paginator[T]) Token() string {
	if p == nil || p.cursor == nil {
		return ""
	}
	return p.cursor.String()
}

func (p *paginator[T]) Paginate(items []T) ([]T, *paginator[T]) {
	startIndex := 0
	if p.cursor != nil {
		for i, item := range items {
			itemValue := item.SortValue(p.OrderBy)
			id := item.SortValue(model.ORDERBYFIELD_ID)
			if id != "" && id == p.cursor.ID && itemValue == p.cursor.Value {
				startIndex = i + 1
				break
			}
		}
	}

	if startIndex >= len(items) {
		return []T{}, nil
	}

	var pagedItems []T
	var next *paginator[T]

	endIndex := min(startIndex+int(p.PageSize), len(items))
	pagedItems = items[startIndex:endIndex]

	if endIndex < len(items) {
		lastItem := pagedItems[len(pagedItems)-1]
		lastItemID := lastItem.SortValue(model.ORDERBYFIELD_ID)
		if lastItemID != "" {
			next = &paginator[T]{
				PageSize:  p.PageSize,
				OrderBy:   p.OrderBy,
				SortOrder: p.SortOrder,
				cursor: &stringCursor{
					Value: lastItem.SortValue(p.OrderBy),
					ID:    lastItemID,
				},
			}
		}
	}

	return pagedItems, next
}

type stringCursor struct {
	Value string
	ID    string
}

func (c *stringCursor) String() string {
	return base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", c.Value, c.ID))
}

func decodeStringCursor(encoded string) (*stringCursor, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding: %w", err)
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token format")
	}
	return &stringCursor{
		Value: parts[0],
		ID:    parts[1],
	}, nil
}
