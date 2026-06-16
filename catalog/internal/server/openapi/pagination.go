package openapi

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"

	model "github.com/kubeflow/hub/catalog/pkg/openapi"
	"github.com/kubeflow/hub/internal/platform/db/scopes"
)

const defaultPageSize int32 = 10

// parsePageSize validates and parses the pageSize parameter, returning the
// page size as int32 (defaulting to defaultPageSize when empty). This is the single source
// of truth for pageSize validation, used by both parsePaginationParams and the
// recommended path in FindModels.
func parsePageSize(pageSize string) (int32, error) {
	if pageSize == "" {
		return defaultPageSize, nil
	}
	parsed, err := strconv.ParseInt(pageSize, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid pageSize: %w", err)
	}
	if parsed < 1 {
		return 0, fmt.Errorf("pageSize must be at least 1, got %d", parsed)
	}
	return int32(parsed), nil
}

// parsePaginationParams validates and parses pageSize and nextPageToken for DB-backed endpoints.
// It returns the page size as int32, or an error if either parameter is invalid.
func parsePaginationParams(pageSize string, nextPageToken string) (int32, error) {
	pageSizeInt, err := parsePageSize(pageSize)
	if err != nil {
		return 0, err
	}
	if nextPageToken != "" {
		if _, err := scopes.DecodeCursor(nextPageToken); err != nil {
			return 0, fmt.Errorf("invalid nextPageToken: %w", err)
		}
	}
	return pageSizeInt, nil
}

// validateRecommendedNextPageToken validates that a nextPageToken for the recommended
// path is a non-negative integer within the int32 range (0..2147483647).
// The recommended path uses numeric offset tokens, unlike the non-recommended path
// which uses base64-encoded DB cursors.
// Using bitSize 31 in ParseUint ensures values >= 2^31 (i.e. > MaxInt32) are
// rejected during parsing, eliminating the need for a separate range check.
func validateRecommendedNextPageToken(nextPageToken string) error {
	if nextPageToken == "" {
		return nil
	}
	_, err := strconv.ParseUint(nextPageToken, 10, 31)
	if err != nil {
		return fmt.Errorf("invalid nextPageToken: must be a non-negative integer (0..%d), got %q: %w", math.MaxInt32, nextPageToken, err)
	}
	return nil
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
		PageSize:  defaultPageSize,
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
