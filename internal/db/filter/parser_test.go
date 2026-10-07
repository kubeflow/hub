package filter

import (
	"fmt"
	"testing"
)

// exprToString converts a FilterExpression to a string representation for testing
func exprToString(expr *FilterExpression) string {
	if expr == nil {
		return ""
	}

	if expr.IsLeaf {
		// Handle arrays for IN operator
		if slice, ok := expr.Value.([]any); ok {
			return fmt.Sprintf("%s %s %v", expr.Property, expr.Operator, slice)
		}
		return fmt.Sprintf("%s %s %v", expr.Property, expr.Operator, expr.Value)
	}

	left := exprToString(expr.Left)
	right := exprToString(expr.Right)

	switch expr.Operator {
	case "AND", "OR":
		return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right)
	default:
		return fmt.Sprintf("%s %s %s", left, expr.Operator, right)
	}
}

func TestParseBasicExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple equality",
			input:    `name = "test"`,
			expected: "name = test",
		},
		{
			name:     "Numeric comparison",
			input:    `accuracy > 0.95`,
			expected: "accuracy > 0.95",
		},
		{
			name:     "Boolean value",
			input:    `enabled = true`,
			expected: "enabled = true",
		},
		{
			name:     "Not equal",
			input:    `status != "inactive"`,
			expected: "status != inactive",
		},
		{
			name:     "LIKE operator",
			input:    `name LIKE "%test%"`,
			expected: "name LIKE %test%",
		},
		{
			name:     "ILIKE operator",
			input:    `name ILIKE "%Test%"`,
			expected: "name ILIKE %Test%",
		},
		{
			name:     "Greater than or equal",
			input:    `version >= 1.0`,
			expected: "version >= 1",
		},
		{
			name:     "Less than",
			input:    `priority < 10`,
			expected: "priority < 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParsePropertyTypeSuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Property with double_value suffix",
			input:    `budget.double_value > 12000`,
			expected: "budget.double_value > 12000",
		},
		{
			name:     "Property with int_value suffix",
			input:    `priority.int_value <= 2`,
			expected: "priority.int_value <= 2",
		},
		{
			name:     "Property with string_value suffix",
			input:    `status.string_value = "active"`,
			expected: "status.string_value = active",
		},
		{
			name:     "Property with bool_value suffix",
			input:    `enabled.bool_value = true`,
			expected: "enabled.bool_value = true",
		},
		{
			name:     "Multiple properties with type suffixes",
			input:    `budget.double_value > 10000 AND priority.int_value < 5`,
			expected: "(budget.double_value > 10000 AND priority.int_value < 5)",
		},
		{
			name:     "Mixed properties with and without type suffixes",
			input:    `name = "test" AND budget.double_value > 5000`,
			expected: "(name = test AND budget.double_value > 5000)",
		},
		{
			name:     "Complex expression with type suffixes",
			input:    `(budget.double_value > 10000 OR budget.double_value < 5000) AND active.bool_value = true`,
			expected: "((budget.double_value > 10000 OR budget.double_value < 5000) AND active.bool_value = true)",
		},
		{
			name:     "LIKE pattern with type suffix",
			input:    `description.string_value LIKE "%test%"`,
			expected: "description.string_value LIKE %test%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseComplexExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AND expression",
			input:    `name = "test" AND version > 1.0`,
			expected: "(name = test AND version > 1)",
		},
		{
			name:     "OR expression",
			input:    `status = "active" OR status = "pending"`,
			expected: "(status = active OR status = pending)",
		},
		{
			name:     "Complex with parentheses",
			input:    `(name = "test" OR name = "demo") AND version >= 2.0`,
			expected: "((name = test OR name = demo) AND version >= 2)",
		},
		{
			name:     "Multiple ANDs",
			input:    `name = "test" AND version > 1.0 AND enabled = true`,
			expected: "((name = test AND version > 1) AND enabled = true)",
		},
		{
			name:     "Multiple ORs",
			input:    `status = "active" OR status = "pending" OR status = "reviewing"`,
			expected: "((status = active OR status = pending) OR status = reviewing)",
		},
		{
			name:     "Mixed operators with precedence",
			input:    `name = "test" AND (version > 1.0 OR version < 0.5)`,
			expected: "(name = test AND (version > 1 OR version < 0.5))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseTypeInferenceAndExplicitTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "String inference",
			input:    `framework = "tensorflow"`,
			expected: "framework = tensorflow",
		},
		{
			name:     "Number inference",
			input:    `accuracy > 0.95`,
			expected: "accuracy > 0.95",
		},
		{
			name:     "Boolean inference",
			input:    `enabled = true`,
			expected: "enabled = true",
		},
		{
			name:     "Explicit string type",
			input:    `framework.string_value = "tensorflow"`,
			expected: "framework.string_value = tensorflow",
		},
		{
			name:     "Explicit double type",
			input:    `accuracy.double_value > 0.95`,
			expected: "accuracy.double_value > 0.95",
		},
		{
			name:     "Explicit bool type",
			input:    `enabled.bool_value = true`,
			expected: "enabled.bool_value = true",
		},
		{
			name:     "Explicit int type",
			input:    `count.int_value = 5`,
			expected: "count.int_value = 5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseEscapedProperties(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Escaped property with dots",
			input:    "`mlflow.source.type` = \"notebook\"",
			expected: "mlflow.source.type = notebook",
		},
		{
			name:     "Escaped property with special characters",
			input:    "`custom-metric` > 0.8",
			expected: "custom-metric > 0.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseILIKEOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ILIKE basic",
			input:    `name ILIKE "%Test%"`,
			expected: "name ILIKE %Test%",
		},
		{
			name:     "ILIKE with case variations",
			input:    `description ILIKE "%PyTorch%"`,
			expected: "description ILIKE %PyTorch%",
		},
		{
			name:     "ILIKE in complex expression",
			input:    `name ILIKE "%model%" AND version > 1.0`,
			expected: "(name ILIKE %model% AND version > 1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseINOperator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "IN with integers",
			input:    `id IN (200, 201)`,
			expected: "id IN [200 201]",
		},
		{
			name:     "IN with strings",
			input:    `name IN ("model-a", "model-b")`,
			expected: "name IN [model-a model-b]",
		},
		{
			name:     "IN with mixed types",
			input:    `status IN ("active", "pending", "completed")`,
			expected: "status IN [active pending completed]",
		},
		{
			name:     "IN with single value",
			input:    `id IN (42)`,
			expected: "id IN [42]",
		},
		{
			name:     "IN in complex expression",
			input:    `id IN (200, 201) AND name = "test"`,
			expected: "(id IN [200 201] AND name = test)",
		},
		{
			name:     "IN with OR expression",
			input:    `id IN (200, 201) OR status = "active"`,
			expected: "(id IN [200 201] OR status = active)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			result := exprToString(expr)
			if result != tt.expected {
				t.Errorf("Parse() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Invalid syntax",
			input: `name = `,
		},
		{
			name:  "Missing operator",
			input: `name "test"`,
		},
		{
			name:  "Unclosed parentheses",
			input: `(name = "test"`,
		},
		{
			name:  "Invalid operator",
			input: `name === "test"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			if err == nil {
				t.Errorf("Parse() should have returned error for input: %s", tt.input)
			}
		})
	}
}

func TestParseEmptyInput(t *testing.T) {
	tests := []string{"", "   ", "\t", "\n"}

	for _, input := range tests {
		t.Run(fmt.Sprintf("empty_%q", input), func(t *testing.T) {
			expr, err := Parse(input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if expr != nil {
				t.Errorf("Parse() = %v, want nil for empty input", expr)
			}
		})
	}
}

func TestParseStringLiteralKeepsInnerQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "double-quoted value ending with single quote", input: `name = "5'"`, expected: `5'`},
		{name: "double-quoted value with quoted word at the end", input: `description = "Trained on 'ImageNet'"`, expected: `Trained on 'ImageNet'`},
		{name: "single-quoted value with quoted word at the end", input: `description = 'He said "hi"'`, expected: `He said "hi"`},
		{name: "escaped quote", input: `name = 'it\'s'`, expected: `it's`},
		{name: "empty string", input: `name = ""`, expected: ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if expr.Value != tt.expected {
				t.Errorf("Parse() value = %q, want %q", expr.Value, tt.expected)
			}
		})
	}
}
