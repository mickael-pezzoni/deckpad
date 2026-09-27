package system

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

var (
	procLockWorkStation = windows.NewLazySystemDLL("user32.dll").NewProc("LockWorkStation")
	procSetSuspendState = windows.NewLazySystemDLL("powrprof.dll").NewProc("SetSuspendState")
)

func run(a Action) error {
	switch a {
	case Lock:
		// Fonctionne car l'agent tourne dans la session de l'utilisateur.
		if r, _, err := procLockWorkStation.Call(); r == 0 {
			return err
		}
	case Sleep:
		// SetSuspendState(hibernate=false, force=false, disableWakeEvent=false) : mise en veille.
		if r, _, err := procSetSuspendState.Call(0, 0, 0); r == 0 {
			return err
		}
	case Restart:
		return shutdownExe("/r")
	case Shutdown:
		return shutdownExe("/s")
	}
	return nil
}

func shutdownExe(flag string) error {
	cmd := exec.Command("shutdown", flag, "/t", "0")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd.Run()
}
