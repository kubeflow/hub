package repositories

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestHfSourceLabelValue(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
		// validLabel is false only for the documented degenerate case (an ID
		// that normalizes to all underscores), where hfSourceLabelValue falls
		// back to a value that is not a valid Kubernetes label value.
		validLabel bool
	}{
		{"lowercase with underscore", "my_source", "MY_SOURCE", true},
		{"already valid", "abc_123", "ABC_123", true},
		{"digits", "abc123", "ABC123", true},
		{"leading underscore", "_test", "TEST", true},
		{"trailing underscore", "my_source_", "MY_SOURCE", true},
		{"leading and trailing underscores", "_my_source_", "MY_SOURCE", true},
		{"all underscores", "___", "___", false},
		{"longer than 63 chars", strings.Repeat("a", 70), strings.Repeat("A", 63), true},
		{"truncation lands on underscore", strings.Repeat("a", 62) + "_bbb", strings.Repeat("A", 62), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hfSourceLabelValue(tt.id)
			assert.Equal(t, tt.want, got)
			errs := validation.IsValidLabelValue(got)
			if tt.validLabel {
				assert.Empty(t, errs, "expected %q to be a valid label value", got)
			} else {
				assert.NotEmpty(t, errs, "expected %q to be an invalid label value", got)
			}
		})
	}
}

func TestValidateCatalogId(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{"valid with underscore", "my_source", nil},
		{"valid alphanumeric with underscore", "abc_123", nil},
		{"empty", "", ErrCatalogSourceIdRequired},
		{"uppercase rejected by regex", "MySource", nil},
		{"hyphen rejected by regex", "my-source", nil},
		{"all underscores", "___", ErrCatalogIdInvalid},
		{"single underscore", "_", ErrCatalogIdInvalid},
		{"too long", strings.Repeat("a", 239), ErrCatalogIDTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCatalogId(tt.id)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else if tt.name == "uppercase rejected by regex" || tt.name == "hyphen rejected by regex" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
