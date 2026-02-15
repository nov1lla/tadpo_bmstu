package testsupport

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

type PostgresEnv struct {
	DSN     string
	DBName  string
	Cleanup func()
}

func SetupPostgres(t *testing.T) PostgresEnv {
	t.Helper()
	failOnMissing := os.Getenv("POSTGRES_REQUIRED") == "1"
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://admin:admin123@localhost:5432/ppo_cource?sslmode=disable"
	}
	adminDSN, testDSN, dbName, err := deriveTestDSN(dsn)
	if err != nil {
		t.Fatalf("derive test dsn: %v", err)
	}

	admin, err := sql.Open("postgres", adminDSN)
	if err != nil {
		if failOnMissing {
			t.Fatalf("open postgres admin: %v", err)
		}
		t.Skipf("open postgres admin: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		_ = admin.Close()
		if failOnMissing {
			t.Fatalf("connect postgres admin: %v", err)
		}
		t.Skipf("connect postgres admin: %v", err)
	}
	if err := createDatabase(ctx, admin, dbName); err != nil {
		_ = admin.Close()
		if failOnMissing {
			t.Fatalf("create test db: %v", err)
		}
		t.Skipf("create test db: %v", err)
	}
	_ = admin.Close()

	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	runMigrations(t, db)
	_ = db.Close()

	cleanup := func() {
		_ = dropDatabase(dsn, dbName)
	}

	return PostgresEnv{DSN: testDSN, DBName: dbName, Cleanup: cleanup}
}

func deriveTestDSN(dsn string) (string, string, string, error) {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return "", "", "", err
	}
	dbName := parsed.Path
	if dbName == "" || dbName == "/" {
		dbName = "ppo_cource"
	} else {
		dbName = dbName[1:]
	}
	suffix, err := randomHex(6)
	if err != nil {
		return "", "", "", err
	}
	testName := dbName + "_test_" + suffix
	parsed.Path = "/postgres"
	adminDSN := parsed.String()
	parsed.Path = "/" + testName
	testDSN := parsed.String()
	return adminDSN, testDSN, testName, nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func createDatabase(ctx context.Context, admin *sql.DB, name string) error {
	_, err := admin.ExecContext(ctx, `CREATE DATABASE "`+name+`"`)
	return err
}

func dropDatabase(dsn, name string) error {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	parsed.Path = "/postgres"
	admin, err := sql.Open("postgres", parsed.String())
	if err != nil {
		return err
	}
	defer admin.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = admin.ExecContext(ctx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, name)
	_, err = admin.ExecContext(ctx, `DROP DATABASE IF EXISTS "`+name+`"`)
	return err
}

func runMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	migrationsDir := filepath.Join(projectRoot(t), "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		files = append(files, name)
	}
	if len(files) == 0 {
		t.Fatalf("no migrations found in %s", migrationsDir)
	}
	sort.Strings(files)
	for _, name := range files {
		schemaPath := filepath.Join(migrationsDir, name)
		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			t.Fatalf("apply migration %s: %v", name, err)
		}
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot determine caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}
