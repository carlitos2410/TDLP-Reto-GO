package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"supervisor-procesos/internal/config"
	"supervisor-procesos/internal/supervisor"
)

func TestAPIListProcesses(t *testing.T) {
	sup, cancel := testSupervisor(t, config.RestartNever)
	defer cancel()

	server := NewServer(sup, ":0")
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/processes", server.handleProcesses)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/processes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var list []statusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("procesos = %d, want 1", len(list))
	}
}

func TestAPIStopAndStart(t *testing.T) {
	sup, cancel := testSupervisor(t, config.RestartAlways)
	defer cancel()

	server := NewServer(sup, ":0")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		server.handleProcessByName(w, r)
	})

	time.Sleep(100 * time.Millisecond)

	stopReq := httptest.NewRequest(http.MethodPost, "/api/v1/processes/demo/stop", nil)
	stopRec := httptest.NewRecorder()
	handler.ServeHTTP(stopRec, stopReq)
	if stopRec.Code != http.StatusOK {
		t.Fatalf("stop status = %d, body = %s", stopRec.Code, stopRec.Body.String())
	}

	time.Sleep(50 * time.Millisecond)

	startReq := httptest.NewRequest(http.MethodPost, "/api/v1/processes/demo/start", nil)
	startRec := httptest.NewRecorder()
	handler.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, body = %s", startRec.Code, startRec.Body.String())
	}
}

func TestAPIRestart(t *testing.T) {
	sup, cancel := testSupervisor(t, config.RestartAlways)
	defer cancel()

	server := NewServer(sup, ":0")
	handler := http.HandlerFunc(server.handleProcessByName)

	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/processes/demo/restart", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("restart status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestAPIUnknownProcess(t *testing.T) {
	sup, cancel := testSupervisor(t, config.RestartNever)
	defer cancel()

	server := NewServer(sup, ":0")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/processes/inexistente", nil)
	rec := httptest.NewRecorder()
	server.handleProcessByName(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestAPIStartAlreadyRunning(t *testing.T) {
	sup, cancel := testSupervisor(t, config.RestartAlways)
	defer cancel()

	server := NewServer(sup, ":0")
	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/processes/demo/start", nil)
	rec := httptest.NewRecorder()
	server.handleProcessByName(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func testSupervisor(t *testing.T, policy config.RestartPolicy) (*supervisor.Supervisor, context.CancelFunc) {
	t.Helper()

	dir := t.TempDir()
	command, args := testEchoCommand()

	cfg := &config.Config{
		GracePeriodSeconds: 0.05,
		Processes: []config.ProcessConfig{
			{
				Name:          "demo",
				Command:       command,
				Args:          args,
				StdoutLog:     filepath.Join(dir, "demo.stdout.log"),
				StderrLog:     filepath.Join(dir, "demo.stderr.log"),
				RestartPolicy: policy,
				Backoff: config.BackoffConfig{
					InitialSeconds: 1,
					Factor:         1,
					MaxSeconds:     1,
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	sup := supervisor.New(cfg)
	go sup.Run(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sup.Status()) > 0 {
			return sup, cancel
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("supervisor no arrancó el worker a tiempo")
	return nil, cancel
}

func testEchoCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo", "hola"}
	}
	return "echo", []string{"hola"}
}
