package supervisor

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"supervisor-procesos/internal/config"
	"supervisor-procesos/internal/process"
)

// Worker supervisa un único proceso hijo, aplica política de reinicio y backoff.
type Worker struct {
	runner      *process.Runner
	policy      config.RestartPolicy
	backoff     *BackoffScheduler
	maxRetries  int
	gracePeriod time.Duration

	mu     sync.RWMutex
	status ProcessStatus
}

// NewWorker crea un worker a partir de la configuración de un proceso.
func NewWorker(cfg config.ProcessConfig, gracePeriod time.Duration) *Worker {
	return &Worker{
		runner:      process.NewRunner(cfg, gracePeriod),
		policy:      cfg.RestartPolicy,
		backoff:     NewBackoffScheduler(cfg.Backoff),
		maxRetries:  cfg.MaxRetries,
		gracePeriod: gracePeriod,
		status: ProcessStatus{
			Name:  cfg.Name,
			State: StateStopped,
		},
	}
}

// Name devuelve el nombre del proceso supervisado.
func (w *Worker) Name() string {
	return w.runner.Name()
}

// Status devuelve una copia thread-safe del estado actual del proceso.
func (w *Worker) Status() ProcessStatus {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.status
}

// Run ejecuta el ciclo de supervisión hasta stopped, failed o cancelación del contexto.
func (w *Worker) Run(ctx context.Context) {
	name := w.runner.Name()

	for {
		if err := ctx.Err(); err != nil {
			w.setState(StateStopped)
			log.Printf("[%s] supervisión detenida: %v", name, err)
			return
		}

		w.setState(StateRunning)
		log.Printf("[%s] arrancando: %s", name, w.runner.Config().Command)
		result := w.runner.RunOnce(ctx)
		w.setLastExitCode(result.ExitCode)

		if errors.Is(result.Err, context.Canceled) || errors.Is(result.Err, context.DeadlineExceeded) {
			w.setState(StateStopped)
			log.Printf("[%s] ejecución cancelada por contexto", name)
			return
		}

		success := result.ExitCode == 0 && result.Err == nil
		if success {
			w.backoff.OnSuccess()
			w.resetConsecutiveFailures()
			log.Printf("[%s] terminó correctamente (código=0)", name)
		} else {
			w.backoff.OnFailure()
			failures := w.incrementConsecutiveFailures()
			log.Printf("[%s] terminó con código=%d: %v", name, result.ExitCode, result.Err)

			if w.maxRetries > 0 && failures > w.maxRetries {
				w.setState(StateFailed)
				log.Printf("[%s] estado failed: superó max_retries=%d", name, w.maxRetries)
				return
			}
		}

		if !shouldRestart(w.policy, result) {
			w.setState(StateStopped)
			log.Printf("[%s] no se reinicia (política=%s)", name, w.policy)
			return
		}

		delay := w.backoff.Delay()
		w.setState(StateBackoff)
		log.Printf("[%s] backoff %s antes de reiniciar (política=%s)", name, delay, w.policy)

		if !w.waitBackoff(ctx, delay) {
			w.setState(StateStopped)
			log.Printf("[%s] backoff interrumpido por contexto", name)
			return
		}

		w.incrementRestartCount()
		log.Printf("[%s] reiniciando tras backoff", name)
	}
}

func (w *Worker) waitBackoff(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (w *Worker) setState(state State) {
	w.mu.Lock()
	w.status.State = state
	w.mu.Unlock()
}

func (w *Worker) setLastExitCode(code int) {
	w.mu.Lock()
	w.status.LastExitCode = code
	w.mu.Unlock()
}

func (w *Worker) incrementRestartCount() {
	w.mu.Lock()
	w.status.RestartCount++
	w.mu.Unlock()
}

func (w *Worker) incrementConsecutiveFailures() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.status.ConsecutiveFailures++
	return w.status.ConsecutiveFailures
}

func (w *Worker) resetConsecutiveFailures() {
	w.mu.Lock()
	w.status.ConsecutiveFailures = 0
	w.mu.Unlock()
}

func shouldRestart(policy config.RestartPolicy, result process.RunResult) bool {
	switch policy {
	case config.RestartAlways:
		return true
	case config.RestartOnFailure:
		return result.ExitCode != 0
	case config.RestartNever:
		return false
	default:
		return false
	}
}
