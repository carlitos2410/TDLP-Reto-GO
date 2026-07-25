//go:build windows

package signal

import (
	"os"
	"os/signal"
	"syscall"
)

func register(notifier *Notifier) {
	signal.Notify(notifier.shutdown, os.Interrupt, syscall.SIGTERM)
}

func reloadSupported() bool {
	return false
}
