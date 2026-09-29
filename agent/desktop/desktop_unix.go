//go:build !windows

package desktop

import (
	"os/exec"
	"syscall"
)

// startDetached lance le programme dans sa propre session : il continue de
// tourner même si l'agent s'arrête.
func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}
