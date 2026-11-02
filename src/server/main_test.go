package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

func clearPostgresMetadataURLEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"DEKART_POSTGRES_URL",
		"DEKART_POSTGRES_USER",
		"DEKART_POSTGRES_PASSWORD",
		"DEKART_POSTGRES_HOST",
		"DEKART_POSTGRES_PORT",
		"DEKART_POSTGRES_DB",
	} {
		key := key
		value, ok := os.LookupEnv(key)
		os.Unsetenv(key)
		t.Cleanup(func() {
			if ok {
				os.Setenv(key, value)
			} else {
				os.Unsetenv(key)
			}
		})
	}
}

func TestValidateStorageConfigDisablesFileUploadForPG(t *testing.T) {
	t.Setenv("DEKART_STORAGE", "PG")
	t.Setenv("DEKART_ALLOW_FILE_UPLOAD", "1")
	t.Setenv("DEKART_CLOUD_STORAGE_BUCKET", "sqlite-backups")

	validateStorageConfig()

	if got := os.Getenv("DEKART_ALLOW_FILE_UPLOAD"); got != "0" {
		t.Fatalf("expected file upload to be disabled, got %q", got)
	}
	if got := os.Getenv("DEKART_CLOUD_STORAGE_BUCKET"); got != "sqlite-backups" {
		t.Fatalf("expected backup bucket to remain configured, got %q", got)
	}
}

func TestPostgresMetadataURLUsesExplicitURL(t *testing.T) {
	clearPostgresMetadataURLEnv(t)
	os.Setenv("DEKART_POSTGRES_URL", "postgres://custom")

	if got := postgresMetadataURL(); got != "postgres://custom" {
		t.Fatalf("unexpected URL: %q", got)
	}
}

func TestPostgresMetadataURLUsesDefaultsForMissingStructuredFields(t *testing.T) {
	clearPostgresMetadataURLEnv(t)
	os.Setenv("DEKART_POSTGRES_HOST", "db")

	want := "postgres://dekart:@db:5432/dekart?sslmode=disable"
	if got := postgresMetadataURL(); got != want {
		t.Fatalf("unexpected URL: %q", got)
	}
}

// A database at fork migration version N must still pass the stock migration check with ErrNoChange,
// which is what keeps rollback to the stock image a plain image swap.
func TestForkMigrationsLeaveStockMigrationCheckAtNoChange(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "dekart.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	stockMigrations := func() (*migrate.Migrate, error) {
		driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return nil, err
		}
		return migrate.NewWithDatabaseInstance("file://../../sqlite/migrations", "sqlite3", driver)
	}

	stock, err := stockMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := stock.Up(); err != nil {
		t.Fatalf("stock migrations: %v", err)
	}
	stockVersion, _, err := stock.Version()
	if err != nil {
		t.Fatal(err)
	}

	if err := applyForkSqliteMigrations(db, "file://../../sqlite/klikpeta-migrations"); err != nil {
		t.Fatalf("fork migrations: %v", err)
	}
	// running the fork track again is a no-op
	if err := applyForkSqliteMigrations(db, "file://../../sqlite/klikpeta-migrations"); err != nil {
		t.Fatalf("fork migrations rerun: %v", err)
	}

	var forkVersion int
	var forkDirty bool
	if err := db.QueryRow("SELECT version, dirty FROM "+forkMigrationsTable).Scan(&forkVersion, &forkDirty); err != nil {
		t.Fatalf("fork version row: %v", err)
	}
	if forkVersion < 1 || forkDirty {
		t.Fatalf("expected clean fork version >= 1, got %d dirty=%v", forkVersion, forkDirty)
	}

	// the stock check, as the stock image runs it at startup
	stock, err = stockMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := stock.Up(); err != migrate.ErrNoChange {
		t.Fatalf("expected ErrNoChange from stock migrations, got %v", err)
	}
	version, dirty, err := stock.Version()
	if err != nil {
		t.Fatal(err)
	}
	if version != stockVersion || dirty {
		t.Fatalf("stock version moved to %d dirty=%v, expected %d", version, dirty, stockVersion)
	}
}
