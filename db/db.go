package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	_ "github.com/sijms/go-ora/v2"
)

type DB struct {
	*sql.DB
	cancelKeepAlive context.CancelFunc
}

func Connect(user, password, dsn string) (*DB, error) {
	u := &url.URL{
		Scheme: "oracle",
		User:   url.UserPassword(user, password),
		Host:   "localhost:1521",
	}
	u.RawQuery = "connStr=" + url.QueryEscape(dsn)

	sqlDB, err := sql.Open("oracle", u.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	db := &DB{DB: sqlDB}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("initial ping: %w", err)
	}

	return db, nil
}

func (db *DB) StartKeepAlive(ctx context.Context) {
	ctx, db.cancelKeepAlive = context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(4 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				db.healthCheck(ctx)
			}
		}
	}()
}

func (db *DB) healthCheck(ctx context.Context) {
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.QueryRowContext(queryCtx, "SELECT 1 FROM DUAL").Scan(new(int)); err != nil {
		slog.Warn("DB health check failed, retrying with backoff", "err", err)
		db.reconnectWithBackoff(ctx)
	} else {
		slog.Info("DB health check OK")
	}
}

func (db *DB) reconnectWithBackoff(ctx context.Context) {
	backoff := 1 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := db.QueryRowContext(queryCtx, "SELECT 1 FROM DUAL").Scan(new(int))
		cancel()

		if err == nil {
			slog.Info("DB reconnected")
			return
		}

		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		slog.Warn("DB reconnect attempt failed, retrying", "backoff", backoff, "err", err)
	}
}

func (db *DB) Close() error {
	if db.cancelKeepAlive != nil {
		db.cancelKeepAlive()
	}
	return db.DB.Close()
}
