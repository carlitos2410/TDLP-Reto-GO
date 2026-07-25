package supervisor

import (
	"sync"
	"time"

	"supervisor-procesos/internal/config"
)

// Backoff calcula tiempos de espera exponenciales para reintentos de procesos.
type Backoff struct {
	cfg    config.BackoffConfig
	mu     sync.Mutex
	attempt map[string]int // intentos por nombre de proceso
}

// NewBackoff crea un calculador de backoff con la configuración dada.
func NewBackoff(cfg config.BackoffConfig) *Backoff {
	return &Backoff{
		cfg:     cfg,
		attempt: make(map[string]int),
	}
}

// Next calcula y retorna el tiempo de espera para el próximo reinicio del proceso.
// Incrementa internamente el contador de intentos.
func (b *Backoff) Next(processName string) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	attempt := b.attempt[processName]
	dur := b.cfg.Duration(attempt)
	b.attempt[processName] = attempt + 1
	return dur
}

// CurrentAttempt retorna el número de intento actual para un proceso sin incrementarlo.
func (b *Backoff) CurrentAttempt(processName string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.attempt[processName]
}

// Reset reinicia el contador de intentos de un proceso.
func (b *Backoff) Reset(processName string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.attempt[processName] = 0
}

// ShouldStop determina si se debe dejar de reintentar un proceso
// según su configuración de max_retries.
func (b *Backoff) ShouldStop(cfg config.ProcessConfig) bool {
	if cfg.MaxRetries <= 0 {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.attempt[cfg.Name] >= cfg.MaxRetries
}
