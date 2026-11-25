package filter

import (
	"fmt"
	"strings"

	"github.com/kubeflow/hub/internal/platform/db/constants"
	"github.com/kubeflow/hub/internal/platform/db/dbutil"
	"gorm.io/gorm"
)

// QueryBuilder builds GORM queries from filter expressions
type QueryBuilder struct {
	entityType     EntityType
	restEntityType RestEntityType
	tablePrefix    string
	joinCounter    int
	db             *gorm.DB
	mappingFuncs   EntityMappingFunctions
}

// NewQueryBuilderForRestEntity creates a new query builder for the specified REST entity type.
// mappingFuncs must not be nil - callers must provide their own EntityMappingFunctions.
func NewQueryBuilderForRestEntity(restEntityType RestEntityType, mappingFuncs EntityMappingFunctions) *QueryBuilder {
	entityType := mappingFuncs.GetMLMDEntityType(restEntityType)

	var tablePrefix string
	switch entityType {
	case EntityTypeContext:
		tablePrefix = "Context"
	case EntityTypeArtifact:
		tablePrefix = "Artifact"
	case EntityTypeExecution:
		tablePrefix = "Execution"
	default:
		tablePrefix = "Context"
	}

	return &QueryBuilder{
		entityType:     entityType,
		restEntityType: restEntityType,
		tablePrefix:    tablePrefix,
		joinCounter:    0,
		mappingFuncs:   mappingFuncs,
	}
}

// TablePrefix returns the table prefix for the query builder (exported for test use).
func (qb *QueryBuilder) TablePrefix() string {
	return qb.tablePrefix
}

// EntityType returns the entity type for the query builder (exported for test use).
func (qb *QueryBuilder) EntityType() EntityType {
	return qb.entityType
}

// RestEntityType returns the REST entity type for the query builder (exported for test use).
func (qb *QueryBuilder) RestEntityType() RestEntityType {
	return qb.restEntityType
}

func (qb *QueryBuilder) BuildQuery(db *gorm.DB, expr *FilterExpression) *gorm.DB {
	if expr == nil {
		return db
	}

	qb.db = db
	qb.applyDatabaseQuoting()

	return qb.buildExpression(db, expr)
}

func (qb *QueryBuilder) applyDatabaseQuoting() {
	if qb.db == nil {
		return
	}
	unquotedName := strings.Trim(qb.tablePrefix, "`\"")
	qb.tablePrefix = dbutil.QuoteTableName(qb.db, unquotedName)
}

func (qb *QueryBuilder) quoteTableName(tableName string) string {
	return dbutil.QuoteTableName(qb.db, tableName)
}

func (qb *QueryBuilder) buildExpression(db *gorm.DB, expr *FilterExpression) *gorm.DB {
	if expr.IsLeaf {
		return qb.buildLeafExpression(db, expr)
	}

	switch expr.Operator {
	case "AND":
		artifactConditions := qb.collectArtifactConditions(expr)
		if len(artifactConditions) > 1 {
			nonArtifactExpr := qb.removeArtifactConditions(expr)

			combinedArtifact := qb.buildCombinedArtifactExistsCondition(artifactConditions)
			db = db.Where(combinedArtifact.condition, combinedArtifact.args...)

			if nonArtifactExpr != nil {
				db = qb.buildExpression(db, nonArtifactExpr)
			}
			return db
		}

		leftQuery := qb.buildExpression(db, expr.Left)
		return qb.buildExpression(leftQuery, expr.Right)

	case "OR":
		leftCondition := qb.buildConditionString(expr.Left)
		rightCondition := qb.buildConditionString(expr.Right)

		condition := fmt.Sprintf("(%s OR %s)", leftCondition.condition, rightCondition.condition)
		args := append(leftCondition.args, rightCondition.args...)

		return db.Where(condition, args...)

	default:
		return db
	}
}

type ConditionResult struct {
	Condition string
	Args      []any
}

// BuildConditionString builds a SQL condition string from a FilterExpression.
func (qb *QueryBuilder) BuildConditionString(expr *FilterExpression) ConditionResult {
	r := qb.buildConditionString(expr)
	return ConditionResult{Condition: r.condition, Args: r.args}
}

type conditionResult struct {
	condition string
	args      []any
}

func (qb *QueryBuilder) buildConditionString(expr *FilterExpression) conditionResult {
	if expr.IsLeaf {
		return qb.buildLeafConditionString(expr)
	}

	switch expr.Operator {
	case "AND":
		artifactConditions := qb.collectArtifactConditions(expr)
		if len(artifactConditions) > 1 {
			nonArtifactExpr := qb.removeArtifactConditions(expr)

			combinedArtifact := qb.buildCombinedArtifactExistsCondition(artifactConditions)

			if nonArtifactExpr == nil {
				return combinedArtifact
			}

			nonArtifact := qb.buildConditionString(nonArtifactExpr)
			condition := fmt.Sprintf("(%s AND %s)", nonArtifact.condition, combinedArtifact.condition)
			args := append(nonArtifact.args, combinedArtifact.args...)
			return conditionResult{condition: condition, args: args}
		}

		left := qb.buildConditionString(expr.Left)
		right := qb.buildConditionString(expr.Right)

		condition := fmt.Sprintf("(%s AND %s)", left.condition, right.condition)
		args := append(left.args, right.args...)

		return conditionResult{condition: condition, args: args}

	case "OR":
		left := qb.buildConditionString(expr.Left)
		right := qb.buildConditionString(expr.Right)

		condition := fmt.Sprintf("(%s OR %s)", left.condition, right.condition)
		args := append(left.args, right.args...)

		return conditionResult{condition: condition, args: args}
	}

	return conditionResult{condition: "1=1", args: []any{}}
}

type artifactConditionInfo struct {
	propDef      PropertyDefinition
	explicitType string
	operator     string
	value        any
}

func (qb *QueryBuilder) isArtifactPropertyCondition(expr *FilterExpression) bool {
	if !expr.IsLeaf {
		return false
	}
	propRef := qb.buildPropertyReference(expr)
	return propRef.PropertyDef.Location == RelatedEntity &&
		propRef.PropertyDef.RelatedEntityType == RelatedEntityArtifact
}

func (qb *QueryBuilder) collectArtifactConditions(expr *FilterExpression) []artifactConditionInfo {
	if expr.IsLeaf {
		if qb.isArtifactPropertyCondition(expr) {
			propRef := qb.buildPropertyReference(expr)
			return []artifactConditionInfo{{
				propDef:      propRef.PropertyDef,
				explicitType: propRef.ExplicitType,
				operator:     expr.Operator,
				value:        expr.Value,
			}}
		}
		return nil
	}

	if expr.Operator != "AND" {
		return nil
	}

	var conditions []artifactConditionInfo
	conditions = append(conditions, qb.collectArtifactConditions(expr.Left)...)
	conditions = append(conditions, qb.collectArtifactConditions(expr.Right)...)
	return conditions
}

func (qb *QueryBuilder) removeArtifactConditions(expr *FilterExpression) *FilterExpression {
	if expr.IsLeaf {
		if qb.isArtifactPropertyCondition(expr) {
			return nil
		}
		return expr
	}

	if expr.Operator != "AND" {
		return expr
	}

	left := qb.removeArtifactConditions(expr.Left)
	right := qb.removeArtifactConditions(expr.Right)

	if left == nil && right == nil {
		return nil
	}
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}

	return &FilterExpression{
		Operator: "AND",
		Left:     left,
		Right:    right,
	}
}

func (qb *QueryBuilder) buildCombinedArtifactExistsCondition(conditions []artifactConditionInfo) conditionResult {
	if len(conditions) == 0 {
		return conditionResult{condition: "1=1", args: []any{}}
	}

	if len(conditions) == 1 {
		c := conditions[0]
		return qb.buildRelatedEntityPropertyConditionString(c.propDef, c.explicitType, c.operator, c.value)
	}

	qb.joinCounter++
	baseCounter := qb.joinCounter

	attrAlias := fmt.Sprintf("attr_%d", baseCounter)
	artAlias := fmt.Sprintf("art_%d", baseCounter)
	attrTable := qb.quoteTableName("Attribution")
	artTable := qb.quoteTableName("Artifact")
	propTable := qb.quoteTableName("ArtifactProperty")

	var propertyJoins []string
	var whereConditions []string
	var joinArgs []any
	var whereArgs []any

	for i, c := range conditions {
		propAlias := fmt.Sprintf("artprop_%d_%d", baseCounter, i)
		valueCondition := qb.buildValueCondition(propAlias, c.explicitType, c.operator, c.value)

		join := fmt.Sprintf("JOIN %s %s ON %s.artifact_id = %s.id AND %s.name = ?",
			propTable, propAlias, propAlias, artAlias, propAlias)
		propertyJoins = append(propertyJoins, join)
		joinArgs = append(joinArgs, c.propDef.RelatedProperty)

		whereConditions = append(whereConditions, valueCondition.condition)
		whereArgs = append(whereArgs, valueCondition.args...)
	}

	subquery := fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s %s "+
			"JOIN %s %s ON %s.id = %s.artifact_id "+
			"%s "+
			"WHERE %s.context_id = %s.id AND %s)",
		attrTable, attrAlias,
		artTable, artAlias, artAlias, attrAlias,
		strings.Join(propertyJoins, " "),
		attrAlias, qb.tablePrefix, strings.Join(whereConditions, " AND "))

	args := append(joinArgs, whereArgs...)

	return conditionResult{condition: subquery, args: args}
}

// BuildPropertyReference is exported for test use.
func (qb *QueryBuilder) BuildPropertyReference(expr *FilterExpression) *PropertyReference {
	return qb.buildPropertyReference(expr)
}

func (qb *QueryBuilder) buildPropertyReference(expr *FilterExpression) *PropertyReference {
	var propDef PropertyDefinition
	propertyName := expr.Property

	var explicitType string
	if parts := strings.Split(propertyName, "."); len(parts) >= 2 {
		lastPart := parts[len(parts)-1]
		switch lastPart {
		case StringValueType, DoubleValueType, IntValueType, BoolValueType, ArrayValueType:
			propertyName = strings.Join(parts[:len(parts)-1], ".")
			explicitType = lastPart
		}
	}

	if qb.restEntityType != "" {
		propDef = qb.mappingFuncs.GetPropertyDefinitionForRestEntity(qb.restEntityType, propertyName)
	} else {
		propDef = GetPropertyDefinition(qb.entityType, propertyName)
	}

	propName := propertyName
	if propDef.Location == PropertyTable && propDef.Column != "" {
		propName = propDef.Column
	}

	propRef := &PropertyReference{
		Name:         propName,
		IsCustom:     propDef.Location == Custom,
		ValueType:    propDef.ValueType,
		ExplicitType: explicitType,
		PropertyDef:  propDef,
	}

	if explicitType != "" {
		propRef.ValueType = explicitType
	} else if propRef.IsCustom {
		propRef.ValueType = qb.inferValueTypeFromInterface(expr.Value)
	}

	return propRef
}

func (qb *QueryBuilder) buildLeafExpression(db *gorm.DB, expr *FilterExpression) *gorm.DB {
	propRef := qb.buildPropertyReference(expr)
	return qb.buildPropertyCondition(db, propRef, expr.Operator, expr.Value)
}

func (qb *QueryBuilder) buildLeafConditionString(expr *FilterExpression) conditionResult {
	propRef := qb.buildPropertyReference(expr)
	return qb.buildPropertyConditionString(propRef, expr.Operator, expr.Value)
}

func (qb *QueryBuilder) inferValueTypeFromInterface(value any) string {
	switch value.(type) {
	case string:
		return StringValueType
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return IntValueType
	case float32, float64:
		return DoubleValueType
	case bool:
		return BoolValueType
	default:
		return StringValueType
	}
}

func (qb *QueryBuilder) buildPropertyCondition(db *gorm.DB, propRef *PropertyReference, operator string, value any) *gorm.DB {
	propDef := propRef.PropertyDef

	switch propDef.Location {
	case EntityTable:
		return qb.buildEntityTablePropertyCondition(db, propRef, operator, value)
	case PropertyTable, Custom:
		return qb.buildPropertyTableCondition(db, propRef, operator, value)
	case RelatedEntity:
		return qb.buildRelatedEntityPropertyCondition(db, propDef, propRef.ValueType, operator, value)
	default:
		return qb.buildEntityTablePropertyCondition(db, propRef, operator, value)
	}
}

func (qb *QueryBuilder) buildPropertyConditionString(propRef *PropertyReference, operator string, value any) conditionResult {
	propDef := propRef.PropertyDef

	switch propDef.Location {
	case EntityTable:
		return qb.buildEntityTablePropertyConditionString(propRef, operator, value)
	case PropertyTable, Custom:
		return qb.buildPropertyTableConditionString(propRef, operator, value)
	case RelatedEntity:
		return qb.buildRelatedEntityPropertyConditionString(propDef, propRef.ValueType, operator, value)
	default:
		return qb.buildEntityTablePropertyConditionString(propRef, operator, value)
	}
}

// ConvertStateValue converts string state values to integers based on entity type
func (qb *QueryBuilder) ConvertStateValue(propertyName string, value any) any {
	if propertyName == "state" {
		if strValue, ok := value.(string); ok {
			switch qb.entityType {
			case EntityTypeArtifact:
				if intValue, exists := constants.ArtifactStateMapping[strValue]; exists {
					return int32(intValue)
				}
				return int32(-1)
			case EntityTypeExecution:
				if intValue, exists := constants.ExecutionStateMapping[strValue]; exists {
					return int32(intValue)
				}
				return int32(-1)
			case EntityTypeContext:
				return value
			}
		}
	}
	return value
}

func (qb *QueryBuilder) buildEntityTablePropertyCondition(db *gorm.DB, propRef *PropertyReference, operator string, value any) *gorm.DB {
	propDef := GetPropertyDefinition(qb.entityType, propRef.Name)
	column := fmt.Sprintf("%s.%s", qb.tablePrefix, propDef.Column)

	value = qb.ConvertStateValue(propRef.Name, value)

	if operator == "=" && qb.restEntityType != "" {
		if expander, ok := qb.mappingFuncs.(EqualityExpander); ok {
			if likeArg, use := expander.GetEqualityExpansion(qb.restEntityType, propRef.Name, value); use {
				condition := fmt.Sprintf("(%s = ? OR %s LIKE ?)", column, column)
				return db.Where(condition, value, likeArg)
			}
		}
	}

	if qb.restEntityType != "" && propRef.Name == "name" && qb.mappingFuncs.IsChildEntity(qb.restEntityType) {
		if strValue, ok := value.(string); ok {
			if operator == "=" {
				operator = "LIKE"
				value = "%:" + escapeLikePattern(strValue)
			} else if operator == "LIKE" && !strings.Contains(strValue, ":") {
				if !strings.HasPrefix(strValue, "%") {
					value = "%:" + strValue
				}
			}
		}
	}

	if operator == "ILIKE" {
		return qb.buildCaseInsensitiveLikeCondition(db, column, value)
	}

	condition := qb.buildOperatorCondition(column, operator, value)
	return db.Where(condition.condition, condition.args...)
}

func (qb *QueryBuilder) buildEntityTablePropertyConditionString(propRef *PropertyReference, operator string, value any) conditionResult {
	propDef := GetPropertyDefinition(qb.entityType, propRef.Name)
	column := fmt.Sprintf("%s.%s", qb.tablePrefix, propDef.Column)

	value = qb.ConvertStateValue(propRef.Name, value)

	if operator == "=" && qb.restEntityType != "" {
		if expander, ok := qb.mappingFuncs.(EqualityExpander); ok {
			if likeArg, use := expander.GetEqualityExpansion(qb.restEntityType, propRef.Name, value); use {
				return conditionResult{condition: fmt.Sprintf("(%s = ? OR %s LIKE ?)", column, column), args: []any{value, likeArg}}
			}
		}
	}

	if qb.restEntityType != "" && propRef.Name == "name" && qb.mappingFuncs.IsChildEntity(qb.restEntityType) {
		if strValue, ok := value.(string); ok {
			if operator == "=" {
				operator = "LIKE"
				value = "%:" + escapeLikePattern(strValue)
			} else if operator == "LIKE" && !strings.Contains(strValue, ":") {
				if !strings.HasPrefix(strValue, "%") {
					value = "%:" + strValue
				}
			}
		}
	}

	return qb.buildOperatorCondition(column, operator, value)
}

func (qb *QueryBuilder) buildPropertyTableCondition(db *gorm.DB, propRef *PropertyReference, operator string, value any) *gorm.DB {
	qb.joinCounter++
	alias := fmt.Sprintf("prop_%d", qb.joinCounter)

	var propertyTable string
	var joinCondition string

	switch qb.entityType {
	case EntityTypeContext:
		propertyTable = qb.quoteTableName("ContextProperty")
		joinCondition = fmt.Sprintf("%s.context_id = %s.id", alias, qb.tablePrefix)
	case EntityTypeArtifact:
		propertyTable = qb.quoteTableName("ArtifactProperty")
		joinCondition = fmt.Sprintf("%s.artifact_id = %s.id", alias, qb.tablePrefix)
	case EntityTypeExecution:
		propertyTable = qb.quoteTableName("ExecutionProperty")
		joinCondition = fmt.Sprintf("%s.execution_id = %s.id", alias, qb.tablePrefix)
	}

	joinClause := fmt.Sprintf("JOIN %s %s ON %s", propertyTable, alias, joinCondition)
	db = db.Joins(joinClause)

	db = db.Where(fmt.Sprintf("%s.name = ?", alias), propRef.Name)

	if operator == "ILIKE" {
		valueColumn := fmt.Sprintf("%s.%s", alias, propRef.ValueType)
		return qb.buildCaseInsensitiveLikeCondition(db, valueColumn, value)
	}

	valueType, inferredAsInt := qb.determinePropertyValueType(propRef, value)

	var condition conditionResult

	if inferredAsInt {
		intColumn := fmt.Sprintf("%s.int_value", alias)
		doubleColumn := fmt.Sprintf("%s.double_value", alias)
		condition = qb.buildDualColumnCondition(intColumn, doubleColumn, operator, value)
	} else if valueType == ArrayValueType && db.Name() == "postgres" {
		valueColumn := fmt.Sprintf("%s.%s", alias, StringValueType)
		condition = qb.buildJSONOperatorCondition(valueColumn, operator, value)
	} else {
		var valueColumn string
		if valueType == ArrayValueType {
			valueColumn = fmt.Sprintf("%s.%s", alias, StringValueType)
		} else {
			valueColumn = fmt.Sprintf("%s.%s", alias, valueType)
		}
		condition = qb.buildOperatorCondition(valueColumn, operator, value)
	}

	return db.Where(condition.condition, condition.args...)
}

func (qb *QueryBuilder) buildPropertyTableConditionString(propRef *PropertyReference, operator string, value any) conditionResult {
	var propertyTable string
	var joinColumn string

	switch qb.entityType {
	case EntityTypeContext:
		propertyTable = qb.quoteTableName("ContextProperty")
		joinColumn = "context_id"
	case EntityTypeArtifact:
		propertyTable = qb.quoteTableName("ArtifactProperty")
		joinColumn = "artifact_id"
	case EntityTypeExecution:
		propertyTable = qb.quoteTableName("ExecutionProperty")
		joinColumn = "execution_id"
	}

	valueType, inferredAsInt := qb.determinePropertyValueType(propRef, value)

	var condition conditionResult

	if inferredAsInt {
		intColumn := fmt.Sprintf("%s.int_value", propertyTable)
		doubleColumn := fmt.Sprintf("%s.double_value", propertyTable)
		condition = qb.buildDualColumnCondition(intColumn, doubleColumn, operator, value)
	} else if valueType == ArrayValueType && qb.db.Name() == "postgres" {
		valueColumn := fmt.Sprintf("%s.%s", propertyTable, StringValueType)
		condition = qb.buildJSONOperatorCondition(valueColumn, operator, value)
	} else {
		var valueColumn string
		if valueType == ArrayValueType {
			valueColumn = fmt.Sprintf("%s.%s", propertyTable, StringValueType)
		} else {
			valueColumn = fmt.Sprintf("%s.%s", propertyTable, valueType)
		}
		condition = qb.buildOperatorCondition(valueColumn, operator, value)
	}

	subquery := fmt.Sprintf("EXISTS (SELECT 1 FROM %s WHERE %s.%s = %s.id AND %s.name = ? AND (%s))",
		propertyTable, propertyTable, joinColumn, qb.tablePrefix, propertyTable, condition.condition)

	args := []any{propRef.Name}
	args = append(args, condition.args...)

	return conditionResult{condition: subquery, args: args}
}

func (qb *QueryBuilder) buildRelatedEntityPropertyCondition(db *gorm.DB, propDef PropertyDefinition, explicitType string, operator string, value any) *gorm.DB {
	conditionResult := qb.buildRelatedEntityPropertyConditionString(propDef, explicitType, operator, value)
	return db.Where(conditionResult.condition, conditionResult.args...)
}

func (qb *QueryBuilder) buildRelatedEntityPropertyConditionString(propDef PropertyDefinition, explicitType string, operator string, value any) conditionResult {
	if qb.entityType != EntityTypeContext || propDef.RelatedEntityType != RelatedEntityArtifact {
		return conditionResult{condition: "1=1", args: []any{}}
	}

	qb.joinCounter++

	aliases := qb.createRelatedEntityAliases(qb.joinCounter)

	valueCondition := qb.buildValueCondition(aliases.propertyAlias, explicitType, operator, value)

	subquery := fmt.Sprintf(
		"EXISTS (SELECT 1 FROM %s %s "+
			"JOIN %s %s ON %s.id = %s.artifact_id "+
			"JOIN %s %s ON %s.artifact_id = %s.id "+
			"WHERE %s.context_id = %s.id AND %s.name = ? AND %s)",
		aliases.attributionTable, aliases.attributionAlias,
		aliases.entityTable, aliases.entityAlias, aliases.entityAlias, aliases.attributionAlias,
		aliases.propertyTable, aliases.propertyAlias, aliases.propertyAlias, aliases.entityAlias,
		aliases.attributionAlias, qb.tablePrefix, aliases.propertyAlias, valueCondition.condition)

	args := []any{propDef.RelatedProperty}
	args = append(args, valueCondition.args...)

	return conditionResult{condition: subquery, args: args}
}

type relatedEntityAliases struct {
	attributionAlias string
	entityAlias      string
	propertyAlias    string
	attributionTable string
	entityTable      string
	propertyTable    string
}

func (qb *QueryBuilder) createRelatedEntityAliases(counter int) relatedEntityAliases {
	return relatedEntityAliases{
		attributionAlias: fmt.Sprintf("attr_%d", counter),
		entityAlias:      fmt.Sprintf("art_%d", counter),
		propertyAlias:    fmt.Sprintf("artprop_%d", counter),
		attributionTable: qb.quoteTableName("Attribution"),
		entityTable:      qb.quoteTableName("Artifact"),
		propertyTable:    qb.quoteTableName("ArtifactProperty"),
	}
}

func (qb *QueryBuilder) determineValueType(explicitType string, value any) (valueType string, inferredAsInt bool) {
	if explicitType != "" {
		return explicitType, false
	}

	inferredType := qb.inferValueTypeFromInterface(value)
	if inferredType == IntValueType {
		return inferredType, true
	}

	return inferredType, false
}

func (qb *QueryBuilder) determinePropertyValueType(propRef *PropertyReference, value any) (valueType string, inferredAsInt bool) {
	if propRef.IsCustom {
		return qb.determineValueType(propRef.ExplicitType, value)
	}

	if propRef.ExplicitType != "" {
		return propRef.ExplicitType, false
	}

	return propRef.ValueType, false
}

func (qb *QueryBuilder) buildDualColumnCondition(intColumn, doubleColumn, operator string, value any) conditionResult {
	intCondition := qb.buildOperatorCondition(intColumn, operator, value)
	doubleCondition := qb.buildOperatorCondition(doubleColumn, operator, value)

	return conditionResult{
		condition: fmt.Sprintf("(%s OR %s)", intCondition.condition, doubleCondition.condition),
		args:      append(intCondition.args, doubleCondition.args...),
	}
}

func (qb *QueryBuilder) buildValueCondition(propertyAlias string, explicitType string, operator string, value any) conditionResult {
	valueType, inferredAsInt := qb.determineValueType(explicitType, value)

	if inferredAsInt {
		intColumn := fmt.Sprintf("%s.int_value", propertyAlias)
		doubleColumn := fmt.Sprintf("%s.double_value", propertyAlias)
		return qb.buildDualColumnCondition(intColumn, doubleColumn, operator, value)
	}

	valueColumn := fmt.Sprintf("%s.%s", propertyAlias, valueType)
	return qb.buildOperatorCondition(valueColumn, operator, value)
}

func (qb *QueryBuilder) buildOperatorCondition(column string, operator string, value any) conditionResult {
	switch operator {
	case "=":
		return conditionResult{condition: fmt.Sprintf("%s = ?", column), args: []any{value}}
	case "!=":
		return conditionResult{condition: fmt.Sprintf("%s != ?", column), args: []any{value}}
	case ">":
		return conditionResult{condition: fmt.Sprintf("%s > ?", column), args: []any{value}}
	case ">=":
		return conditionResult{condition: fmt.Sprintf("%s >= ?", column), args: []any{value}}
	case "<":
		return conditionResult{condition: fmt.Sprintf("%s < ?", column), args: []any{value}}
	case "<=":
		return conditionResult{condition: fmt.Sprintf("%s <= ?", column), args: []any{value}}
	case "LIKE":
		return conditionResult{condition: fmt.Sprintf("%s LIKE ?", column), args: []any{value}}
	case "ILIKE":
		if strValue, ok := value.(string); ok {
			return conditionResult{
				condition: fmt.Sprintf("UPPER(%s) LIKE UPPER(?)", column),
				args:      []any{strValue},
			}
		}
		return conditionResult{condition: fmt.Sprintf("%s LIKE ?", column), args: []any{value}}
	case "IN":
		if valueSlice, ok := value.([]any); ok {
			if len(valueSlice) == 0 {
				return conditionResult{condition: "1 = 0", args: []any{}}
			}
			condition := fmt.Sprintf("%s IN (?%s)", column, strings.Repeat(",?", len(valueSlice)-1))
			return conditionResult{condition: condition, args: valueSlice}
		}
		return conditionResult{condition: fmt.Sprintf("%s IN (?)", column), args: []any{value}}
	default:
		return conditionResult{condition: fmt.Sprintf("%s = ?", column), args: []any{value}}
	}
}

//nolint:staticcheck
func (qb *QueryBuilder) buildCaseInsensitiveLikeCondition(db *gorm.DB, column string, value any) *gorm.DB {
	if strValue, ok := value.(string); ok {
		switch db.Name() {
		case "postgres":
			return db.Where(fmt.Sprintf("%s ILIKE ?", column), strValue)
		case "mysql":
			return db.Where(fmt.Sprintf("%s LIKE ? COLLATE utf8mb4_unicode_ci", column), strValue)
		case "sqlite":
			return db.Where(fmt.Sprintf("%s LIKE ?", column), strValue)
		default:
			return db.Where(fmt.Sprintf("UPPER(%s) LIKE UPPER(?)", column), strValue)
		}
	}

	return db.Where(fmt.Sprintf("%s LIKE ?", column), value)
}

func (qb *QueryBuilder) buildJSONOperatorCondition(column string, operator string, value any) conditionResult {
	switch operator {
	case "IN":
		if valueSlice, ok := value.([]any); ok {
			if len(valueSlice) == 0 {
				return conditionResult{condition: "1 = 0", args: []any{}}
			}
			return conditionResult{
				condition: fmt.Sprintf("%s IS JSON ARRAY AND %s::jsonb ? array[?%s]", column, column, strings.Repeat(",?", len(valueSlice)-1)),
				args:      append([]any{gorm.Expr("?|")}, valueSlice...),
			}
		}
		fallthrough
	case "=":
		return conditionResult{condition: fmt.Sprintf("%s IS JSON ARRAY AND %s::jsonb ? array[?]", column, column), args: []any{gorm.Expr("?|"), value}}
	case "!=":
		return conditionResult{condition: fmt.Sprintf("%s IS NOT JSON ARRAY OR NOT %s::jsonb ? array[?]", column, column), args: []any{gorm.Expr("?|"), value}}
	default:
		return qb.buildOperatorCondition(column, operator, value)
	}
}

// escapeLikePattern escapes the LIKE metacharacters (\, %, _) in s so that it matches literally.
func escapeLikePattern(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
