package postgres

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestPool opens WHITE_TEST_DATABASE_URL and applies the given migrations.
// Tests that need a database call this and skip when the variable is unset,
// so `go test ./...` stays green on a laptop without Postgres.
func TestPool(t *testing.T, migrations fs.FS, dir, schema string) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("WHITE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("WHITE_TEST_DATABASE_URL tidak diset")
	}
	ctx := context.Background()
	pool, err := Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := Migrate(ctx, pool, migrations, dir, schema, slog.New(slog.NewTextHandler(os.Stderr, nil))); err != nil {
		t.Fatal(err)
	}
	return pool
}

// TruncateSchema empties every table of a schema between tests.
func TruncateSchema(t *testing.T, pool *pgxpool.Pool, schema string) {
	t.Helper()
	ctx := context.Background()
	rows, err := pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = $1 AND tablename <> 'goose_db_version'`, schema)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, schema+"."+name)
	}
	rows.Close()
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, `TRUNCATE TABLE `+tbl+` CASCADE`); err != nil {
			t.Fatal(err)
		}
	}
}
