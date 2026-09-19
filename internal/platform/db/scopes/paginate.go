package scopes

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	"github.com/kubeflow/hub/internal/platform/db/entity"
	"gorm.io/gorm"
)

// AllowedOrderByColumns contains allowed column names for orderBy to prevent SQL injection
var AllowedOrderByColumns = map[string]string{
	"ID":               "id",
	"CREATE_TIME":      "create_time_since_epoch",
	"LAST_UPDATE_TIME": "last_update_time_since_epoch",
	"id":               "id",
}

// AllowedSortOrders contains allowed sort orders to prevent SQL injection
var AllowedSortOrders = map[string]string{
	"ASC":  entity.SortOrderAsc,
	"DESC": entity.SortOrderDesc,
}

// IsValidTablePrefix validates table prefix to prevent SQL injection
func IsValidTablePrefix(tablePrefix string) bool {
	if tablePrefix == "" {
		return true
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, tablePrefix)
	return matched
}

func Paginate(value any, pagination *entity.Pagination, db *gorm.DB) func(db *gorm.DB) *gorm.DB {
	return PaginateWithTablePrefix(value, pagination, db, "")
}

func PaginateWithTablePrefix(value any, pagination *entity.Pagination, db *gorm.DB, tablePrefix string) func(db *gorm.DB) *gorm.DB {
	return PaginateWithOptions(value, pagination, db, tablePrefix, nil)
}

func PaginateWithOptions(value any, pagination *entity.Pagination, db *gorm.DB, tablePrefix string, customAllowedColumns map[string]string) func(db *gorm.DB) *gorm.DB {
	pageSize := pagination.GetPageSize()
	orderBy := pagination.GetOrderBy()
	sortOrder := pagination.GetSortOrder()
	nextPageToken := pagination.GetNextPageToken()

	columnsMap := AllowedOrderByColumns
	if customAllowedColumns != nil {
		columnsMap = customAllowedColumns
	}

	return func(db *gorm.DB) *gorm.DB {
		if pageSize > 0 {
			db = db.Limit(int(pageSize) + 1)
		}

		if orderBy != "" && sortOrder != "" {
			sanitizedOrderBy, ok := columnsMap[orderBy]
			if !ok {
				sanitizedOrderBy = entity.DefaultOrderBy
			}

			sanitizedSortOrder := entity.DefaultSortOrder
			if so, ok := AllowedSortOrders[sortOrder]; ok {
				sanitizedSortOrder = so
			}

			orderByColumn := sanitizedOrderBy
			if tablePrefix != "" {
				quotedPrefix := dbutil.QuoteTableName(db, tablePrefix)
				orderByColumn = quotedPrefix + "." + sanitizedOrderBy
			}

			db = db.Order(fmt.Sprintf("%s %s", orderByColumn, sanitizedSortOrder))
		}

		if nextPageToken != "" {
			decodedCursor, err := DecodeCursor(nextPageToken)
			if err != nil {
				_ = db.AddError(fmt.Errorf("invalid nextPageToken: %w", err))
			} else {
				db = buildWhereClause(db, decodedCursor, orderBy, sortOrder, tablePrefix)
			}
		}

		return db
	}
}

func buildWhereClause(db *gorm.DB, cursor *Cursor, orderBy string, sortOrder string, tablePrefix string) *gorm.DB {
	if !IsValidTablePrefix(tablePrefix) {
		tablePrefix = ""
	}

	if tablePrefix != "" {
		tablePrefix = dbutil.QuoteTableName(db, tablePrefix)
	}

	idColumn := "id"
	if tablePrefix != "" {
		idColumn = tablePrefix + ".id"
	}

	orderByColumn, ok := AllowedOrderByColumns[orderBy]
	if !ok {
		orderByColumn = entity.DefaultOrderBy
	}

	if tablePrefix != "" && orderByColumn != "" {
		orderByColumn = tablePrefix + "." + orderByColumn
	}

	sanitizedSortOrder := entity.DefaultSortOrder
	if so, ok := AllowedSortOrders[sortOrder]; ok {
		sanitizedSortOrder = so
	}

	if orderBy == "" {
		if sanitizedSortOrder == "ASC" {
			return db.Where(idColumn+" > ?", cursor.ID)
		}
		return db.Where(idColumn+" < ?", cursor.ID)
	}

	if sanitizedSortOrder == "ASC" {
		return db.Where("("+orderByColumn+" > ? OR ("+orderByColumn+" = ? AND "+idColumn+" > ?))",
			cursor.Value, cursor.Value, cursor.ID)
	}
	return db.Where("("+orderByColumn+" < ? OR ("+orderByColumn+" = ? AND "+idColumn+" < ?))",
		cursor.Value, cursor.Value, cursor.ID)
}

type Cursor struct {
	ID    int32
	Value string
}

func DecodeCursor(token string) (*Cursor, error) {
	if len(token) > 1024 {
		return nil, fmt.Errorf("invalid cursor format")
	}

	decoded, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	// Split only on the first ":" so the value can contain colons (e.g. stored names "sourceId:modelName").
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cursor format")
	}

	id, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return nil, err
	}

	return &Cursor{
		ID:    int32(id),
		Value: parts[1],
	}, nil
}

func CreateNextPageToken(id int32, value any) string {
	var valueString string

	switch tval := value.(type) {
	case string:
		valueString = tval
	case *string:
		if tval != nil {
			valueString = *tval
		}
	case *float64:
		if tval != nil {
			// Shortest representation that parses back to the same float64.
			// A fixed number of decimals rounds values such as 0.8523489932885906,
			// and a cursor that no longer equals the stored value repeats or
			// skips rows at the page boundary.
			valueString = strconv.FormatFloat(*tval, 'g', -1, 64)
		}
	case *int64:
		if tval != nil {
			valueString = fmt.Sprintf("%d", *tval)
		}
	default:
		valueString = fmt.Sprintf("%v", value)
	}
	cursor := fmt.Sprintf("%d:%s", id, valueString)
	return base64.StdEncoding.EncodeToString([]byte(cursor))
}
