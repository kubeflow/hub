package tls_test

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tlsconfig "github.com/kubeflow/hub/internal/platform/tls"
)

const (
	// Test certificate and key (self-signed, for testing only)
	// the following keys do not correspond to any system or live system
	// and has been generated with self-signature locally
	testCert = `-----BEGIN CERTIFICATE-----
MIIDczCCAlugAwIBAgIUMF77xY8/4njgitPJbFfCpfdDYvYwDQYJKoZIhvcNAQEL
BQAwSTELMAkGA1UEBhMCVVMxDTALBgNVBAgMBFRlc3QxDTALBgNVBAcMBFRlc3Qx
DTALBgNVBAoMBFRlc3QxDTALBgNVBAMMBHRlc3QwHhcNMjUwNjIwMTA1MTU0WhcN
MjYwNjIwMTA1MTU0WjBJMQswCQYDVQQGEwJVUzENMAsGA1UECAwEVGVzdDENMAsG
A1UEBwwEVGVzdDENMAsGA1UECgwEVGVzdDENMAsGA1UEAwwEdGVzdDCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBALNB2t9FATuWauWWZIpoHCC9fIfX/DxT
hxHpDA72dsJlP8aScOPzSDBlmZf6cWBEpZsaYRKdkCT3eqANhXg+ciid6nh3ZPqm
BJQx3AqY2YtfMQFjBljp/Glwyc21vgoP7v4Gk1ZUojhFwBfZJWWlW0mdzJshpYTB
quBYPqrD+5q23PVFgZqQSOGSUpiAERg93wTy7VeRxiws4FoKgxUCdbLJPw9KTDf4
PhQ+dnaWBYfWlBQxgEuB38vhMAn7RAinNxjoQDxfPethiBQU06xMqay0QNVueiAG
iR7e+QeQugIRj7yELpSgAGHxIibrcs6TPNGcI58+gAP/NfkI4hi272kCAwEAAaNT
MFEwHQYDVR0OBBYEFJ0eGqMERBgaN8GeggJj2yM+MnfyMB8GA1UdIwQYMBaAFJ0e
GqMERBgaN8GeggJj2yM+MnfyMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADggEBAGfcNw+HjYu/I6MhOgk1jgWov2yTXE2kjq097irdrHmlZcPXGWaMl+lw
rN4/hdFf0vhjSVWKjCjMP0dBAPCyht8xUg0/rsxYEa9UszFqYhtTT/k8HNyEXtSH
f2c5eSbc8n/IGvbiHPGmq/iHtZyDSKw83GQyRxSsXdgM7ru7N0YcP7qAkA6aqjYq
3/LbAvJOrPUMYlbpG7/+ZKareiMDy/1z8bWVFd3LiT12RhSJuqsCwMa3KePNAjaq
QTYxfer470vq8Y9cBwlMid0+RNGOcuQxbr20QeexEpGN8wHyxC4sO5ByBbpnPUsN
lcYEIoSHV3P8mTOqEVmCzBnDerbQfis=
-----END CERTIFICATE-----` // notsecret

	testKey = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCzQdrfRQE7lmrl
lmSKaBwgvXyH1/w8U4cR6QwO9nbCZT/GknDj80gwZZmX+nFgRKWbGmESnZAk93qg
DYV4PnIonep4d2T6pgSUMdwKmNmLXzEBYwZY6fxpcMnNtb4KD+7+BpNWVKI4RcAX
2SVlpVtJncybIaWEwargWD6qw/uattz1RYGakEjhklKYgBEYPd8E8u1XkcYsLOBa
CoMVAnWyyT8PSkw3+D4UPnZ2lgWH1pQUMYBLgd/L4TAJ+0QIpzcY6EA8Xz3rYYgU
FNOsTKmstEDVbnogBoke3vkHkLoCEY+8hC6UoABh8SIm63LOkzzRnCOfPoAD/zX5
COIYtu9pAgMBAAECggEAGuP36LLiELWLkyXYrr+X6pxqTDmNE+Km2ju2zJ7R6W8F
Xm005Kkn7OSs4hTWga3Clw4hvkhnKXh7i3uDyGo7t1abKBenDQevG6kQHIHZ7pOo
1w+rEdcF/65FA6guGjXSMQa8/wAitqTWAG3Zc5JW66fxm9r0CMKBtvZd7kGIoqhn
1b3CZUQyzgOqqB8jkyTYzhGxqqL/83L43O8dPtF3mRl3xHBDRMwuJtwaeik9ENJ5
mAZhWHiihE4icPXh7okrOKRhf4+2PtGVJVEcsCOoWn+u8Lrknr0pnWRw/RoX6qOB
NBcQKIhidl9XjbxLHGSZDGiKI8UuLqjLKfy+L5JARQKBgQDrb23cnmUSNFn28uDn
pEq9+Qpmyo0XZJsikcUmGodq/qIClDghNzeBfh1mO5vdgVd6yVkXprFIA6HeIvQs
10m2Y4ryKncrw5fIm3FsN66yjZPQBRq7Booh+h7JT10yYGO/Ew4Ec9q883PiNmjx
6JBjzKPSkd9W5fonXJFpThUBRQKBgQDC6jp14rTRKAx/JOoKJxcE7Rc28j3/7R0W
80GQMyFe6dhGcyQkjn5Fun45QU4+puycGEizWtFcgvWC5KSmwbtNlcGoPvZWy2Xw
lY4Lj/m4ziFSMz+q4GMdra1ymBlcFm3BdyqAJY+JpzihnioDNuJVIS5wM0iFRLcH
LsIv+9Dt1QKBgB43N9dXsMsMUvuBomG4USteefpFRqRY8hwWr0G7p+OQeIRyN13z
8zi4UdecEN31yp9klf2WFCyU4sJapBHZM4mn7t4zmwXP3XwOjxj/cHlT+EN7VDnq
lfHUYv0dJW3gtwx/yo3BvLIBYL8IkqFxYo6cZe4RcKN7coZ4t+TW85UtAoGAZoAG
fjfaHqOQ7svax7wGvvBvZNW/BPcMdSU3NT2uLtuKgIHMX+0POlv4ROOy4f+mLfAX
SzpXHu8/bLYQYCFA/mviizeRE9OiqAH90NbF3AmKPE/3C0U02kabD8gsjeC9lx+z
mfAmq5zkixlBvq7+FwZ8BUTyviKEnaJZPCKQnIECgYEAt26lsdwbUtistx9isZ8/
/NB4rd/GPQnBCyiQZy2mftYROuEonOHZq0XmuGhoU/Vvv0/9149eDCOED0BllBRy
qaXDpxpd8DoJVEvJDXAQVPu9WqcrYieJDc9a40tBMaf3YvcaCuHPj6LywzXlH45f
COUgUAV+r5AGr5R23tHluuQ=
-----END PRIVATE KEY-----` // notsecret

	testRootCA = `-----BEGIN CERTIFICATE-----
MIIDczCCAlugAwIBAgIUMF77xY8/4njgitPJbFfCpfdDYvYwDQYJKoZIhvcNAQEL
BQAwSTELMAkGA1UEBhMCVVMxDTALBgNVBAgMBFRlc3QxDTALBgNVBAcMBFRlc3Qx
DTALBgNVBAoMBFRlc3QxDTALBgNVBAMMBHRlc3QwHhcNMjUwNjIwMTA1MTU0WhcN
MjYwNjIwMTA1MTU0WjBJMQswCQYDVQQGEwJVUzENMAsGA1UECAwEVGVzdDENMAsG
A1UEBwwEVGVzdDENMAsGA1UECgwEVGVzdDENMAsGA1UEAwwEdGVzdDCCASIwDQYJ
KoZIhvcNAQEBBQADggEPADCCAQoCggEBALNB2t9FATuWauWWZIpoHCC9fIfX/DxT
hxHpDA72dsJlP8aScOPzSDBlmZf6cWBEpZsaYRKdkCT3eqANhXg+ciid6nh3ZPqm
BJQx3AqY2YtfMQFjBljp/Glwyc21vgoP7v4Gk1ZUojhFwBfZJWWlW0mdzJshpYTB
quBYPqrD+5q23PVFgZqQSOGSUpiAERg93wTy7VeRxiws4FoKgxUCdbLJPw9KTDf4
PhQ+dnaWBYfWlBQxgEuB38vhMAn7RAinNxjoQDxfPethiBQU06xMqay0QNVueiAG
iR7e+QeQugIRj7yELpSgAGHxIibrcs6TPNGcI58+gAP/NfkI4hi272kCAwEAAaNT
MFEwHQYDVR0OBBYEFJ0eGqMERBgaN8GeggJj2yM+MnfyMB8GA1UdIwQYMBaAFJ0e
GqMERBgaN8GeggJj2yM+MnfyMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEL
BQADggEBAGfcNw+HjYu/I6MhOgk1jgWov2yTXE2kjq097irdrHmlZcPXGWaMl+lw
rN4/hdFf0vhjSVWKjCjMP0dBAPCyht8xUg0/rsxYEa9UszFqYhtTT/k8HNyEXtSH
f2c5eSbc8n/IGvbiHPGmq/iHtZyDSKw83GQyRxSsXdgM7ru7N0YcP7qAkA6aqjYq
3/LbAvJOrPUMYlbpG7/+ZKareiMDy/1z8bWVFd3LiT12RhSJuqsCwMa3KePNAjaq
QTYxfer470vq8Y9cBwlMid0+RNGOcuQxbr20QeexEpGN8wHyxC4sO5ByBbpnPUsN
lcYEIoSHV3P8mTOqEVmCzBnDerbQfis=
-----END CERTIFICATE-----` // notsecret
)

func TestNewTLSConfig(t *testing.T) {
	tests := []struct {
		name             string
		certPath         string
		keyPath          string
		rootCertPath     string
		caPath           string
		cipher           string
		verifyServerCert bool
	}{
		{
			name:             "all parameters",
			certPath:         "/path/to/cert.pem",
			keyPath:          "/path/to/key.pem",
			rootCertPath:     "/path/to/root.pem",
			caPath:           "/path/to/ca.pem",
			cipher:           "TLS_AES_256_GCM_SHA384",
			verifyServerCert: true,
		},
		{
			name:             "minimal parameters",
			certPath:         "",
			keyPath:          "",
			rootCertPath:     "",
			caPath:           "",
			cipher:           "",
			verifyServerCert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tlsconfig.NewTLSConfig(tt.certPath, tt.keyPath, tt.rootCertPath, tt.caPath, tt.cipher, tt.verifyServerCert)

			if config.CertPath != tt.certPath {
				t.Errorf("CertPath = %v, want %v", config.CertPath, tt.certPath)
			}
			if config.KeyPath != tt.keyPath {
				t.Errorf("KeyPath = %v, want %v", config.KeyPath, tt.keyPath)
			}
			if config.RootCertPath != tt.rootCertPath {
				t.Errorf("RootCertPath = %v, want %v", config.RootCertPath, tt.rootCertPath)
			}
			if config.CAPath != tt.caPath {
				t.Errorf("CAPath = %v, want %v", config.CAPath, tt.caPath)
			}
			if config.Cipher != tt.cipher {
				t.Errorf("Cipher = %v, want %v", config.Cipher, tt.cipher)
			}
			if config.VerifyServerCert != tt.verifyServerCert {
				t.Errorf("VerifyServerCert = %v, want %v", config.VerifyServerCert, tt.verifyServerCert)
			}
		})
	}
}

func TestBuildTLSConfig_Basic(t *testing.T) {
	// Test basic configuration without certificates
	config := tlsconfig.NewTLSConfig("", "", "", "", "", false)
	tlsConf, err := config.BuildTLSConfig()

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
		return
	}

	if tlsConf == nil {
		t.Fatal("BuildTLSConfig() returned nil config")
		return
	}

	if !tlsConf.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true when VerifyServerCert is false")
	}

	if tlsConf.RootCAs != nil {
		t.Error("Expected RootCAs to be nil when no cert paths provided and VerifyServerCert is false")
	}
}

func TestBuildTLSConfig_VerifyServerCert(t *testing.T) {
	// Test with server cert verification enabled
	config := tlsconfig.NewTLSConfig("", "", "", "", "", true)
	tlsConf, err := config.BuildTLSConfig()

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}

	if tlsConf.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false when VerifyServerCert is true")
	}

	// Should use system cert pool when VerifyServerCert is true but no custom certs provided
	if tlsConf.RootCAs == nil {
		t.Error("Expected RootCAs to be set when VerifyServerCert is true")
	}
}

func TestBuildTLSConfig_WithCertificates(t *testing.T) {
	// Create temporary files for testing
	tempDir := t.TempDir()

	certFile := filepath.Join(tempDir, "cert.pem")
	keyFile := filepath.Join(tempDir, "key.pem")
	rootCAFile := filepath.Join(tempDir, "rootca.pem")

	// Write test certificates
	if err := os.WriteFile(certFile, []byte(testCert), 0644); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte(testKey), 0644); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	if err := os.WriteFile(rootCAFile, []byte(testRootCA), 0644); err != nil {
		t.Fatalf("Failed to write root CA file: %v", err)
	}

	config := tlsconfig.NewTLSConfig(certFile, keyFile, rootCAFile, "", "", true)
	tlsConf, err := config.BuildTLSConfig()

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}

	if len(tlsConf.Certificates) != 1 {
		t.Errorf("Expected 1 certificate, got %d", len(tlsConf.Certificates))
	}

	if tlsConf.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}

	if tlsConf.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be false when VerifyServerCert is true")
	}
}

func TestBuildTLSConfig_WithCiphers(t *testing.T) {
	config := tlsconfig.NewTLSConfig("", "", "", "", "TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256", false)
	tlsConf, err := config.BuildTLSConfig()

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}

	expectedCiphers := []uint16{tls.TLS_AES_256_GCM_SHA384, tls.TLS_AES_128_GCM_SHA256}
	if len(tlsConf.CipherSuites) != len(expectedCiphers) {
		t.Errorf("Expected %d cipher suites, got %d", len(expectedCiphers), len(tlsConf.CipherSuites))
	}

	for i, expected := range expectedCiphers {
		if i >= len(tlsConf.CipherSuites) || tlsConf.CipherSuites[i] != expected {
			t.Errorf("Expected cipher suite %x at index %d, got %x", expected, i, tlsConf.CipherSuites[i])
		}
	}
}

func TestBuildTLSConfig_ErrorCases(t *testing.T) {
	tests := []struct {
		name         string
		setupConfig  func(t *testing.T) *tlsconfig.TLSConfig
		wantErrorMsg string
	}{
		{
			name: "invalid cert file",
			setupConfig: func(t *testing.T) *tlsconfig.TLSConfig {
				return tlsconfig.NewTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem", "", "", "", false)
			},
			wantErrorMsg: "failed to load SSL certificate pair",
		},
		{
			name: "invalid root cert file",
			setupConfig: func(t *testing.T) *tlsconfig.TLSConfig {
				return tlsconfig.NewTLSConfig("", "", "/nonexistent/root.pem", "", "", false)
			},
			wantErrorMsg: "failed to read SSL root certificate",
		},
		{
			name: "invalid CA cert file",
			setupConfig: func(t *testing.T) *tlsconfig.TLSConfig {
				return tlsconfig.NewTLSConfig("", "", "", "/nonexistent/ca.pem", "", false)
			},
			wantErrorMsg: "failed to read SSL CA certificate",
		},
		{
			name: "invalid cert content",
			setupConfig: func(t *testing.T) *tlsconfig.TLSConfig {
				tempDir := t.TempDir()
				invalidCertFile := filepath.Join(tempDir, "invalid.pem")
				if err := os.WriteFile(invalidCertFile, []byte("invalid certificate content"), 0644); err != nil {
					t.Fatalf("Failed to write invalid cert file: %v", err)
				}
				return tlsconfig.NewTLSConfig("", "", invalidCertFile, "", "", false)
			},
			wantErrorMsg: "failed to parse SSL root certificate",
		},
		{
			name: "invalid cipher suites",
			setupConfig: func(t *testing.T) *tlsconfig.TLSConfig {
				return tlsconfig.NewTLSConfig("", "", "", "", "INVALID_CIPHER_SUITE:ANOTHER_INVALID_CIPHER", false)
			},
			wantErrorMsg: "invalid cipher suite",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.setupConfig(t)
			_, err := config.BuildTLSConfig()

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrorMsg) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErrorMsg, err.Error())
			}
		})
	}
}

func TestParseCipherSuites(t *testing.T) {
	tests := []struct {
		name        string
		cipherStr   string
		expectedLen int
		expected    []uint16
	}{
		{
			name:        "empty string",
			cipherStr:   "",
			expectedLen: 0,
			expected:    nil,
		},
		{
			name:        "single cipher",
			cipherStr:   "TLS_AES_256_GCM_SHA384",
			expectedLen: 1,
			expected:    []uint16{tls.TLS_AES_256_GCM_SHA384},
		},
		{
			name:        "multiple ciphers",
			cipherStr:   "TLS_AES_256_GCM_SHA384:TLS_AES_128_GCM_SHA256",
			expectedLen: 2,
			expected:    []uint16{tls.TLS_AES_256_GCM_SHA384, tls.TLS_AES_128_GCM_SHA256},
		},
		{
			name:        "cipher with spaces",
			cipherStr:   " TLS_AES_256_GCM_SHA384 : TLS_AES_128_GCM_SHA256 ",
			expectedLen: 2,
			expected:    []uint16{tls.TLS_AES_256_GCM_SHA384, tls.TLS_AES_128_GCM_SHA256},
		},
		{
			name:        "mixed valid and invalid ciphers should error",
			cipherStr:   "TLS_AES_256_GCM_SHA384:UNKNOWN_CIPHER:TLS_AES_128_GCM_SHA256",
			expectedLen: -1, // Should error on first invalid cipher
			expected:    nil,
		},
		{
			name:        "all unknown ciphers should error",
			cipherStr:   "UNKNOWN_CIPHER1:UNKNOWN_CIPHER2",
			expectedLen: -1, // Special value to indicate we expect an error
			expected:    nil,
		},
		{
			name:        "comprehensive cipher list",
			cipherStr:   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:TLS_CHACHA20_POLY1305_SHA256",
			expectedLen: 2,
			expected: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use reflection to access the unexported function
			// Since we can't access unexported functions directly, we'll test through BuildTLSConfig
			config := tlsconfig.NewTLSConfig("", "", "", "", tt.cipherStr, false)
			tlsConf, err := config.BuildTLSConfig()

			if tt.expectedLen == -1 {
				// We expect an error for invalid ciphers
				if err == nil {
					t.Errorf("Expected error for invalid cipher suite, got nil")
					return
				}
				if !strings.Contains(err.Error(), "invalid cipher suite") {
					t.Errorf("Expected error to contain 'invalid cipher suite', got: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("BuildTLSConfig() error = %v", err)
			}

			if len(tlsConf.CipherSuites) != tt.expectedLen {
				t.Errorf("Expected %d cipher suites, got %d", tt.expectedLen, len(tlsConf.CipherSuites))
			}

			if tt.expected != nil {
				for i, expected := range tt.expected {
					if i >= len(tlsConf.CipherSuites) || tlsConf.CipherSuites[i] != expected {
						t.Errorf("Expected cipher suite %x at index %d, got %x", expected, i, tlsConf.CipherSuites[i])
					}
				}
			}
		})
	}
}

func TestBuildTLSConfig_SeparateCAAndRootCert(t *testing.T) {
	tempDir := t.TempDir()

	rootCAFile := filepath.Join(tempDir, "rootca.pem")
	caFile := filepath.Join(tempDir, "ca.pem")

	// Write different certificates to test both are loaded
	if err := os.WriteFile(rootCAFile, []byte(testRootCA), 0644); err != nil {
		t.Fatalf("Failed to write root CA file: %v", err)
	}
	if err := os.WriteFile(caFile, []byte(testCert), 0644); err != nil {
		t.Fatalf("Failed to write CA file: %v", err)
	}

	config := tlsconfig.NewTLSConfig("", "", rootCAFile, caFile, "", true)
	tlsConf, err := config.BuildTLSConfig()

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}

	if tlsConf.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}

	// Verify both certificates were added to the pool
	// We can't directly inspect the cert pool contents, but we can verify it was created
	if tlsConf.RootCAs == nil {
		t.Error("Expected certificate pool to contain both root CA and CA certificates")
	}
}

func TestBuildTLSConfig_SameCAAndRootCert(t *testing.T) {
	tempDir := t.TempDir()

	rootCAFile := filepath.Join(tempDir, "rootca.pem")

	if err := os.WriteFile(rootCAFile, []byte(testRootCA), 0644); err != nil {
		t.Fatalf("Failed to write root CA file: %v", err)
	}

	// Use same file for both root cert and CA path
	config := tlsconfig.NewTLSConfig("", "", rootCAFile, rootCAFile, "", true)
	tlsConf, err := config.BuildTLSConfig()

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}

	if tlsConf.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}
}

// TestBuildTLSConfig_RootCAWithoutVerifyServerCert checks that a configured root
// CA is enforced when VerifyServerCert is false: the server certificate must
// chain to the CA, but its host name is not checked.
func TestBuildTLSConfig_RootCAWithoutVerifyServerCert(t *testing.T) {
	ca := newTestCert(t, "test-ca", nil)
	signed := newTestCert(t, "db.example.com", &ca)
	untrusted := newTestCert(t, "db.example.com", nil)

	rootCAFile := filepath.Join(t.TempDir(), "rootca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Leaf.Raw})
	if err := os.WriteFile(rootCAFile, caPEM, 0644); err != nil {
		t.Fatalf("Failed to write root CA file: %v", err)
	}

	tlsConf, err := tlsconfig.NewTLSConfig("", "", rootCAFile, "", "", false).BuildTLSConfig()
	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}

	// The client dials 127.0.0.1, which is not in either certificate.
	if err := tlsHandshake(t, tlsConf, signed); err != nil {
		t.Errorf("Expected handshake to succeed with a server certificate signed by the root CA, got: %v", err)
	}
	if err := tlsHandshake(t, tlsConf, untrusted); err == nil {
		t.Error("Expected handshake to fail with a server certificate not signed by the root CA")
	}
}

// newTestCert returns a certificate for cn signed by parent, or self-signed
// when parent is nil.
func newTestCert(t *testing.T, cn string, parent *tls.Certificate) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: cn},
		DNSNames:              []string{cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  parent == nil,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	issuer, signer := template, crypto.Signer(key)
	if parent != nil {
		issuer, signer = parent.Leaf, parent.PrivateKey.(crypto.Signer)
	}
	der, err := x509.CreateCertificate(rand.Reader, template, issuer, &key.PublicKey, signer)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}
}

// tlsHandshake runs a TLS handshake between clientConf and a local server
// presenting serverCert, and returns the client's error.
func tlsHandshake(t *testing.T, clientConf *tls.Config, serverCert tls.Certificate) error {
	t.Helper()

	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{serverCert}})
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_ = conn.(*tls.Conn).Handshake()
	}()

	conn, err := tls.Dial("tcp", ln.Addr().String(), clientConf)
	if err != nil {
		return err
	}
	return conn.Close()
}
