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
