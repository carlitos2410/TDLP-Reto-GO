//go:build unix

package signal

import (
	"os/signal"
	"syscall"
)

func register(notifier *Notifier) {
	signal.Notify(notifier.shutdown, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(notifier.reload, syscall.SIGHUP)
}

func reloadSupported() bool {
	return true
}
