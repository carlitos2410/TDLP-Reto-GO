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
