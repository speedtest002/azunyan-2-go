package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (d *DB) Migrate(ctx context.Context) error {
	entries, err := fs.ReadDir(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("db.Migrate read dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := fs.ReadFile(migrationsFS, "migrations/"+entry.Name())
		if err != nil {
			return fmt.Errorf("db.Migrate read %s: %w", entry.Name(), err)
		}

		for _, stmt := range splitSQLStatements(string(content)) {
			if _, err := d.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("db.Migrate (%s): %w", entry.Name(), err)
			}
		}
	}

	slog.Info("DB migrations applied", slog.Int("files", len(entries)))
	return nil
}

func splitSQLStatements(raw string) []string {
	var cleanLines []string
	for _, line := range strings.Split(raw, "\n") {
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}
		if strings.TrimSpace(line) != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	cleaned := strings.Join(cleanLines, "\n")
	parts := strings.Split(cleaned, ";")
	var stmts []string
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt != "" {
			stmts = append(stmts, stmt)
		}
	}
	return stmts
}
