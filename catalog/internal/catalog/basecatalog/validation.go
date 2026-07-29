package basecatalog

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// ValidArtifactURISchemes defines the valid URI schemes for model artifacts.
var ValidArtifactURISchemes = []string{
	"oci",
	"http",
	"https",
	"s3",
	"gs",
	"az",
	"file",
}

// ValidateArtifactURI validates that an artifact URI has a valid scheme.
// Returns an error if the URI is empty, malformed, or has an unsupported scheme.
func ValidateArtifactURI(uri string) error {
	if uri == "" {
		return fmt.Errorf("artifact URI cannot be empty")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("invalid artifact URI %q: %w", uri, err)
	}

	if parsed.Scheme == "" {
		return fmt.Errorf("artifact URI %q must have a scheme (valid schemes: %v)", uri, ValidArtifactURISchemes)
	}

	if !slices.Contains(ValidArtifactURISchemes, parsed.Scheme) {
		return fmt.Errorf("artifact URI %q has unsupported scheme %q (valid schemes: %v)", uri, parsed.Scheme, ValidArtifactURISchemes)
	}

	return nil
}

// supportedOperators defines the valid operators for named query field filters
var supportedOperators = map[string]bool{
	"=":      true,
	"!=":     true,
	">":      true,
	"<":      true,
	">=":     true,
	"<=":     true,
	"LIKE":   true,
	"ILIKE":  true,
	"IN":     true,
	"NOT IN": true,
}

// ValidateNamedQueries validates the structure and content of named queries
func ValidateNamedQueries(namedQueries map[string]NamedQuery) error {
	validAssetTypes := map[string]bool{
		AssetTypeModels:     true,
		AssetTypeMCPServers: true,
		AssetTypeAgents:     true,
		AssetTypeSkills:     true,
	}

	for queryName, nq := range namedQueries {
		if queryName == "" {
			return fmt.Errorf("named query name cannot be empty")
		}

		if nq.AssetType != "" && !validAssetTypes[nq.AssetType] {
			return fmt.Errorf("named query '%s' has invalid assetType '%s' (valid values: %s, %s, %s, %s)", queryName, nq.AssetType, AssetTypeModels, AssetTypeMCPServers, AssetTypeAgents, AssetTypeSkills)
		}

		if len(nq.Filters) == 0 {
			return fmt.Errorf("named query '%s' must contain at least one field filter", queryName)
		}

		for fieldName, filter := range nq.Filters {
			if fieldName == "" {
				return fmt.Errorf("field name cannot be empty in named query '%s'", queryName)
			}

			if err := validateFieldFilter(queryName, fieldName, filter); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateFieldFilter validates a single field filter within a named query
func validateFieldFilter(queryName, fieldName string, filter FieldFilter) error {
	if filter.Operator == "" {
		return fmt.Errorf("operator cannot be empty for field '%s' in named query '%s'", fieldName, queryName)
	}

	normalizedOperator := strings.ToUpper(filter.Operator)
	if !supportedOperators[normalizedOperator] {
		return fmt.Errorf("unsupported operator '%s' for field '%s' in named query '%s'", filter.Operator, fieldName, queryName)
	}

	if filter.Value == nil {
		return fmt.Errorf("value cannot be nil for field '%s' in named query '%s'", fieldName, queryName)
	}

	// Additional validation based on operator type
	switch normalizedOperator {
	case "IN", "NOT IN":
		// Value should be an array
		if _, ok := filter.Value.([]any); !ok {
			return fmt.Errorf("operator '%s' requires array value for field '%s' in named query '%s'", filter.Operator, fieldName, queryName)
		}
	}

	return nil
}
