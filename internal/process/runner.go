package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"supervisor-procesos/internal/config"
)

// RunResult resume el resultado de una ejecución del proceso hijo.
type RunResult struct {
	ExitCode int
	Err      error
}

// Runner lanza un proceso hijo y redirige su salida a archivos de log.
type Runner struct {
	cfg         config.ProcessConfig
	gracePeriod time.Duration
}

// NewRunner crea un runner listo para ejecutar el proceso descrito en la configuración.
func NewRunner(cfg config.ProcessConfig, gracePeriod time.Duration) *Runner {
	return &Runner{cfg: cfg, gracePeriod: gracePeriod}
}

// Name devuelve el identificador del proceso supervisado.
func (r *Runner) Name() string {
	return r.cfg.Name
}

// Config devuelve la configuración asociada al runner.
func (r *Runner) Config() config.ProcessConfig {
	return r.cfg
}

// RunOnce inicia el proceso una vez, captura stdout/stderr y espera su terminación.
// Si el contexto se cancela, envía señal de terminación, espera el periodo de gracia
// y fuerza SIGKILL si el hijo sigue vivo.
func (r *Runner) RunOnce(ctx context.Context) RunResult {
	if err := ensureLogDir(r.cfg.StdoutLog); err != nil {
		return RunResult{ExitCode: -1, Err: fmt.Errorf("proceso %q: preparar log stdout: %w", r.cfg.Name, err)}
	}
	if err := ensureLogDir(r.cfg.StderrLog); err != nil {
		return RunResult{ExitCode: -1, Err: fmt.Errorf("proceso %q: preparar log stderr: %w", r.cfg.Name, err)}
	}

	stdoutFile, err := os.OpenFile(r.cfg.StdoutLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return RunResult{ExitCode: -1, Err: fmt.Errorf("proceso %q: abrir stdout log: %w", r.cfg.Name, err)}
	}
	defer stdoutFile.Close()

	stderrFile, err := os.OpenFile(r.cfg.StderrLog, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return RunResult{ExitCode: -1, Err: fmt.Errorf("proceso %q: abrir stderr log: %w", r.cfg.Name, err)}
	}
	defer stderrFile.Close()

	cmd := exec.CommandContext(ctx, r.cfg.Command, r.cfg.Args...)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile
	cmd.Env = buildEnv(r.cfg.Env)

	if r.cfg.WorkDir != "" {
		cmd.Dir = r.cfg.WorkDir
	}

	if err := cmd.Start(); err != nil {
		return RunResult{ExitCode: -1, Err: fmt.Errorf("proceso %q: arrancar: %w", r.cfg.Name, err)}
	}

	waitDone := make(chan RunResult, 1)
	go func() {
		waitErr := cmd.Wait()
		waitDone <- RunResult{ExitCode: exitCode(waitErr), Err: waitErr}
	}()

	select {
	case <-ctx.Done():
		return r.shutdownChild(cmd, waitDone, ctx.Err())
	case result := <-waitDone:
		if result.Err != nil && result.ExitCode == 0 {
			result.ExitCode = -1
		}
		return result
	}
}

func (r *Runner) shutdownChild(cmd *exec.Cmd, waitDone <-chan RunResult, ctxErr error) RunResult {
	if cmd.Process != nil {
		_ = sendTerminateSignal(cmd.Process)
	}

	graceTimer := time.NewTimer(r.gracePeriod)
	defer graceTimer.Stop()

	select {
	case result := <-waitDone:
		result.Err = ctxErr
		return result
	case <-graceTimer.C:
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		result := <-waitDone
		result.Err = ctxErr
		return result
	}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}