package bdd

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/cucumber/godog"

	_ "github.com/lib/pq"
)

type appEnv struct {
	repoRoot    string
	buildDir    string
	staticDir   string
	dataPlugin  string
	bizPlugin   string
	webappBin   string
	baseURL     string
	testToken   string
	serverCmd   *exec.Cmd
	postgresEnv *postgresEnv
}

type scenarioState struct {
	app *appEnv

	login       string
	password    string
	newPassword string

	challengeID string
	code        string
	token       string
}

var globalApp *appEnv

func InitializeTestSuite(ctx *godog.TestSuiteContext) {
	ctx.BeforeSuite(func() {
		globalApp = startWebappOrDie()
	})
	ctx.AfterSuite(func() {
		if globalApp != nil {
			globalApp.stop()
		}
	})
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	state := &scenarioState{app: globalApp}

	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		// fresh credentials per scenario (technical user), no plaintext in repo/feature files.
		state.login = fmt.Sprintf("tech_%d@example.test", time.Now().UnixNano())
		state.password = randomPassword()
		state.newPassword = ""
		state.challengeID = ""
		state.code = ""
		state.token = ""
		return ctx, nil
	})

	ctx.Step(`^a technical user is registered$`, state.stepTechnicalUserRegistered)
	ctx.Step(`^I start login$`, state.stepStartLogin)
	ctx.Step(`^I receive a "([^"]+)" code$`, state.stepReceiveCode)
	ctx.Step(`^I verify login with the code$`, state.stepVerifyLogin)
	ctx.Step(`^I am authenticated$`, state.stepAuthenticated)
	ctx.Step(`^I request a password change$`, state.stepPasswordChangeRequest)
	ctx.Step(`^I confirm the password change with the code$`, state.stepPasswordChangeConfirm)
	ctx.Step(`^login with the old password is rejected$`, state.stepOldPasswordRejected)
	ctx.Step(`^login with the new password succeeds$`, state.stepNewPasswordSucceeds)
	ctx.Step(`^I enter a wrong code 3 times$`, state.stepWrongCodeThreeTimes)
	ctx.Step(`^my account becomes locked$`, state.stepAccountLocked)
	ctx.Step(`^my account is locked$`, state.stepEnsureLocked)
	ctx.Step(`^I request account recovery$`, state.stepRecoveryRequest)
	ctx.Step(`^I confirm recovery with a new password$`, state.stepRecoveryConfirm)
	ctx.Step(`^login with the recovered password succeeds$`, state.stepRecoveredLoginSucceeds)
}

func startWebappOrDie() *appEnv {
	buildDir := mustTempDir()
	staticDir := filepath.Join(buildDir, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		panic(err)
	}

	repoRoot := projectRoot()
	dataDir := filepath.Join(repoRoot, "code/components/data")
	businessDir := filepath.Join(repoRoot, "code/components/business")
	webappDir := filepath.Join(repoRoot, "code/apps/webapp")

	dataPlugin := filepath.Join(buildDir, "data.so")
	bizPlugin := filepath.Join(buildDir, "business.so")
	webappBin := filepath.Join(buildDir, "webapp")

	runCmd(dataDir, "go", "build", "-tags", "plugin", "-buildmode=plugin", "-o", dataPlugin, "./cmd/plugin")
	runCmd(businessDir, "go", "build", "-tags", "plugin", "-buildmode=plugin", "-o", bizPlugin, "./cmd/plugin")
	runCmd(webappDir, "go", "build", "-o", webappBin, "./cmd/webapp")

	pg, err := setupPostgres(repoRoot)
	if err != nil {
		panic(err)
	}

	addr := freeAddr()
	baseURL := "http://" + addr

	testToken := "bdd-token"
	server := exec.Command(webappBin)
	server.Env = append(os.Environ(),
		"WEBAPP_ADDR="+addr,
		"DATA_PLUGIN_PATH="+dataPlugin,
		"BUSINESS_PLUGIN_PATH="+bizPlugin,
		"DATA_SOURCE="+pg.DSN,
		"WEBAPP_STATIC_DIR="+staticDir,
		"WEBAPP_TEST_MODE=1",
		"WEBAPP_TEST_TOKEN="+testToken,
		"AUTH_MAX_ATTEMPTS=3",
		"AUTH_LOCK_DURATION=5s",
		"AUTH_CODE_TTL=2m",
		"AUTH_SESSION_TTL=2m",
	)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr

	if err := server.Start(); err != nil {
		pg.Cleanup()
		panic(err)
	}

	waitForServerOrDie(baseURL)

	return &appEnv{
		repoRoot:    repoRoot,
		buildDir:    buildDir,
		staticDir:   staticDir,
		dataPlugin:  dataPlugin,
		bizPlugin:   bizPlugin,
		webappBin:   webappBin,
		baseURL:     baseURL,
		testToken:   testToken,
		serverCmd:   server,
		postgresEnv: pg,
	}
}

func (a *appEnv) stop() {
	if a.serverCmd != nil && a.serverCmd.Process != nil {
		_ = a.serverCmd.Process.Signal(os.Interrupt)
		_, _ = a.serverCmd.Process.Wait()
	}
	if a.postgresEnv != nil {
		a.postgresEnv.Cleanup()
	}
	_ = os.RemoveAll(a.buildDir)
}

func (s *scenarioState) stepTechnicalUserRegistered() error {
	body := map[string]string{"login": s.login, "password": s.password}
	resp, status, err := postJSON(s.app.baseURL+"/api/auth/register", body, nil)
	if err != nil {
		return err
	}
	if status == http.StatusCreated {
		_ = resp
		return nil
	}
	// If already exists, acceptable for "technical user exists" semantics.
	if status == http.StatusBadRequest {
		return nil
	}
	return fmt.Errorf("unexpected register status: %d", status)
}

func (s *scenarioState) stepStartLogin() error {
	body := map[string]string{"login": s.login, "password": s.password}
	resp, status, err := postJSON(s.app.baseURL+"/api/auth/login", body, nil)
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("expected 202, got %d", status)
	}
	s.challengeID = getString(resp, "challengeId")
	return nil
}

func (s *scenarioState) stepReceiveCode(purpose string) error {
	msg, status, err := getJSON(s.app.baseURL+"/api/test/outbox?login="+urlQueryEscape(s.login)+"&purpose="+urlQueryEscape(purpose), map[string]string{
		"X-Test-Token": s.app.testToken,
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	s.code = getString(msg, "code")
	if s.code == "" {
		return errors.New("empty code")
	}
	return nil
}

func (s *scenarioState) stepVerifyLogin() error {
	req := map[string]string{"challengeId": s.challengeID, "code": s.code}
	resp, status, err := postJSON(s.app.baseURL+"/api/auth/login/verify", req, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	s.token = getString(resp, "token")
	if s.token == "" {
		return errors.New("empty token")
	}
	return nil
}

func (s *scenarioState) stepAuthenticated() error {
	if s.token == "" {
		return errors.New("token not set")
	}
	return nil
}

func (s *scenarioState) stepPasswordChangeRequest() error {
	s.newPassword = randomPassword()
	req := map[string]string{"oldPassword": s.password, "newPassword": s.newPassword}
	resp, status, err := postJSON(s.app.baseURL+"/api/auth/password/change/request", req, map[string]string{
		"Authorization": "Bearer " + s.token,
	})
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("expected 202, got %d", status)
	}
	s.challengeID = getString(resp, "challengeId")
	return nil
}

func (s *scenarioState) stepPasswordChangeConfirm() error {
	req := map[string]string{"challengeId": s.challengeID, "code": s.code}
	_, status, err := postJSON(s.app.baseURL+"/api/auth/password/change/confirm", req, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	return nil
}

func (s *scenarioState) stepOldPasswordRejected() error {
	body := map[string]string{"login": s.login, "password": s.password}
	_, status, err := postJSON(s.app.baseURL+"/api/auth/login", body, nil)
	if err != nil {
		return err
	}
	if status == http.StatusAccepted {
		return errors.New("expected old password login to be rejected")
	}
	return nil
}

func (s *scenarioState) stepNewPasswordSucceeds() error {
	body := map[string]string{"login": s.login, "password": s.newPassword}
	resp, status, err := postJSON(s.app.baseURL+"/api/auth/login", body, nil)
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("expected 202, got %d", status)
	}
	chID := getString(resp, "challengeId")

	msg, status, err := getJSON(s.app.baseURL+"/api/test/outbox?login="+urlQueryEscape(s.login)+"&purpose=login", map[string]string{
		"X-Test-Token": s.app.testToken,
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	code := getString(msg, "code")

	verifyReq := map[string]string{"challengeId": chID, "code": code}
	verifyResp, status, err := postJSON(s.app.baseURL+"/api/auth/login/verify", verifyReq, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	token := getString(verifyResp, "token")
	if token == "" {
		return errors.New("empty token")
	}
	return nil
}

func (s *scenarioState) stepWrongCodeThreeTimes() error {
	wrong := "000000"
	for i := 0; i < 3; i++ {
		req := map[string]string{"challengeId": s.challengeID, "code": wrong}
		_, _, err := postJSON(s.app.baseURL+"/api/auth/login/verify", req, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *scenarioState) stepAccountLocked() error {
	body := map[string]string{"login": s.login, "password": s.password}
	_, status, err := postJSON(s.app.baseURL+"/api/auth/login", body, nil)
	if err != nil {
		return err
	}
	if status != http.StatusTooManyRequests {
		return fmt.Errorf("expected 429, got %d", status)
	}
	return nil
}

func (s *scenarioState) stepEnsureLocked() error {
	// create a lock by exhausting attempts
	if err := s.stepStartLogin(); err != nil {
		return err
	}
	if err := s.stepWrongCodeThreeTimes(); err != nil {
		return err
	}
	return s.stepAccountLocked()
}

func (s *scenarioState) stepRecoveryRequest() error {
	req := map[string]string{"login": s.login}
	_, status, err := postJSON(s.app.baseURL+"/api/auth/recover/request", req, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	return nil
}

func (s *scenarioState) stepRecoveryConfirm() error {
	s.newPassword = randomPassword()
	req := map[string]string{"login": s.login, "code": s.code, "newPassword": s.newPassword}
	_, status, err := postJSON(s.app.baseURL+"/api/auth/recover/confirm", req, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	return nil
}

func (s *scenarioState) stepRecoveredLoginSucceeds() error {
	body := map[string]string{"login": s.login, "password": s.newPassword}
	resp, status, err := postJSON(s.app.baseURL+"/api/auth/login", body, nil)
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("expected 202, got %d", status)
	}
	chID := getString(resp, "challengeId")
	msg, status, err := getJSON(s.app.baseURL+"/api/test/outbox?login="+urlQueryEscape(s.login)+"&purpose=login", map[string]string{
		"X-Test-Token": s.app.testToken,
	})
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	code := getString(msg, "code")
	verifyReq := map[string]string{"challengeId": chID, "code": code}
	verifyResp, status, err := postJSON(s.app.baseURL+"/api/auth/login/verify", verifyReq, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("expected 200, got %d", status)
	}
	token := getString(verifyResp, "token")
	if token == "" {
		return errors.New("empty token")
	}
	return nil
}

func projectRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("cannot resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func mustTempDir() string {
	dir, err := os.MkdirTemp("", "tadpo-bdd-*")
	if err != nil {
		panic(err)
	}
	return dir
}

func freeAddr() string {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return l.Addr().String()
}

func waitForServerOrDie(baseURL string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/api/users")
		if err == nil && resp.StatusCode == http.StatusMethodNotAllowed {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(150 * time.Millisecond)
	}
	panic("server did not start")
}

func runCmd(dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func postJSON(url string, payload any, headers map[string]string) (map[string]any, int, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return decoded, resp.StatusCode, nil
}

func getJSON(url string, headers map[string]string) (map[string]any, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return decoded, resp.StatusCode, nil
}

func getString(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return text
}

func randomPassword() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, 14)
	if _, err := rand.Read(buf); err != nil {
		return "fallbackPass12345"
	}
	var b strings.Builder
	for i := 0; i < len(buf); i++ {
		b.WriteByte(letters[int(buf[i])%len(letters)])
	}
	return b.String()
}

func urlQueryEscape(s string) string {
	return url.QueryEscape(s)
}

type postgresEnv struct {
	DSN     string
	DBName  string
	Cleanup func()
}

func setupPostgres(repoRoot string) (*postgresEnv, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://admin:admin123@localhost:5432/ppo_cource?sslmode=disable"
	}

	adminDSN, testDSN, dbName, err := deriveTestDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("derive test dsn: %w", err)
	}

	admin, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres admin: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		_ = admin.Close()
		return nil, fmt.Errorf("connect postgres admin: %w", err)
	}
	if err := createDatabase(ctx, admin, dbName); err != nil {
		_ = admin.Close()
		return nil, fmt.Errorf("create test db: %w", err)
	}
	_ = admin.Close()

	db, err := sql.Open("postgres", testDSN)
	if err != nil {
		return nil, fmt.Errorf("open test db: %w", err)
	}
	if err := runMigrations(repoRoot, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	_ = db.Close()

	cleanup := func() {
		_ = dropDatabase(dsn, dbName)
	}

	return &postgresEnv{DSN: testDSN, DBName: dbName, Cleanup: cleanup}, nil
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
	testName := dbName + "_bdd_" + suffix
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
	const digits = "0123456789abcdef"
	var b strings.Builder
	for _, v := range buf {
		b.WriteByte(digits[int(v)>>4])
		b.WriteByte(digits[int(v)&0x0f])
	}
	return b.String(), nil
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

func runMigrations(repoRoot string, db *sql.DB) error {
	migrationsDir := filepath.Join(repoRoot, "code/components/data/migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
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
		return fmt.Errorf("no migrations found in %s", migrationsDir)
	}
	sort.Strings(files)
	for _, name := range files {
		schemaPath := filepath.Join(migrationsDir, name)
		schema, err := os.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.Exec(string(schema)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
