package postgres

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

var testDBLock sync.Mutex

type testEnv struct {
	DB *sql.DB
}

func SetupTestDB(t *testing.T) testEnv {
	t.Helper()
	testDBLock.Lock()
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://admin:admin123@localhost:5432/ppo_cource?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		testDBLock.Unlock()
		t.Skipf("cannot open postgres: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		testDBLock.Unlock()
		_ = db.Close()
		t.Skipf("cannot connect to postgres: %v", err)
	}

	t.Cleanup(func() {
		truncateAll(t, db)
		_ = db.Close()
		testDBLock.Unlock()
	})

	runMigrations(t, db)
	truncateAll(t, db)
	return testEnv{DB: db}
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

func truncateAll(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(`TRUNCATE moves, game_board_states, games, user_credentials, users RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot determine caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
