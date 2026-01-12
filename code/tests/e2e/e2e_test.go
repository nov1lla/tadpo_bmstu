//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"ppo/data/testsupport"
)

func TestWebappE2EFlow(t *testing.T) {
	repoRoot := projectRoot(t)
	buildDir := t.TempDir()
	staticDir := filepath.Join(buildDir, "static")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatalf("create static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>ok</html>"), 0o644); err != nil {
		t.Fatalf("write static: %v", err)
	}

	dataPlugin := filepath.Join(buildDir, "data.so")
	businessPlugin := filepath.Join(buildDir, "business.so")
	webappBin := filepath.Join(buildDir, "webapp")

	dataDir := filepath.Join(repoRoot, "code/components/data")
	businessDir := filepath.Join(repoRoot, "code/components/business")
	webappDir := filepath.Join(repoRoot, "code/apps/webapp")

	runCmd(t, dataDir, "go", "build", "-tags", "plugin", "-buildmode=plugin", "-o", dataPlugin, "./cmd/plugin")
	runCmd(t, businessDir, "go", "build", "-tags", "plugin", "-buildmode=plugin", "-o", businessPlugin, "./cmd/plugin")
	runCmd(t, webappDir, "go", "build", "-o", webappBin, "./cmd/webapp")

	env := testsupport.SetupPostgres(t)
	defer env.Cleanup()

	addr := freeAddr(t)
	server := exec.Command(webappBin)
	server.Env = append(os.Environ(),
		"WEBAPP_ADDR="+addr,
		"DATA_PLUGIN_PATH="+dataPlugin,
		"BUSINESS_PLUGIN_PATH="+businessPlugin,
		"DATA_SOURCE="+env.DSN,
		"WEBAPP_STATIC_DIR="+staticDir,
	)
	server.Stdout = os.Stdout
	server.Stderr = os.Stderr

	if err := server.Start(); err != nil {
		t.Fatalf("start webapp: %v", err)
	}
	defer func() {
		_ = server.Process.Signal(os.Interrupt)
		_, _ = server.Process.Wait()
	}()

	baseURL := "http://" + addr
	waitForServer(t, baseURL)

	user := postJSON(t, baseURL+"/api/users", map[string]string{"name": "E2E User"})
	userID := getString(t, user, "id")

	getJSON(t, baseURL+"/api/users/"+userID)

	gameResp := postJSON(t, baseURL+"/api/games", map[string]interface{}{
		"userId":        userID,
		"playerColor":   "light",
		"isPlayerFirst": true,
	})
	game := getMap(t, gameResp, "game")
	gameID := getString(t, game, "id")

	moveReq := map[string]interface{}{
		"start": map[string]int{"row": 5, "col": 0},
		"steps": []map[string]int{{"row": 4, "col": 1}},
	}
	postJSON(t, baseURL+"/api/games/"+gameID+"/moves", moveReq)

	moves := getJSONArray(t, baseURL+"/api/games/"+gameID+"/moves")
	if len(moves) != 1 {
		t.Fatalf("expected 1 move, got %d", len(moves))
	}

	req, err := http.NewRequest(http.MethodDelete, baseURL+"/api/games/"+gameID, nil)
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete game: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected delete status: %d", resp.StatusCode)
	}
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run %s: %v", strings.Join(append([]string{name}, args...), " "), err)
	}
}

func waitForServer(t *testing.T, baseURL string) {
	t.Helper()
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
	t.Fatalf("server did not start")
}

func postJSON(t *testing.T, url string, payload interface{}) map[string]interface{} {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("unexpected status %d for %s", resp.StatusCode, url)
	}
	var decoded map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func getJSON(t *testing.T, url string) map[string]interface{} {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("unexpected status %d for %s", resp.StatusCode, url)
	}
	var decoded map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func getJSONArray(t *testing.T, url string) []interface{} {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("unexpected status %d for %s", resp.StatusCode, url)
	}
	var decoded []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return decoded
}

func getString(t *testing.T, data map[string]interface{}, key string) string {
	t.Helper()
	value, ok := data[key]
	if !ok {
		t.Fatalf("missing key %s", key)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("key %s is not a string", key)
	}
	return text
}

func getMap(t *testing.T, data map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	value, ok := data[key]
	if !ok {
		t.Fatalf("missing key %s", key)
	}
	mapped, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("key %s is not a map", key)
	}
	return mapped
}

func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

func projectRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("cannot resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
