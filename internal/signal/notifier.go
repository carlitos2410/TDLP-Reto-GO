package signal

import (
	"context"
	"log"
	"os"
)

// Notifier escucha señales del sistema y delega apagado o recarga.
type Notifier struct {
	shutdown chan os.Signal
	reload   chan os.Signal
}

// NewNotifier registra las señales soportadas en la plataforma actual.
func NewNotifier() *Notifier {
	n := &Notifier{
		shutdown: make(chan os.Signal, 1),
		reload:   make(chan os.Signal, 1),
	}
	register(n)
	return n
}

// Run bloquea hasta apagado (SIGINT/SIGTERM) o invoca onReload en cada recarga (SIGHUP en Unix).
func (n *Notifier) Run(ctx context.Context, onShutdown func(), onReload func()) {
	for {
		select {
		case <-ctx.Done():
			return
		case sig := <-n.shutdown:
			log.Printf("señal recibida: %s, iniciando apagado ordenado", sig)
			onShutdown()
			return
		case sig := <-n.reload:
			if !reloadSupported() {
				continue
			}
			log.Printf("señal recibida: %s, recargando configuración", sig)
			onReload()
		}
	}
}
