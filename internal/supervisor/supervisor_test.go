package supervisor

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"supervisor-procesos/internal/config"
)

func TestSupervisorRunsAllWorkers(t *testing.T) {
	dir := t.TempDir()
	command, args := testEchoCommand()

	cfg := &config.Config{
		Processes: []config.ProcessConfig{
			{
				Name:          "a",
				Command:       command,
				Args:          args,
				StdoutLog:     filepath.Join(dir, "a.stdout.log"),
				StderrLog:     filepath.Join(dir, "a.stderr.log"),
				RestartPolicy: config.RestartNever,
			},
			{
				Name:          "b",
				Command:       command,
				Args:          args,
				StdoutLog:     filepath.Join(dir, "b.stdout.log"),
				StderrLog:     filepath.Join(dir, "b.stderr.log"),
				RestartPolicy: config.RestartNever,
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	sup := New(cfg)

	done := make(chan struct{})
	go func() {
		sup.Run(ctx)
		close(done)
	}()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(sup.Status()) == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("supervisor no terminó tras apagado")
	}
}

func TestSupervisorReloadAddsProcess(t *testing.T) {
	dir := t.TempDir()
	command, args := testEchoCommand()

	cfg := &config.Config{
		Processes: []config.ProcessConfig{
			{
				Name:          "a",
				Command:       command,
				Args:          args,
				StdoutLog:     filepath.Join(dir, "a.stdout.log"),
				StderrLog:     filepath.Join(dir, "a.stderr.log"),
				RestartPolicy: config.RestartAlways,
				Backoff: config.BackoffConfig{
					InitialSeconds: 1,
					Factor:         1,
					MaxSeconds:     1,
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sup := New(cfg)
	go sup.Run(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if len(sup.Status()) == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	reloaded := &config.Config{
		Processes: []config.ProcessConfig{
			cfg.Processes[0],
			{
				Name:          "b",
				Command:       command,
				Args:          args,
				StdoutLog:     filepath.Join(dir, "b.stdout.log"),
				StderrLog:     filepath.Join(dir, "b.stderr.log"),
				RestartPolicy: config.RestartNever,
			},
		},
	}

	if err := sup.Reload(reloaded); err != nil {
		t.Fatalf("Reload() error = %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status := sup.Status()
		if len(status) == 2 {
			cancel()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Status() = %d procesos, want 2 tras recarga", len(sup.Status()))
}

func TestSupervisorShutdownWaitsForWorkers(t *testing.T) {
	dir := t.TempDir()
	command, args := testEchoCommand()

	cfg := &config.Config{
		GracePeriodSeconds: 0.05,
		Processes: []config.ProcessConfig{
			{
				Name:          "long",
				Command:       command,
				Args:          args,
				StdoutLog:     filepath.Join(dir, "long.stdout.log"),
				StderrLog:     filepath.Join(dir, "long.stderr.log"),
				RestartPolicy: config.RestartAlways,
				Backoff: config.BackoffConfig{
					InitialSeconds: 5,
					Factor:         1,
					MaxSeconds:     5,
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		New(cfg).Run(ctx)
		close(done)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("supervisor no completó apagado tras cancelar contexto")
	}
}

func TestSupervisorStartStopRestart(t *testing.T) {
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
				RestartPolicy: config.RestartAlways,
				Backoff: config.BackoffConfig{
					InitialSeconds: 1,
					Factor:         1,
					MaxSeconds:     1,
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sup := New(cfg)
	go sup.Run(ctx)

	time.Sleep(100 * time.Millisecond)

	if err := sup.StopProcess("demo"); err != nil {
		t.Fatalf("StopProcess() error = %v", err)
	}

	if err := sup.StartProcess("demo"); err != nil {
		t.Fatalf("StartProcess() error = %v", err)
	}

	if err := sup.RestartProcess("demo"); err != nil {
		t.Fatalf("RestartProcess() error = %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	status, ok := sup.StatusOf("demo")
	if !ok {
		t.Fatal("StatusOf() no encontró demo")
	}
	if status.State != StateRunning && status.State != StateBackoff {
		t.Fatalf("State = %q, want running o backoff", status.State)
	}
}
