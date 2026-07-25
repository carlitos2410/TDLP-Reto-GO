package supervisor

import (
	"sync"
	"time"
)

// ProcessState representa el estado actual de un proceso supervisado.
type ProcessState int

const (
	StateIdle       ProcessState = iota // no iniciado
	StateStarting                       // proceso arrancando
	StateRunning                        // proceso en ejecución
	StateStopping                       // deteniendo con periodo de gracia
	StateStopped                        // detenido intencionalmente
	StateBackingOff                     // esperando antes de reiniciar
	StateFailed                         // fallo y no se reiniciará
)

// String retorna una representación legible del estado.
func (s ProcessState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	case StateBackingOff:
		return "backing-off"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ProcessInfo contiene el estado y metadatos de un proceso supervisado.
type ProcessInfo struct {
	Name         string
	State        ProcessState
	Retries      int
	LastExitCode int
	StartedAt    time.Time
	StoppedAt    time.Time
	Error        error
}

// StateTracker mantiene el estado de todos los procesos supervisados.
type StateTracker struct {
	mu      sync.RWMutex
	processes map[string]*ProcessInfo
}

// NewStateTracker crea un tracker vacío.
func NewStateTracker() *StateTracker {
	return &StateTracker{
		processes: make(map[string]*ProcessInfo),
	}
}

// Register registra un proceso en el tracker.
func (st *StateTracker) Register(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.processes[name] = &ProcessInfo{
		Name:  name,
		State: StateIdle,
	}
}

// SetState actualiza el estado de un proceso.
func (st *StateTracker) SetState(name string, state ProcessState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if p, ok := st.processes[name]; ok {
		p.State = state
		now := time.Now()
		switch state {
		case StateRunning:
			p.StartedAt = now
		case StateStopped, StateFailed:
			p.StoppedAt = now
		}
	}
}

// SetError guarda un error y marca el proceso como fallido.
func (st *StateTracker) SetError(name string, err error, exitCode int) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if p, ok := st.processes[name]; ok {
		p.Error = err
		p.LastExitCode = exitCode
		p.State = StateFailed
		p.StoppedAt = time.Now()
	}
}

// IncrementRetries incrementa el contador de reintentos de un proceso.
func (st *StateTracker) IncrementRetries(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if p, ok := st.processes[name]; ok {
		p.Retries++
	}
}

// ResetRetries reinicia el contador de reintentos de un proceso.
func (st *StateTracker) ResetRetries(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if p, ok := st.processes[name]; ok {
		p.Retries = 0
	}
}

// Get retorna la información de un proceso.
func (st *StateTracker) Get(name string) (ProcessInfo, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	p, ok := st.processes[name]
	if !ok {
		return ProcessInfo{}, false
	}
	return *p, true
}

// GetAll retorna el estado de todos los procesos registrados.
func (st *StateTracker) GetAll() map[string]ProcessInfo {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := make(map[string]ProcessInfo, len(st.processes))
	for name, p := range st.processes {
		out[name] = *p
	}
	return out
}

// Summary retorna un map nombre→estado legible de todos los procesos.
func (st *StateTracker) Summary() map[string]string {
	st.mu.RLock()
	defer st.mu.RUnlock()
	out := make(map[string]string, len(st.processes))
	for name, p := range st.processes {
		out[name] = p.State.String()
	}
	return out
}

// RunningCount retorna la cantidad de procesos en estado Running.
func (st *StateTracker) RunningCount() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	count := 0
	for _, p := range st.processes {
		if p.State == StateRunning {
			count++
		}
	}
	return count
}

// FailedCount retorna la cantidad de procesos en estado Failed.
func (st *StateTracker) FailedCount() int {
	st.mu.RLock()
	defer st.mu.RUnlock()
	count := 0
	for _, p := range st.processes {
		if p.State == StateFailed {
			count++
		}
	}
	return count
}
