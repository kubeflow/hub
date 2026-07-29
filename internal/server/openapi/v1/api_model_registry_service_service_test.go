package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCombinedFilterQuery(t *testing.T) {
	testCases := []struct {
		name        string
		filterQuery string
		nameParam   string
		externalID  string
		expected    string
	}{
		{
			name:        "empty parameters",
			filterQuery: "",
			nameParam:   "",
			externalID:  "",
			expected:    "",
		},
		{
			name:        "only filter query",
			filterQuery: "state = 'LIVE'",
			nameParam:   "",
			externalID:  "",
			expected:    "(state = 'LIVE')",
		},
		{
			name:        "only name parameter",
			filterQuery: "",
			nameParam:   "my-artifact",
			externalID:  "",
			expected:    "name = 'my-artifact'",
		},
		{
			name:        "only externalID parameter",
			filterQuery: "",
			nameParam:   "",
			externalID:  "ext-123",
			expected:    "externalId = 'ext-123'",
		},
		{
			name:        "name takes precedence over externalID",
			filterQuery: "",
			nameParam:   "my-artifact",
			externalID:  "ext-123",
			expected:    "name = 'my-artifact'",
		},
		{
			name:        "filter query with name",
			filterQuery: "state = 'LIVE'",
			nameParam:   "my-artifact",
			externalID:  "",
			expected:    "(state = 'LIVE') AND name = 'my-artifact'",
		},
		{
			name:        "filter query with externalID",
			filterQuery: "state = 'LIVE'",
			nameParam:   "",
			externalID:  "ext-123",
			expected:    "(state = 'LIVE') AND externalId = 'ext-123'",
		},
		{
			name:        "single quotes escaped in name",
			filterQuery: "",
			nameParam:   "my-artifact's-name",
			externalID:  "",
			expected:    "name = 'my-artifact''s-name'",
		},
		{
			name:        "single quotes escaped in externalID",
			filterQuery: "",
			nameParam:   "",
			externalID:  "ext-'123'",
			expected:    "externalId = 'ext-''123'''",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := buildCombinedFilterQuery(tc.filterQuery, tc.nameParam, tc.externalID)
			assert.Equal(t, tc.expected, result)
		})
	}
}
