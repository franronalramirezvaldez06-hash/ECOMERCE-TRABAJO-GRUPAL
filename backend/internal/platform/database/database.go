// Package database maneja la conexion a PostgreSQL y las migraciones.
package database

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"ecommerce/internal/apperr"
	"ecommerce/migrations"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect abre el pool y reintenta mientras la base arranca (docker compose).
func Connect(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("DATABASE_URL invalida: %w", err)
	}
	var lastErr error
	for attempt := 1; attempt <= 15; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		log.Warn("base de datos no disponible, reintentando", "intento", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return nil, fmt.Errorf("no se pudo conectar a la base de datos: %w", lastErr)
}

// Migrate aplica en orden los .sql embebidos que aun no se hayan ejecutado.
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || len(name) < 5 || name[len(name)-4:] != ".sql" {
			continue
		}
		var applied bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		script, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(script)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migracion %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		log.Info("migracion aplicada", "version", name)
	}
	return nil
}

// MapError traduce errores de pgx/PostgreSQL a errores de negocio.
func MapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: valor duplicado (%s)", apperr.ErrConflict, pg.ConstraintName)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: referencia inexistente (%s)", apperr.ErrNotFound, pg.ConstraintName)
		case "22P02": // invalid_text_representation (uuid mal formado)
			return apperr.ErrNotFound
		case "23514": // check_violation
			return apperr.Validation("dato fuera de rango (%s)", pg.ConstraintName)
		}
	}
	return err
}
