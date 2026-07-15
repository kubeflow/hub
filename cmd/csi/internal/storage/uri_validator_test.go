package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestURIValidatorValidate(t *testing.T) {
	tests := []struct {
		name            string
		allowedPrefixes []string
		artifactURI     string
		expectError     bool
		expectedErr     error
	}{
		// Blocklist: cloud metadata endpoints blocked regardless of allowlist
		{
			name:        "block AWS metadata endpoint",
			artifactURI: "http://169.254.169.254/latest/meta-data/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block AWS metadata IMDSv2",
			artifactURI: "https://169.254.169.254/latest/api/token",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block AWS IPv6 metadata",
			artifactURI: "http://[fd00:ec2::254]/latest/meta-data/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block GCP metadata endpoint",
			artifactURI: "http://metadata.google.internal/computeMetadata/v1/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block metadata with non-standard port",
			artifactURI: "http://169.254.169.254:8080/something",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block metadata IP in s3 URI",
			artifactURI: "s3://169.254.169.254/bucket/key",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block AWS IPv6 metadata with port",
			artifactURI: "http://[fd00:ec2::254]:8080/latest/meta-data/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block link-local address in range",
			artifactURI: "http://169.254.42.42/exfiltrate",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block GCP metadata.goog endpoint",
			artifactURI: "http://metadata.goog/computeMetadata/v1/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block trailing dot hostname bypass",
			artifactURI: "http://metadata.google.internal./computeMetadata/v1/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block trailing dot metadata.goog bypass",
			artifactURI: "http://metadata.goog./computeMetadata/v1/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:        "block uppercase hostname bypass",
			artifactURI: "http://METADATA.GOOGLE.INTERNAL/computeMetadata/v1/",
			expectError: true,
			expectedErr: ErrArtifactURIBlocked,
		},
		{
			name:            "blocklist overrides allowlist",
			allowedPrefixes: []string{"http://169.254.169.254/"},
			artifactURI:     "http://169.254.169.254/latest/",
			expectError:     true,
			expectedErr:     ErrArtifactURIBlocked,
		},

		// Empty allowlist: backward compatibility, all non-blocked URIs accepted
		{
			name:        "empty allowlist accepts s3",
			artifactURI: "s3://any-bucket/model.tar.gz",
			expectError: false,
		},
		{
			name:        "empty allowlist accepts gs",
			artifactURI: "gs://any-bucket/model.tar.gz",
			expectError: false,
		},
		{
			name:        "empty allowlist accepts https",
			artifactURI: "https://example.com/models/model.tar.gz",
			expectError: false,
		},
		{
			name:        "empty allowlist accepts http",
			artifactURI: "http://example.com/models/model.tar.gz",
			expectError: false,
		},
		{
			name:            "nil allowlist accepts all",
			allowedPrefixes: nil,
			artifactURI:     "s3://any-bucket/any-path",
			expectError:     false,
		},

		// Allowlist matching: only matching prefixes accepted
		{
			name:            "allowlist match with prefix",
			allowedPrefixes: []string{"s3://my-bucket/"},
			artifactURI:     "s3://my-bucket/models/v1/model.tar.gz",
			expectError:     false,
		},
		{
			name:            "allowlist exact match",
			allowedPrefixes: []string{"s3://my-bucket/"},
			artifactURI:     "s3://my-bucket/",
			expectError:     false,
		},
		{
			name:            "allowlist rejects non-matching bucket",
			allowedPrefixes: []string{"s3://my-bucket/"},
			artifactURI:     "s3://evil-bucket/payload",
			expectError:     true,
			expectedErr:     ErrArtifactURINotAllowed,
		},
		{
			name:            "allowlist rejects non-matching protocol",
			allowedPrefixes: []string{"s3://my-bucket/"},
			artifactURI:     "https://example.com/model.tar.gz",
			expectError:     true,
			expectedErr:     ErrArtifactURINotAllowed,
		},
		{
			name:            "multiple allowlist entries",
			allowedPrefixes: []string{"s3://my-bucket/", "gs://trusted/"},
			artifactURI:     "gs://trusted/model.bin",
			expectError:     false,
		},
		{
			name:            "multiple allowlist entries rejects non-matching",
			allowedPrefixes: []string{"s3://my-bucket/", "gs://trusted/"},
			artifactURI:     "gs://other-bucket/model.bin",
			expectError:     true,
			expectedErr:     ErrArtifactURINotAllowed,
		},

		// Protocol-only allowlist entries
		{
			name:            "protocol-only allowlist accepts matching protocol",
			allowedPrefixes: []string{"s3://"},
			artifactURI:     "s3://any-bucket/any-path",
			expectError:     false,
		},
		{
			name:            "protocol-only allowlist rejects different protocol",
			allowedPrefixes: []string{"s3://"},
			artifactURI:     "gs://any-bucket/any-path",
			expectError:     true,
			expectedErr:     ErrArtifactURINotAllowed,
		},

		// Empty prefix must not bypass allowlist
		{
			name:            "empty prefix in allowlist does not match everything",
			allowedPrefixes: []string{"", "s3://my-bucket/"},
			artifactURI:     "gs://evil-bucket/payload",
			expectError:     true,
			expectedErr:     ErrArtifactURINotAllowed,
		},

		// Trailing slash subtlety
		{
			name:            "no trailing slash matches similar bucket names",
			allowedPrefixes: []string{"s3://my-bucket"},
			artifactURI:     "s3://my-bucket-evil/payload",
			expectError:     false,
		},
		{
			name:            "trailing slash prevents similar bucket name match",
			allowedPrefixes: []string{"s3://my-bucket/"},
			artifactURI:     "s3://my-bucket-evil/payload",
			expectError:     true,
			expectedErr:     ErrArtifactURINotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewURIValidator(tt.allowedPrefixes)
			err := validator.Validate(tt.artifactURI)
			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedErr != nil {
					assert.ErrorIs(t, err, tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsBlockedHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		blocked bool
	}{
		{"AWS metadata IP", "169.254.169.254", true},
		{"AWS metadata with port", "169.254.169.254:8080", true},
		{"AWS IPv6 metadata", "[fd00:ec2::254]", true},
		{"AWS IPv6 metadata unbracketed", "fd00:ec2::254", true},
		{"AWS IPv6 metadata with port", "[fd00:ec2::254]:8080", true},
		{"GCP metadata hostname", "metadata.google.internal", true},
		{"GCP metadata.goog hostname", "metadata.goog", true},
		{"GCP metadata trailing dot", "metadata.google.internal.", true},
		{"GCP metadata.goog trailing dot", "metadata.goog.", true},
		{"uppercase hostname", "METADATA.GOOGLE.INTERNAL", true},
		{"mixed case hostname", "Metadata.Google.Internal", true},
		{"link-local address", "169.254.1.1", true},
		{"normal S3 host", "my-bucket", false},
		{"normal hostname", "example.com", false},
		{"normal IP", "10.0.0.1", false},
		{"localhost", "127.0.0.1", false},
		{"empty host", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.blocked, isBlockedHost(tt.host))
		})
	}
}
