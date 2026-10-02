package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// A stable, database-scoped lock independent of ingestion leadership and the
// SQL migration library. No tables are required, even during schema recovery.
const initializationLockID int64 = 0x4b464843494e4954

// WithInitializationLock serializes initialization across PostgreSQL clients.
// The callback must finish its database work before returning.
func WithInitializationLock(ctx context.Context, db *gorm.DB, initialize func() error) (result error) {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := conn.Close(); !errors.Is(err, sql.ErrConnDone) {
			result = errors.Join(result, err)
		}
	}()
	// Discard the session if acquisition or release fails: the server may have
	// acquired the lock even when the client did not receive its result.
	locked := false
	defer func() {
		if locked {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			var released bool
			err := conn.QueryRowContext(cleanupCtx, "SELECT pg_advisory_unlock($1)", initializationLockID).Scan(&released)
			if err == nil && released {
				return
			}
			if err == nil {
				err = errors.New("session no longer holds the lock")
			}
			result = errors.Join(result, fmt.Errorf("unable to release initialization lock: %w", err))
		}
		err := conn.Raw(func(any) error { return driver.ErrBadConn })
		if err != nil && !errors.Is(err, driver.ErrBadConn) && !errors.Is(err, sql.ErrConnDone) {
			result = errors.Join(result, err)
		}
	}()
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", initializationLockID); err != nil {
		return errors.Join(ctx.Err(), fmt.Errorf("unable to acquire initialization lock: %w", err))
	}
	locked = true
	return initialize()
}
