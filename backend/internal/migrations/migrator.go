package migrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationTableSQL = `
CREATE TABLE IF NOT EXISTS public.schema_migrations (
	version BIGINT PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`

// ApplyUpMigrations applies pending *.up.sql files in lexical version order.
func ApplyUpMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	resolvedDir, err := resolveMigrationsDir(migrationsDir)
	if err != nil {
		return err
	}

	if _, err := pool.Exec(ctx, migrationTableSQL); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	migrationFiles, err := collectUpMigrationFiles(resolvedDir)
	if err != nil {
		return err
	}

	for _, migrationFile := range migrationFiles {
		alreadyApplied, err := isMigrationApplied(ctx, pool, migrationFile.Version)
		if err != nil {
			return fmt.Errorf("failed to check migration %s: %w", migrationFile.Name, err)
		}
		if alreadyApplied {
			continue
		}

		if err := applySingleMigration(ctx, pool, migrationFile); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migrationFile.Name, err)
		}
	}

	return nil
}

type migrationFile struct {
	Name    string
	Path    string
	Version int64
}

func resolveMigrationsDir(migrationsDir string) (string, error) {
	candidateDirs := []string{migrationsDir, filepath.Join("backend", migrationsDir)}

	for _, candidateDir := range candidateDirs {
		if strings.TrimSpace(candidateDir) == "" {
			continue
		}
		if info, err := os.Stat(candidateDir); err == nil && info.IsDir() {
			return candidateDir, nil
		}
	}

	return "", fmt.Errorf("migrations directory %q not found", migrationsDir)
}

func collectUpMigrationFiles(migrationsDir string) ([]migrationFile, error) {
	directoryEntries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations dir: %w", err)
	}

	upMigrations := make([]migrationFile, 0)
	for _, directoryEntry := range directoryEntries {
		if directoryEntry.IsDir() {
			continue
		}
		fileName := directoryEntry.Name()
		if !strings.HasSuffix(fileName, ".up.sql") {
			continue
		}

		version, err := parseMigrationVersion(fileName)
		if err != nil {
			return nil, err
		}

		upMigrations = append(upMigrations, migrationFile{
			Name:    fileName,
			Path:    filepath.Join(migrationsDir, fileName),
			Version: version,
		})
	}

	sort.Slice(upMigrations, func(left, right int) bool {
		return upMigrations[left].Version < upMigrations[right].Version
	})

	return upMigrations, nil
}

func parseMigrationVersion(fileName string) (int64, error) {
	parts := strings.SplitN(fileName, "_", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("migration file %q must start with numeric version", fileName)
	}

	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version in %q: %w", fileName, err)
	}

	return version, nil
}

func isMigrationApplied(ctx context.Context, pool *pgxpool.Pool, version int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM public.schema_migrations WHERE version = $1)`

	var exists bool
	if err := pool.QueryRow(ctx, query, version).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func applySingleMigration(ctx context.Context, pool *pgxpool.Pool, migration migrationFile) error {
	sqlBytes, err := os.ReadFile(migration.Path)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	sqlText := strings.TrimSpace(string(sqlBytes))
	if sqlText == "" {
		return errors.New("migration file is empty")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start migration transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, sqlText); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO public.schema_migrations (version, name) VALUES ($1, $2)`,
		migration.Version,
		migration.Name,
	); err != nil {
		return fmt.Errorf("failed to register applied migration: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}
