package postgres

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	platformdb "github.com/kubeflow/hub/internal/platform/db"
	"github.com/kubeflow/hub/internal/platform/db/types"
	_tls "github.com/kubeflow/hub/internal/platform/tls"
	"golang.org/x/sync/semaphore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	// postgresMaxRetriesDefault is the maximum number of attempts to retry PostgreSQL connection.
	postgresMaxRetriesDefault = 25 // 25 attempts with incremental backoff (1s, 2s, 3s, ..., 25s) it's ~5 minutes
)

func init() {
	platformdb.RegisterConnectorFactory(types.DatabaseTypePostgres, func(dsn string, tlsConfig *_tls.TLSConfig) platformdb.Connector {
		return NewPostgresDBConnector(dsn, tlsConfig)
	})
}

type PostgresDBConnector struct {
	DSN          string
	TLSConfig    *_tls.TLSConfig
	db           *gorm.DB
	connectMutex *semaphore.Weighted
	maxRetries   int
}

func NewPostgresDBConnector(
	dsn string,
	tlsConfig *_tls.TLSConfig,
) *PostgresDBConnector {
	return &PostgresDBConnector{
		DSN:          dsn,
		TLSConfig:    tlsConfig,
		maxRetries:   postgresMaxRetriesDefault,
		connectMutex: semaphore.NewWeighted(1),
	}
}

func (c *PostgresDBConnector) WithMaxRetries(maxRetries int) *PostgresDBConnector {
	c.maxRetries = maxRetries

	return c
}

func (c *PostgresDBConnector) Connect() (*gorm.DB, error) {
	return c.ConnectContext(context.Background())
}

// ConnectContext bounds connection attempts and retry delays by ctx. The
// returned database handle remains usable after ctx is canceled.
func (c *PostgresDBConnector) ConnectContext(ctx context.Context) (*gorm.DB, error) {
	// Use mutex to ensure only one connection attempt at a time
	if err := c.connectMutex.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	defer c.connectMutex.Release(1)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// If we already have a working connection, return it
	if c.db != nil {
		return c.db, nil
	}

	var err error

	dsn := c.DSN
	if c.needsTLSConfig() {
		dsn, err = c.BuildDSNWithTLS()
		if err != nil {
			return nil, fmt.Errorf("failed to build DSN with TLS: %w", err)
		}
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL connection configuration: %w", err)
	}

	for i := range c.maxRetries {
		glog.V(2).Infof("Attempting to connect to PostgreSQL (attempt %d/%d)", i+1, c.maxRetries)
		sqlDB := stdlib.OpenDB(*config)
		err = sqlDB.PingContext(ctx)
		if err == nil {
			var connectedDB *gorm.DB
			connectedDB, err = gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
				Logger:               logger.Default.LogMode(logger.Silent),
				TranslateError:       true,
				DisableAutomaticPing: true,
			})
			if err == nil {
				c.db = connectedDB
				glog.Info("Successfully connected to PostgreSQL database")
				return c.db, nil
			}
		}
		err = errors.Join(err, sqlDB.Close())
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		glog.Warningf("Retrying connection to PostgreSQL (attempt %d/%d): %v", i+1, c.maxRetries, err)
		if i+1 < c.maxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(i+1) * time.Second):
			}
		}
	}
	return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
}

func (c *PostgresDBConnector) DB() *gorm.DB {
	return c.db
}

func (c *PostgresDBConnector) needsTLSConfig() bool {
	if c.TLSConfig == nil {
		return false
	}

	// Log warning if cipher configuration is specified (not supported by PostgreSQL)
	if c.TLSConfig.Cipher != "" {
		glog.Warningf("SSL cipher configuration is not supported for PostgreSQL connections, ignoring cipher: %s", c.TLSConfig.Cipher)
	}

	return c.TLSConfig.CertPath != "" || c.TLSConfig.KeyPath != "" || c.TLSConfig.RootCertPath != "" || c.TLSConfig.CAPath != "" || c.TLSConfig.VerifyServerCert
}

// BuildDSNWithTLS builds a PostgreSQL DSN with SSL/TLS parameters based on the TLS configuration
func (c *PostgresDBConnector) BuildDSNWithTLS() (string, error) {
	if c.TLSConfig == nil {
		return c.DSN, nil
	}

	// Parse the existing DSN to determine format (URL or key=value pairs)
	if strings.HasPrefix(c.DSN, "postgres://") || strings.HasPrefix(c.DSN, "postgresql://") {
		return c.buildURLDSNWithTLS()
	}

	return c.buildKeyValueDSNWithTLS()
}

// buildURLDSNWithTLS handles URL-format DSNs (postgres://...)
func (c *PostgresDBConnector) buildURLDSNWithTLS() (string, error) {
	parsedURL, err := url.Parse(c.DSN)
	if err != nil {
		return "", fmt.Errorf("failed to parse PostgreSQL URL DSN: %w", err)
	}

	query := parsedURL.Query()

	// Set SSL mode based on TLS configuration
	if c.TLSConfig.VerifyServerCert {
		query.Set("sslmode", "verify-full")
	} else if c.TLSConfig.CertPath != "" || c.TLSConfig.KeyPath != "" || c.TLSConfig.RootCertPath != "" || c.TLSConfig.CAPath != "" {
		query.Set("sslmode", "require")
	}

	// Add certificate paths
	if c.TLSConfig.CertPath != "" {
		query.Set("sslcert", c.TLSConfig.CertPath)
	}
	if c.TLSConfig.KeyPath != "" {
		query.Set("sslkey", c.TLSConfig.KeyPath)
	}
	if c.TLSConfig.RootCertPath != "" {
		query.Set("sslrootcert", c.TLSConfig.RootCertPath)
	} else if c.TLSConfig.CAPath != "" {
		query.Set("sslrootcert", c.TLSConfig.CAPath)
	}

	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

// buildKeyValueDSNWithTLS handles key=value format DSNs
func (c *PostgresDBConnector) buildKeyValueDSNWithTLS() (string, error) {
	dsn := c.DSN

	// Set SSL mode based on TLS configuration
	if c.TLSConfig.VerifyServerCert {
		dsn = c.addOrUpdateDSNParam(dsn, "sslmode", "verify-full")
	} else if c.TLSConfig.CertPath != "" || c.TLSConfig.KeyPath != "" || c.TLSConfig.RootCertPath != "" || c.TLSConfig.CAPath != "" {
		dsn = c.addOrUpdateDSNParam(dsn, "sslmode", "require")
	}

	// Add certificate paths
	if c.TLSConfig.CertPath != "" {
		dsn = c.addOrUpdateDSNParam(dsn, "sslcert", c.TLSConfig.CertPath)
	}
	if c.TLSConfig.KeyPath != "" {
		dsn = c.addOrUpdateDSNParam(dsn, "sslkey", c.TLSConfig.KeyPath)
	}
	if c.TLSConfig.RootCertPath != "" {
		dsn = c.addOrUpdateDSNParam(dsn, "sslrootcert", c.TLSConfig.RootCertPath)
	} else if c.TLSConfig.CAPath != "" {
		dsn = c.addOrUpdateDSNParam(dsn, "sslrootcert", c.TLSConfig.CAPath)
	}

	return dsn, nil
}

// addOrUpdateDSNParam adds or updates a parameter in a key=value DSN string
func (c *PostgresDBConnector) addOrUpdateDSNParam(dsn, key, value string) string {
	// Split DSN into individual parameters
	parts := strings.Fields(dsn)
	keyPrefix := key + "="
	updated := false

	// Update existing parameter or collect all parts
	for i, part := range parts {
		if strings.HasPrefix(part, keyPrefix) {
			parts[i] = keyPrefix + value
			updated = true
			break
		}
	}

	// Add new parameter if it didn't exist
	if !updated {
		parts = append(parts, keyPrefix+value)
	}

	return strings.Join(parts, " ")
}
