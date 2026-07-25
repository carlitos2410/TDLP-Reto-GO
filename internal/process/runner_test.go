package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"supervisor-procesos/internal/config"
)

func TestRunnerCapturesStdout(t *testing.T) {
	dir := t.TempDir()
	stdoutLog := filepath.Join(dir, "demo.stdout.log")
	stderrLog := filepath.Join(dir, "demo.stderr.log")

	command, args := echoCommand("hola-supervisor")
	runner := NewRunner(config.ProcessConfig{
		Name:      "demo",
		Command:   command,
		Args:      args,
		StdoutLog: stdoutLog,
		StderrLog: stderrLog,
	}, 100*time.Millisecond)

	result := runner.RunOnce(context.Background())
	if result.Err != nil {
		t.Fatalf("RunOnce() error = %v", result.Err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}

	data, err := os.ReadFile(stdoutLog)
	if err != nil {
		t.Fatalf("leer stdout log: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("stdout log está vacío")
	}
}

func TestRunOnceRespectsContextCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sleep no interrumpible de forma fiable en Windows para esta prueba")
	}

	dir := t.TempDir()
	runner := NewRunner(config.ProcessConfig{
		Name:      "sleepy",
		Command:   "sleep",
		Args:      []string{"10"},
		StdoutLog: filepath.Join(dir, "sleepy.stdout.log"),
		StderrLog: filepath.Join(dir, "sleepy.stderr.log"),
	}, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan RunResult, 1)
	go func() {
		done <- runner.RunOnce(ctx)
	}()

	cancel()
	result := <-done
	if result.Err == nil {
		t.Fatal("se esperaba error por cancelación de contexto")
	}
	if !errors.Is(result.Err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", result.Err)
	}
}

func echoCommand(message string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", "echo", message}
	}
	return "echo", []string{message}
}
