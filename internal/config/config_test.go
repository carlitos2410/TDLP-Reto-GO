package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "supervisor.yaml")
	content := `
log_dir: logs
processes:
  - name: worker-a
    command: echo
    args: ["hola"]
  - name: worker-b
    command: echo
    args: ["mundo"]
    work_dir: .
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir config temporal: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Processes) != 2 {
		t.Fatalf("esperaba 2 procesos, obtuve %d", len(cfg.Processes))
	}
	if cfg.Processes[0].StdoutLog != filepath.Join("logs", "worker-a.stdout.log") {
		t.Fatalf("stdout_log por defecto inesperado: %q", cfg.Processes[0].StdoutLog)
	}
}

func TestLoadRejectsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "supervisor.yaml")
	content := `
processes:
  - name: dup
    command: echo
  - name: dup
    command: echo
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir config temporal: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() debería fallar con nombres duplicados")
	}
}

func TestLoadRejectsEmptyProcesses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, []byte("processes: []\n"), 0o644); err != nil {
		t.Fatalf("escribir config temporal: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() debería fallar sin procesos")
	}
}

func TestLoadParsesRestartPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "supervisor.yaml")
	content := `
processes:
  - name: worker
    command: echo
    restart_policy: always
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir config temporal: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Processes[0].RestartPolicy != RestartAlways {
		t.Fatalf("RestartPolicy = %q, want %q", cfg.Processes[0].RestartPolicy, RestartAlways)
	}
}

func TestLoadRejectsInvalidRestartPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "supervisor.yaml")
	content := `
processes:
  - name: worker
    command: echo
    restart_policy: sometimes
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("escribir config temporal: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("Load() debería fallar con política inválida")
	}
}

func TestShouldRestartAlways(t *testing.T) {
	pc := ProcessConfig{RestartPolicy: RestartAlways, MaxRetries: 0}
	if !pc.ShouldRestart(0) {
		t.Fatal("ShouldRestart(0) = false con always e ilimitado")
	}
	if !pc.ShouldRestart(100) {
		t.Fatal("ShouldRestart(100) = false con always e ilimitado")
	}
}

func TestShouldRestartAlwaysWithLimit(t *testing.T) {
	pc := ProcessConfig{RestartPolicy: RestartAlways, MaxRetries: 3}
	if !pc.ShouldRestart(0) {
		t.Fatal("ShouldRestart(0) = false, want true")
	}
	if !pc.ShouldRestart(2) {
		t.Fatal("ShouldRestart(2) = false, want true")
	}
	if pc.ShouldRestart(3) {
		t.Fatal("ShouldRestart(3) = true, want false (agotado)")
	}
}

func TestShouldRestartOnFailure(t *testing.T) {
	pc := ProcessConfig{RestartPolicy: RestartOnFailure, MaxRetries: 2}
	if !pc.ShouldRestart(0) {
		t.Fatal("ShouldRestart(0) = false")
	}
	if !pc.ShouldRestart(1) {
		t.Fatal("ShouldRestart(1) = false")
	}
	if pc.ShouldRestart(2) {
		t.Fatal("ShouldRestart(2) = true, want false")
	}
}

func TestShouldRestartNever(t *testing.T) {
	pc := ProcessConfig{RestartPolicy: RestartNever, MaxRetries: 0}
	if pc.ShouldRestart(0) {
		t.Fatal("ShouldRestart(0) = true con never, want false")
	}
}

func TestShouldRestartZeroRetriesUnlimited(t *testing.T) {
	pc := ProcessConfig{RestartPolicy: RestartOnFailure, MaxRetries: 0}
	for i := 0; i < 50; i++ {
		if !pc.ShouldRestart(i) {
			t.Fatalf("ShouldRestart(%d) = false con MaxRetries=0 (ilimitado)", i)
		}
	}
}
