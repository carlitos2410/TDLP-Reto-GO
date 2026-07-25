package supervisor

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"supervisor-procesos/internal/config"
	"supervisor-procesos/internal/process"
)

const testGracePeriod = 50 * time.Millisecond

func TestShouldRestartPolicies(t *testing.T) {
	t.Parallel()

	success := process.RunResult{ExitCode: 0}
	failure := process.RunResult{ExitCode: 1}

	cases := []struct {
		name   string
		policy config.RestartPolicy
		result process.RunResult
		want   bool
	}{
		{"never on success", config.RestartNever, success, false},
		{"never on failure", config.RestartNever, failure, false},
		{"always on success", config.RestartAlways, success, true},
		{"always on failure", config.RestartAlways, failure, true},
		{"on-failure success", config.RestartOnFailure, success, false},
		{"on-failure failure", config.RestartOnFailure, failure, true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldRestart(tc.policy, tc.result); got != tc.want {
				t.Fatalf("shouldRestart() = %v, want %v", got, tc.want)
			}
		})
	}
}


func TestWorkerNeverRestartsAfterSuccess(t *testing.T) {
	dir := t.TempDir()
	command, args := testEchoCommand()

	worker := NewWorker(config.ProcessConfig{
		Name:          "once",
		Command:       command,
		Args:          args,
		StdoutLog:     filepath.Join(dir, "once.stdout.log"),
		StderrLog:     filepath.Join(dir, "once.stderr.log"),
		RestartPolicy: config.RestartNever,
	}, testGracePeriod)

	done := make(chan struct{})
	go func() {
		worker.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker no terminó con política never")
	}
}

func TestWorkerRestartsOnFailureUntilContextCancelled(t *testing.T) {
	dir := t.TempDir()
	command, args := testFailingCommand()

	worker := NewWorker(config.ProcessConfig{
		Name:          "flaky",
		Command:       command,
		Args:          args,
		StdoutLog:     filepath.Join(dir, "flaky.stdout.log"),
		StderrLog:     filepath.Join(dir, "flaky.stderr.log"),
		RestartPolicy: config.RestartOnFailure,
	}, testGracePeriod)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	worker.Run(ctx)
}

func TestWorkerAlwaysRestartsUntilContextCancelled(t *testing.T) {
	dir := t.TempDir()
	command, args := testEchoCommand()

	worker := NewWorker(config.ProcessConfig{
		Name:          "loop",
		Command:       command,
		Args:          args,
		StdoutLog:     filepath.Join(dir, "loop.stdout.log"),
		StderrLog:     filepath.Join(dir, "loop.stderr.log"),
		RestartPolicy: config.RestartAlways,
	}, testGracePeriod)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	worker.Run(ctx)
}

func TestWorkerEntersFailedAfterMaxRetries(t *testing.T) {
	dir := t.TempDir()
	command, args := testFailingCommand()

	worker := NewWorker(config.ProcessConfig{
		Name:          "limited",
		Command:       command,
		Args:          args,
		StdoutLog:     filepath.Join(dir, "limited.stdout.log"),
		StderrLog:     filepath.Join(dir, "limited.stderr.log"),
		RestartPolicy: config.RestartOnFailure,
		MaxRetries:    1,
		Backoff: config.BackoffConfig{
			InitialSeconds: 0.01,
			Factor:         1,
			MaxSeconds:     0.05,
		},
	}, testGracePeriod)

	done := make(chan struct{})
	go func() {
		worker.Run(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("worker no terminó tras agotar reintentos")
	}

	status := worker.Status()
	if status.State != StateFailed {
		t.Fatalf("State = %q, want %q", status.State, StateFailed)
	}
	if status.ConsecutiveFailures <= 1 {
		t.Fatalf("ConsecutiveFailures = %d, want > 1", status.ConsecutiveFailures)
	}
}

func TestWorkerAppliesBackoffBeforeRestart(t *testing.T) {
	dir := t.TempDir()
	command, args := testEchoCommand()

	worker := NewWorker(config.ProcessConfig{
		Name:          "delayed",
		Command:       command,
		Args:          args,
		StdoutLog:     filepath.Join(dir, "delayed.stdout.log"),
		StderrLog:     filepath.Join(dir, "delayed.stderr.log"),
		RestartPolicy: config.RestartAlways,
		Backoff: config.BackoffConfig{
			InitialSeconds: 0.2,
			Factor:         1,
			MaxSeconds:     0.2,
		},
	}, testGracePeriod)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	start := time.Now()
	worker.Run(ctx)
	elapsed := time.Since(start)

	if elapsed < 200*time.Millisecond {
		t.Fatalf("elapsed = %s, se esperaba al menos un backoff de ~200ms", elapsed)
	}
}