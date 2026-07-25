//go:build windows

package process

import "os"

// sendTerminateSignal en Windows no tiene SIGTERM portable; se confía en el periodo de gracia y SIGKILL.
func sendTerminateSignal(proc *os.Process) error {
	return nil
}
