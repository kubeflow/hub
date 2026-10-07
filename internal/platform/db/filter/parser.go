package filter

import (
	"fmt"
	"strings"
	"sync"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Constants for property value types
const (
	StringValueType = "string_value"
	DoubleValueType = "double_value"
	IntValueType    = "int_value"
	BoolValueType   = "bool_value"
	ArrayValueType  = "array_value"
)

var sqlLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "whitespace", Pattern: `\s+`},
	{Name: "Comment", Pattern: `--[^\r\n]*`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
	{Name: "Float", Pattern: `[-+]?\d*\.\d+([eE][-+]?\d+)?|[-+]?\d+[eE][-+]?\d+`},
	{Name: "Int", Pattern: `[-+]?\d+`},
	{Name: "String", Pattern: `'([^'\\]|\\.)*'|"([^"\\]|\\.)*"`},
	{Name: "EscapedIdent", Pattern: "`([^`\\\\]|\\\\.)*`"},
	{Name: "Operators", Pattern: `>=|<=|!=|<>|=|>|<`},
	{Name: "Punct", Pattern: `[().,]`},
})

var (
	globalParser *participle.Parser[WhereClause]
	parserOnce   sync.Once
)

func initParser() {
	globalParser = participle.MustBuild[WhereClause](
		participle.Lexer(sqlLexer),
		participle.Elide("whitespace", "Comment"),
		participle.CaseInsensitive("OR", "AND", "LIKE", "ILIKE", "IN", "true", "false", "TRUE", "FALSE"),
		participle.CaseInsensitive(StringValueType, DoubleValueType, IntValueType, BoolValueType, ArrayValueType),
	)
}

func getParser() *participle.Parser[WhereClause] {
	parserOnce.Do(initParser)
	return globalParser
}

//nolint:govet
type WhereClause struct {
	Expression *Expression `@@`
}

//nolint:govet
type Expression struct {
	Or *OrExpression `@@`
}

//nolint:govet
type OrExpression struct {
	Left  *AndExpression   `@@`
	Right []*AndExpression `("OR" @@)*`
}

//nolint:govet
type AndExpression struct {
	Left  *Term   `@@`
	Right []*Term `("AND" @@)*`
}

//nolint:govet
type Term struct {
	Group      *Expression `"(" @@ ")"`
	Comparison *Comparison `| @@`
}

//nolint:govet
type Comparison struct {
	Left     *PropertyRef `@@`
	Operator string       `@("=" | "!=" | "<>" | ">=" | "<=" | ">" | "<" | "LIKE" | "ILIKE" | "IN")`
	Right    *Value       `@@`
}

//nolint:govet
type PropertyRef struct {
	EscapedName string   `@EscapedIdent`
	Name        string   `| @Ident`
	Path        []string `("." @Ident)*`
	Type        string   `("." @("string_value" | "double_value" | "int_value" | "bool_value" | "array_value"))?`
}

//nolint:govet
type Value struct {
	String    *string    `@String`
	Integer   *int64     `| @Int`
	Float     *float64   `| @Float`
	Boolean   *string    `| @("true" | "false" | "TRUE" | "FALSE")`
	ValueList *ValueList `| @@`
}

//nolint:govet
type ValueList struct {
	Values []*SingleValue `"(" (@@  ("," @@)*)? ")"`
}

//nolint:govet
type SingleValue struct {
	String  *string  `@String`
	Integer *int64   `| @Int`
	Float   *float64 `| @Float`
	Boolean *string  `| @("true" | "false" | "TRUE" | "FALSE")`
}

type FilterExpression struct {
	Left     *FilterExpression
	Right    *FilterExpression
	Operator string
	Property string
	Value    any
	IsLeaf   bool
}

type PropertyReference struct {
	Name         string
	IsCustom     bool
	ValueType    string
	ExplicitType string
	IsEscaped    bool
	PropertyDef  PropertyDefinition
}

func Parse(input string) (*FilterExpression, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}

	parser := getParser()
	whereClause, err := parser.ParseString("", input)
	if err != nil {
		return nil, fmt.Errorf("error parsing filter query: %w", err)
	}

	return convertToFilterExpression(whereClause.Expression), nil
}

func convertToFilterExpression(expr *Expression) *FilterExpression {
	return convertOrExpression(expr.Or)
}

func convertOrExpression(expr *OrExpression) *FilterExpression {
	left := convertAndExpression(expr.Left)

	for _, right := range expr.Right {
		rightExpr := convertAndExpression(right)
		left = &FilterExpression{
			Left:     left,
			Right:    rightExpr,
			Operator: "OR",
			IsLeaf:   false,
		}
	}

	return left
}

func convertAndExpression(expr *AndExpression) *FilterExpression {
	left := convertTerm(expr.Left)

	for _, right := range expr.Right {
		rightExpr := convertTerm(right)
		left = &FilterExpression{
			Left:     left,
			Right:    rightExpr,
			Operator: "AND",
			IsLeaf:   false,
		}
	}

	return left
}

func convertTerm(term *Term) *FilterExpression {
	if term.Group != nil {
		return convertToFilterExpression(term.Group)
	}

	return convertComparison(term.Comparison)
}

func convertComparison(comp *Comparison) *FilterExpression {
	propRef := convertPropertyRef(comp.Left, comp.Right)
	value := convertValue(comp.Right)

	propertyName := propRef.Name

	if comp.Left.Type != "" {
		propertyName = propRef.Name + "." + comp.Left.Type
	}

	return &FilterExpression{
		Property: propertyName,
		Operator: comp.Operator,
		Value:    value,
		IsLeaf:   true,
	}
}

func convertPropertyRef(prop *PropertyRef, value *Value) *PropertyReference {
	var name string
	var isEscaped bool
	if prop.EscapedName != "" {
		name = strings.Trim(prop.EscapedName, "`")
		name = strings.ReplaceAll(name, `\.`, `.`)
		name = strings.ReplaceAll(name, `\\`, `\`)
		isEscaped = true
	} else {
		name = prop.Name
		if len(prop.Path) > 0 {
			name = name + "." + strings.Join(prop.Path, ".")
		}
		isEscaped = false
	}

	var valueType string

	if prop.Type != "" {
		valueType = prop.Type
	} else {
		valueType = inferValueType(value)
	}

	return &PropertyReference{
		Name:      name,
		IsCustom:  true,
		ValueType: valueType,
		IsEscaped: isEscaped,
	}
}

func convertValue(val *Value) any {
	if val.String != nil {
		return unquoteStringValue(*val.String)
	}

	if val.Integer != nil {
		return *val.Integer
	}

	if val.Float != nil {
		return *val.Float
	}

	if val.Boolean != nil {
		return strings.ToLower(*val.Boolean) == "true"
	}

	if val.ValueList != nil {
		var values []any
		for _, singleVal := range val.ValueList.Values {
			values = append(values, convertSingleValue(singleVal))
		}
		return values
	}

	return nil
}

func convertSingleValue(val *SingleValue) any {
	if val.String != nil {
		return unquoteStringValue(*val.String)
	}

	if val.Integer != nil {
		return *val.Integer
	}

	if val.Float != nil {
		return *val.Float
	}

	if val.Boolean != nil {
		return strings.ToLower(*val.Boolean) == "true"
	}

	return nil
}

func unquoteStringValue(str string) string {
	// The lexer only emits String tokens wrapped in a matching pair of quotes,
	// so strip just that pair and keep quotes that are part of the value.
	result := str[1 : len(str)-1]
	result = strings.ReplaceAll(result, `\"`, `"`)
	result = strings.ReplaceAll(result, `\'`, `'`)
	result = strings.ReplaceAll(result, `\\`, `\`)
	return result
}

func inferValueType(val *Value) string {
	if val.ValueList != nil && len(val.ValueList.Values) > 0 {
		return inferSingleValueType(val.ValueList.Values[0])
	}
	singleVal := &SingleValue{
		String:  val.String,
		Integer: val.Integer,
		Float:   val.Float,
		Boolean: val.Boolean,
	}
	return inferSingleValueType(singleVal)
}

func inferSingleValueType(val *SingleValue) string {
	if val.String != nil {
		return StringValueType
	}
	if val.Integer != nil {
		return IntValueType
	}
	if val.Float != nil {
		return DoubleValueType
	}
	if val.Boolean != nil {
		return BoolValueType
	}
	return StringValueType
}
