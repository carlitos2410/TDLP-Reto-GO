//go:build unix

package process

import (
	"os"
	"syscall"
)

// sendTerminateSignal pide terminación ordenada al proceso hijo (SIGTERM).
func sendTerminateSignal(proc *os.Process) error {
	return proc.Signal(syscall.SIGTERM)
}
