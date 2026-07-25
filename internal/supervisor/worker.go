package supervisor

import (
	"context"
	"log"
	"sync"
	"time"

	"supervisor-procesos/internal/config"
	"supervisor-procesos/internal/process"
)

// Worker administra el ciclo de vida de un proceso hijo supervisado.
type Worker struct {
	cfg         config.ProcessConfig
	gracePeriod time.Duration
	runner      *process.Runner
	backoff     *Backoff
	tracker     *StateTracker

	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	lastErr error
}

// NewWorker crea un worker listo para gestionar el proceso descrito en la configuración.
func NewWorker(cfg config.ProcessConfig, gracePeriod time.Duration, b *Backoff, tracker *StateTracker) *Worker {
	tracker.Register(cfg.Name)
	return &Worker{
		cfg:         cfg,
		gracePeriod: gracePeriod,
		runner:      process.NewRunner(cfg, gracePeriod),
		backoff:     b,
		tracker:     tracker,
		done:        make(chan struct{}),
	}
}

// Name retorna el nombre del proceso gestionado por este worker.
func (w *Worker) Name() string {
	return w.cfg.Name
}

// Start inicia el loop de supervisión del worker en una goroutine.
// El loop ejecuta el proceso, y si termina con error según la política de reinicio,
// aplica backoff y reintenta hasta agotar reintentos o cancelar el contexto principal.
func (w *Worker) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cancel = cancel
	w.mu.Unlock()

	go w.loop(ctx)
}

// Stop detiene el worker de forma ordenada.
func (w *Worker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	<-w.done
}

// LastErr retorna el último error registrado, o nil si no hubo errores.
func (w *Worker) LastErr() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastErr
}

// Status retorna la información actual del proceso desde el StateTracker.
func (w *Worker) Status() ProcessInfo {
	info, ok := w.tracker.Get(w.cfg.Name)
	if !ok {
		return ProcessInfo{Name: w.cfg.Name, State: StateIdle}
	}
	return info
}

// Config retorna la configuración del proceso gestionado por este worker.
func (w *Worker) Config() config.ProcessConfig {
	return w.cfg
}

// loop es el ciclo de vida principal del worker.
func (w *Worker) loop(ctx context.Context) {
	defer close(w.done)

	for {
		w.tracker.SetState(w.cfg.Name, StateRunning)
		log.Printf("[%s] iniciando proceso", w.cfg.Name)

		result := w.runner.RunOnce(ctx)

		w.mu.Lock()
		w.lastErr = result.Err
		w.mu.Unlock()

		w.tracker.SetState(w.cfg.Name, StateIdle)
		log.Printf("[%s] proceso terminó con exit code %d", w.cfg.Name, result.ExitCode)

		if ctx.Err() != nil {
			w.tracker.SetState(w.cfg.Name, StateStopped)
			log.Printf("[%s] supervisor cancelado, deteniendo", w.cfg.Name)
			return
		}

		if result.ExitCode == 0 && w.cfg.RestartPolicy == config.RestartOnFailure {
			w.tracker.SetState(w.cfg.Name, StateStopped)
			w.backoff.Reset(w.cfg.Name)
			log.Printf("[%s] terminó exitosamente (on-failure), no se reinicia", w.cfg.Name)
			return
		}

		if !w.cfg.ShouldRestart(w.backoff.CurrentAttempt(w.cfg.Name)) {
			w.tracker.SetError(w.cfg.Name, result.Err, result.ExitCode)
			log.Printf("[%s] sin reintentos restantes, marcado como fallido", w.cfg.Name)
			return
		}

		w.tracker.IncrementRetries(w.cfg.Name)
		waitDur := w.backoff.Next(w.cfg.Name)
		w.tracker.SetState(w.cfg.Name, StateBackingOff)
		log.Printf("[%s] reintento %d, esperando %v antes de reiniciar",
			w.cfg.Name, w.backoff.CurrentAttempt(w.cfg.Name), waitDur)

		select {
		case <-ctx.Done():
			w.tracker.SetState(w.cfg.Name, StateStopped)
			log.Printf("[%s] supervisor cancelado durante backoff", w.cfg.Name)
			return
		case <-time.After(waitDur):
		}
	}
}
