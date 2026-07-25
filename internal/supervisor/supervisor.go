package supervisor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"supervisor-procesos/internal/config"
)

type managedWorker struct {
	worker *Worker
	cancel context.CancelFunc
	cfg    config.ProcessConfig
	done   chan struct{}
}

// Supervisor coordina todos los workers que gestionan los procesos hijos.
type Supervisor struct {
	cfg         *config.Config
	gracePeriod time.Duration

	mu      sync.RWMutex
	workers map[string]*managedWorker
	rootCtx context.Context
	wg      sync.WaitGroup
}

// New crea un supervisor a partir de la configuración cargada.
func New(cfg *config.Config) *Supervisor {
	return &Supervisor{
		cfg:         cfg,
		gracePeriod: cfg.GracePeriod(),
		workers:     make(map[string]*managedWorker),
	}
}

// Run lanza los workers iniciales y espera señal de apagado por contexto.
func (s *Supervisor) Run(ctx context.Context) {
	s.rootCtx = ctx

	for _, procCfg := range s.cfg.Processes {
		s.startWorker(procCfg)
	}

	<-ctx.Done()
	log.Printf("supervisor: apagado solicitado, deteniendo procesos hijos")
	s.stopAllWorkers()
	s.wg.Wait()
}

// Reload aplica una configuración recargada sin reiniciar el binario del supervisor.
func (s *Supervisor) Reload(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	desired := make(map[string]config.ProcessConfig, len(cfg.Processes))
	for _, procCfg := range cfg.Processes {
		desired[procCfg.Name] = procCfg
	}

	s.mu.RLock()
	currentNames := make([]string, 0, len(s.workers))
	for name := range s.workers {
		currentNames = append(currentNames, name)
	}
	s.mu.RUnlock()

	for _, name := range currentNames {
		if _, keep := desired[name]; !keep {
			log.Printf("recarga: deteniendo proceso %q (eliminado de la configuración)", name)
			s.stopWorker(name)
		}
	}

	s.mu.Lock()
	s.gracePeriod = cfg.GracePeriod()
	s.cfg = cfg
	s.mu.Unlock()

	for _, procCfg := range cfg.Processes {
		s.mu.RLock()
		mw, exists := s.workers[procCfg.Name]
		s.mu.RUnlock()

		if !exists {
			log.Printf("recarga: iniciando proceso nuevo %q", procCfg.Name)
			s.startWorker(procCfg)
			continue
		}

		if config.ProcessConfigEqual(mw.cfg, procCfg) {
			continue
		}

		log.Printf("recarga: reconfigurando proceso %q", procCfg.Name)
		s.stopWorker(procCfg.Name)
		s.startWorker(procCfg)
	}

	return nil
}

// Status devuelve el estado de los workers activos.
func (s *Supervisor) Status() map[string]ProcessStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make(map[string]ProcessStatus, len(s.workers))
	for name, mw := range s.workers {
		statuses[name] = mw.worker.Status()
	}
	return statuses
}

// AllStatus devuelve el estado de todos los procesos definidos en la configuración.
func (s *Supervisor) AllStatus() map[string]ProcessStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make(map[string]ProcessStatus, len(s.cfg.Processes))
	for _, procCfg := range s.cfg.Processes {
		statuses[procCfg.Name] = ProcessStatus{
			Name:  procCfg.Name,
			State: StateStopped,
		}
	}
	for name, mw := range s.workers {
		statuses[name] = mw.worker.Status()
	}
	return statuses
}

// StatusOf devuelve el estado de un proceso por nombre.
func (s *Supervisor) StatusOf(name string) (ProcessStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	found := false
	for _, procCfg := range s.cfg.Processes {
		if procCfg.Name == name {
			found = true
			break
		}
	}
	if !found {
		return ProcessStatus{}, false
	}

	if mw, ok := s.workers[name]; ok {
		return mw.worker.Status(), true
	}
	return ProcessStatus{Name: name, State: StateStopped}, true
}

// StartProcess arranca la supervisión de un proceso definido en la configuración.
func (s *Supervisor) StartProcess(name string) error {
	if s.rootCtx != nil && s.rootCtx.Err() != nil {
		return fmt.Errorf("supervisor en apagado")
	}

	procCfg, ok := s.processConfig(name)
	if !ok {
		return fmt.Errorf("proceso %q no está definido en la configuración", name)
	}

	s.mu.RLock()
	_, running := s.workers[name]
	s.mu.RUnlock()
	if running {
		return fmt.Errorf("proceso %q ya está en ejecución", name)
	}

	log.Printf("control: iniciando proceso %q", name)
	s.startWorker(procCfg)
	return nil
}

// StopProcess detiene un proceso supervisado en ejecución.
func (s *Supervisor) StopProcess(name string) error {
	s.mu.RLock()
	_, running := s.workers[name]
	s.mu.RUnlock()
	if !running {
		return fmt.Errorf("proceso %q no está en ejecución", name)
	}

	log.Printf("control: deteniendo proceso %q", name)
	s.stopWorker(name)
	return nil
}

// RestartProcess detiene (si corre) y vuelve a iniciar un proceso.
func (s *Supervisor) RestartProcess(name string) error {
	if s.rootCtx != nil && s.rootCtx.Err() != nil {
		return fmt.Errorf("supervisor en apagado")
	}

	procCfg, ok := s.processConfig(name)
	if !ok {
		return fmt.Errorf("proceso %q no está definido en la configuración", name)
	}

	s.mu.RLock()
	_, running := s.workers[name]
	s.mu.RUnlock()
	if running {
		log.Printf("control: reiniciando proceso %q", name)
		s.stopWorker(name)
	} else {
		log.Printf("control: iniciando proceso %q (restart)", name)
	}

	s.startWorker(procCfg)
	return nil
}

func (s *Supervisor) processConfig(name string) (config.ProcessConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, procCfg := range s.cfg.Processes {
		if procCfg.Name == name {
			return procCfg, true
		}
	}
	return config.ProcessConfig{}, false
}

func (s *Supervisor) startWorker(cfg config.ProcessConfig) {
	if s.rootCtx == nil {
		s.rootCtx = context.Background()
	}

	workerCtx, cancel := context.WithCancel(s.rootCtx)
	worker := NewWorker(cfg, s.gracePeriod)
	mw := &managedWorker{
		worker: worker,
		cancel: cancel,
		cfg:    cfg,
		done:   make(chan struct{}),
	}

	s.mu.Lock()
	s.workers[cfg.Name] = mw
	s.mu.Unlock()

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer close(mw.done)
		defer s.cleanupWorker(cfg.Name, mw)
		worker.Run(workerCtx)
	}()
}

func (s *Supervisor) cleanupWorker(name string, mw *managedWorker) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if current, ok := s.workers[name]; ok && current == mw {
		delete(s.workers, name)
	}
}

func (s *Supervisor) stopWorker(name string) {
	s.mu.Lock()
	mw, ok := s.workers[name]
	if !ok {
		s.mu.Unlock()
		return
	}
	cancel := mw.cancel
	done := mw.done
	s.mu.Unlock()

	cancel()
	<-done

	s.mu.Lock()
	delete(s.workers, name)
	s.mu.Unlock()
}

func (s *Supervisor) stopAllWorkers() {
	s.mu.RLock()
	names := make([]string, 0, len(s.workers))
	for name := range s.workers {
		names = append(names, name)
	}
	s.mu.RUnlock()

	for _, name := range names {
		s.stopWorker(name)
	}
}

// GracePeriod expone el periodo de gracia configurado (útil para tests).
func (s *Supervisor) GracePeriod() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gracePeriod
}
