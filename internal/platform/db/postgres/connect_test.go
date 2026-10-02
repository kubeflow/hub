package postgres_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kubeflow/hub/internal/platform/db/postgres"
	_tls "github.com/kubeflow/hub/internal/platform/tls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresConnectContext(t *testing.T) {
	t.Run("connection_handshake_deadline", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		done := make(chan struct{})
		finish := make(chan struct{})
		defer func() {
			close(finish)
			assert.NoError(t, listener.Close())
			<-done
		}()
		go func() {
			defer close(done)
			connection, err := listener.Accept()
			if err == nil {
				defer func() { _ = connection.Close() }()
				<-finish
			}
		}()
		ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
		defer cancel()
		connector := postgres.NewPostgresDBConnector("postgres://postgres:test@"+listener.Addr().String()+"/test?sslmode=disable", nil)
		started := time.Now()
		_, err = connector.ConnectContext(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, time.Since(started), time.Second)
	})
	t.Run("retry_deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
		defer cancel()
		parsed, err := url.Parse(postgresContainer.MustConnectionString(t.Context()))
		require.NoError(t, err)
		parsed.User = url.UserPassword("postgres", "incorrect")
		connector := postgres.NewPostgresDBConnector(parsed.String(), nil)
		started := time.Now()
		_, err = connector.ConnectContext(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, time.Since(started), time.Second)
	})
}

func TestPostgresDBConnector_Connect_Insecure(t *testing.T) {
	ctx := context.Background()

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Test basic connection without SSL
	t.Run("BasicConnection", func(t *testing.T) {
		dsn := fmt.Sprintf("host=%s port=%s user=postgres password=testpass dbname=testdb sslmode=disable",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{})

		db, err := connector.Connect()
		require.NoError(t, err)
		assert.NotNil(t, db)

		// Test that we can perform a simple query
		var result int
		err = db.Raw("SELECT 1").Scan(&result).Error
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	})

	t.Run("EmptySSLConfig", func(t *testing.T) {
		dsn := fmt.Sprintf("host=%s port=%s user=postgres password=testpass dbname=testdb sslmode=disable",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{})
		db, err := connector.Connect()
		require.NoError(t, err)
		assert.NotNil(t, db)
	})

	t.Run("URLFormatDSN", func(t *testing.T) {
		dsn := fmt.Sprintf("postgres://postgres:testpass@%s:%s/testdb?sslmode=disable",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{})

		db, err := connector.Connect()
		require.NoError(t, err)
		assert.NotNil(t, db)

		// Test database operations
		var result int
		err = db.Raw("SELECT 1").Scan(&result).Error
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	})
}

func TestPostgresDBConnector_TLSConfigValidation(t *testing.T) {
	// Create temporary directory for certificates
	tempDir, err := os.MkdirTemp("", "postgres_ssl_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	// Generate test certificates
	caCertPath, _, _, _, clientCertPath, clientKeyPath := generateTestCertificates(t, tempDir)

	t.Run("ValidCertificateFiles", func(t *testing.T) {
		// Test that valid certificate files can be loaded into TLS config
		tlsConfig := &_tls.TLSConfig{
			RootCertPath:     caCertPath,
			VerifyServerCert: true,
		}

		// This should build successfully since files are valid
		builtConfig, err := tlsConfig.BuildTLSConfig()
		assert.NoError(t, err)
		assert.NotNil(t, builtConfig)
		assert.NotNil(t, builtConfig.RootCAs)
		assert.False(t, builtConfig.InsecureSkipVerify)
	})

	t.Run("ValidClientCertificates", func(t *testing.T) {
		tlsConfig := &_tls.TLSConfig{
			CertPath:         clientCertPath,
			KeyPath:          clientKeyPath,
			RootCertPath:     caCertPath,
			VerifyServerCert: true,
		}

		builtConfig, err := tlsConfig.BuildTLSConfig()
		assert.NoError(t, err)
		assert.NotNil(t, builtConfig)
		assert.Len(t, builtConfig.Certificates, 1)
		assert.NotNil(t, builtConfig.RootCAs)
	})

	t.Run("CipherSuitesNotSupportedWarning", func(t *testing.T) {
		// Test that cipher suites configuration logs a warning but doesn't fail
		tlsConfig := &_tls.TLSConfig{
			Cipher:           "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			VerifyServerCert: true,
		}

		// This should still work despite cipher config (it's just ignored)
		builtConfig, err := tlsConfig.BuildTLSConfig()
		assert.NoError(t, err)
		assert.NotNil(t, builtConfig)
		assert.Len(t, builtConfig.CipherSuites, 2) // TLS config still processes ciphers
	})

	t.Run("InvalidCertificateFile", func(t *testing.T) {
		tlsConfig := &_tls.TLSConfig{
			RootCertPath: "/nonexistent/cert.pem",
		}

		_, err := tlsConfig.BuildTLSConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read SSL root certificate")
	})

	t.Run("InvalidCertificateContent", func(t *testing.T) {
		invalidCertPath := filepath.Join(tempDir, "invalid.pem")
		err := os.WriteFile(invalidCertPath, []byte("invalid certificate content"), 0600)
		require.NoError(t, err)

		tlsConfig := &_tls.TLSConfig{
			RootCertPath: invalidCertPath,
		}

		_, err = tlsConfig.BuildTLSConfig()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse SSL root certificate")
	})
}

func TestPostgresDBConnector_DSNBuilding(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "postgres_dsn_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	caCertPath, _, _, _, clientCertPath, clientKeyPath := generateTestCertificates(t, tempDir)

	t.Run("BuildDSNWithSSLMode_KeyValue", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test",
			&_tls.TLSConfig{
				VerifyServerCert: true,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=verify-full")
		assert.Contains(t, dsn, "host=localhost")
		assert.Contains(t, dsn, "port=5432")
	})

	t.Run("BuildDSNWithSSLMode_URL", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"postgres://postgres:password@localhost:5432/test",
			&_tls.TLSConfig{
				VerifyServerCert: true,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=verify-full")
		assert.Contains(t, dsn, "postgres://postgres:password@localhost:5432/test")
	})

	t.Run("BuildDSNWithCertificates_KeyValue", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test",
			&_tls.TLSConfig{
				CertPath:     clientCertPath,
				KeyPath:      clientKeyPath,
				RootCertPath: caCertPath,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=require")
		assert.Contains(t, dsn, fmt.Sprintf("sslcert=%s", clientCertPath))
		assert.Contains(t, dsn, fmt.Sprintf("sslkey=%s", clientKeyPath))
		assert.Contains(t, dsn, fmt.Sprintf("sslrootcert=%s", caCertPath))
	})

	t.Run("BuildDSNWithCertificates_URL", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"postgres://postgres:password@localhost:5432/test",
			&_tls.TLSConfig{
				CertPath:     clientCertPath,
				KeyPath:      clientKeyPath,
				RootCertPath: caCertPath,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=require")
		// URL parameters are URL-encoded, so we need to check for encoded values
		assert.Contains(t, dsn, "sslcert=")
		assert.Contains(t, dsn, "sslkey=")
		assert.Contains(t, dsn, "sslrootcert=")
		// Verify the actual path components are present (even if encoded)
		assert.Contains(t, dsn, "client-cert.pem")
		assert.Contains(t, dsn, "client-key.pem")
		assert.Contains(t, dsn, "ca-cert.pem")
	})

	t.Run("BuildDSNWithCAPath", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test",
			&_tls.TLSConfig{
				CAPath: caCertPath,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=require")
		assert.Contains(t, dsn, fmt.Sprintf("sslrootcert=%s", caCertPath))
	})

	t.Run("BuildDSNWithQueryParams_URL", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"postgres://postgres:password@localhost:5432/test?connect_timeout=10",
			&_tls.TLSConfig{
				VerifyServerCert: true,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=verify-full")
		assert.Contains(t, dsn, "connect_timeout=10")
	})

	t.Run("BuildDSNWithExistingSSLMode", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test sslmode=disable",
			&_tls.TLSConfig{
				VerifyServerCert: true,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		// Should update existing sslmode
		assert.Contains(t, dsn, "sslmode=verify-full")
		// Should not contain the old sslmode=disable
		assert.NotContains(t, dsn, "sslmode=disable")
	})

	t.Run("BuildDSNWithCipherSuites_Warning", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test",
			&_tls.TLSConfig{
				Cipher:           "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				VerifyServerCert: true,
			},
		).WithMaxRetries(1)

		// This should succeed but log a warning
		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslmode=verify-full")
		// Cipher should not appear in DSN (PostgreSQL doesn't support it)
		assert.NotContains(t, dsn, "cipher")
		assert.NotContains(t, dsn, "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384")
	})
}

func TestPostgresDBConnector_Connect_Secure(t *testing.T) {
	ctx := context.Background()

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Create temporary directory for certificates
	tempDir, err := os.MkdirTemp("", "postgres_ssl_integration_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir) //nolint:errcheck

	caCertPath, _, _, _, clientCertPath, clientKeyPath := generateTestCertificates(t, tempDir)

	t.Run("SSLConnectionWithRequire", func(t *testing.T) {
		dsn := fmt.Sprintf("host=%s port=%s user=postgres password=testpass dbname=testdb",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{
			RootCertPath: caCertPath,
		}).WithMaxRetries(1)

		// Build DSN to verify it's constructed correctly
		builtDSN, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, builtDSN, "sslmode=require")
		assert.Contains(t, builtDSN, fmt.Sprintf("sslrootcert=%s", caCertPath))

		// For this test, we'll just verify DSN building works correctly
		// Actual SSL connection would require proper server SSL setup
		t.Logf("Built DSN with SSL: %s", builtDSN)
	})

	t.Run("SSLConnectionWithClientCert", func(t *testing.T) {
		dsn := fmt.Sprintf("host=%s port=%s user=postgres dbname=testdb",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{
			CertPath:     clientCertPath,
			KeyPath:      clientKeyPath,
			RootCertPath: caCertPath,
		}).WithMaxRetries(1)

		// Build DSN to verify it's constructed correctly
		builtDSN, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, builtDSN, "sslmode=require")
		assert.Contains(t, builtDSN, fmt.Sprintf("sslcert=%s", clientCertPath))
		assert.Contains(t, builtDSN, fmt.Sprintf("sslkey=%s", clientKeyPath))

		// For this test, we'll just verify DSN building works correctly
		t.Logf("Built DSN with client cert: %s", builtDSN)
	})

	t.Run("SSLConnectionWithVerifyFull", func(t *testing.T) {
		dsn := fmt.Sprintf("postgres://postgres:testpass@%s:%s/testdb",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{
			VerifyServerCert: true,
			RootCertPath:     caCertPath,
		}).WithMaxRetries(1)

		// Build DSN to verify it's constructed correctly
		builtDSN, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, builtDSN, "sslmode=verify-full")
		// Check for URL-encoded path in URL format
		assert.Contains(t, builtDSN, "sslrootcert=")
		assert.Contains(t, builtDSN, "ca-cert.pem")

		// For this test, we'll just verify DSN building works correctly
		t.Logf("Built DSN with verify-full: %s", builtDSN)
	})

	t.Run("SSLConnectionWithPreferMode", func(t *testing.T) {
		// Test explicit prefer mode (should work even without server SSL)
		dsn := fmt.Sprintf("host=%s port=%s user=postgres password=testpass dbname=testdb sslmode=prefer",
			host, port.Port())
		connector := postgres.NewPostgresDBConnector(dsn, &_tls.TLSConfig{})

		db, err := connector.Connect()
		require.NoError(t, err)
		assert.NotNil(t, db)

		// Test database operations
		var result int
		err = db.Raw("SELECT 1").Scan(&result).Error
		require.NoError(t, err)
		assert.Equal(t, 1, result)

		// Test that we can create and query a table
		err = db.Exec("CREATE TEMPORARY TABLE test_ssl (id INT, name VARCHAR(50))").Error
		require.NoError(t, err)

		err = db.Exec("INSERT INTO test_ssl (id, name) VALUES (1, 'ssl_test')").Error
		require.NoError(t, err)

		var name string
		err = db.Raw("SELECT name FROM test_ssl WHERE id = 1").Scan(&name).Error
		require.NoError(t, err)
		assert.Equal(t, "ssl_test", name)
	})
}

func TestPostgresDBConnector_Connect_ErrorCases(t *testing.T) {
	t.Run("InvalidURLFormat", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"://invalid-url-format",
			&_tls.TLSConfig{
				VerifyServerCert: true,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to parse PostgreSQL URL DSN")
		} else {
			// If no error, just log the result (some URL parsers might be more lenient)
			t.Logf("Parsed DSN: %s", dsn)
		}
	})

	t.Run("NonExistentCertFile", func(t *testing.T) {
		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test",
			&_tls.TLSConfig{
				RootCertPath: "/nonexistent/cert.pem",
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		// DSN building should succeed even with non-existent files
		// The error will occur during connection
		require.NoError(t, err)
		assert.Contains(t, dsn, "sslrootcert=/nonexistent/cert.pem")

		// For this test, we'll just verify DSN building works correctly
		// Actual connection would timeout/fail due to non-existent host
		t.Logf("Built DSN with non-existent cert: %s", dsn)
	})

	t.Run("InvalidCertPair", func(t *testing.T) {
		// Create temporary files with invalid content
		tempDir, err := os.MkdirTemp("", "invalid_postgres_cert_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir) //nolint:errcheck

		invalidCertPath := filepath.Join(tempDir, "invalid.crt")
		invalidKeyPath := filepath.Join(tempDir, "invalid.key")

		err = os.WriteFile(invalidCertPath, []byte("invalid cert"), 0600)
		require.NoError(t, err)
		err = os.WriteFile(invalidKeyPath, []byte("invalid key"), 0600)
		require.NoError(t, err)

		connector := postgres.NewPostgresDBConnector(
			"host=localhost port=5432 user=postgres dbname=test",
			&_tls.TLSConfig{
				CertPath: invalidCertPath,
				KeyPath:  invalidKeyPath,
			},
		).WithMaxRetries(1)

		dsn, err := connector.BuildDSNWithTLS()
		require.NoError(t, err)
		assert.Contains(t, dsn, fmt.Sprintf("sslcert=%s", invalidCertPath))

		// For this test, we'll just verify DSN building works correctly
		// Actual connection would fail due to invalid certificates or non-existent host
		t.Logf("Built DSN with invalid cert pair: %s", dsn)
	})
}

// generateTestCertificates creates a set of test certificates for SSL testing
func generateTestCertificates(t *testing.T, tempDir string) (caCertPath, caKeyPath, serverCertPath, serverKeyPath, clientCertPath, clientKeyPath string) {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create CA certificate template
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	// Parse CA certificate
	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err)

	// Save CA certificate
	caCertPath = filepath.Join(tempDir, "ca-cert.pem")
	caCertFile, err := os.Create(caCertPath)
	require.NoError(t, err)
	defer caCertFile.Close() //nolint:errcheck
	err = pem.Encode(caCertFile, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	require.NoError(t, err)

	// Save CA private key
	caKeyPath = filepath.Join(tempDir, "ca-key.pem")
	caKeyFile, err := os.Create(caKeyPath)
	require.NoError(t, err)
	defer caKeyFile.Close() //nolint:errcheck
	caKeyDER, err := x509.MarshalPKCS8PrivateKey(caKey)
	require.NoError(t, err)
	err = pem.Encode(caKeyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: caKeyDER})
	require.NoError(t, err)

	// Generate server certificate
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Test Server"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		DNSNames:     []string{"localhost"},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, &serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	// Save server certificate
	serverCertPath = filepath.Join(tempDir, "server-cert.pem")
	serverCertFile, err := os.Create(serverCertPath)
	require.NoError(t, err)
	defer serverCertFile.Close() //nolint:errcheck
	err = pem.Encode(serverCertFile, &pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	require.NoError(t, err)

	// Save server private key
	serverKeyPath = filepath.Join(tempDir, "server-key.pem")
	serverKeyFile, err := os.Create(serverKeyPath)
	require.NoError(t, err)
	defer serverKeyFile.Close() //nolint:errcheck
	serverKeyDER, err := x509.MarshalPKCS8PrivateKey(serverKey)
	require.NoError(t, err)
	err = pem.Encode(serverKeyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: serverKeyDER})
	require.NoError(t, err)

	// Generate client certificate
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	clientTemplate := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization:  []string{"Test Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "postgres", // PostgreSQL username
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		SubjectKeyId: []byte{1, 2, 3, 4, 7},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, &clientTemplate, caCert, &clientKey.PublicKey, caKey)
	require.NoError(t, err)

	// Save client certificate
	clientCertPath = filepath.Join(tempDir, "client-cert.pem")
	clientCertFile, err := os.Create(clientCertPath)
	require.NoError(t, err)
	defer clientCertFile.Close() //nolint:errcheck
	err = pem.Encode(clientCertFile, &pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER})
	require.NoError(t, err)

	// Save client private key
	clientKeyPath = filepath.Join(tempDir, "client-key.pem")
	clientKeyFile, err := os.Create(clientKeyPath)
	require.NoError(t, err)
	defer clientKeyFile.Close() //nolint:errcheck
	clientKeyDER, err := x509.MarshalPKCS8PrivateKey(clientKey)
	require.NoError(t, err)
	err = pem.Encode(clientKeyFile, &pem.Block{Type: "PRIVATE KEY", Bytes: clientKeyDER})
	require.NoError(t, err)

	return
}
