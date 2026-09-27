package stats

import (
	"os/exec"
	"syscall"
)

// hideWindow évite qu'une console s'ouvre à chaque appel de nvidia-smi.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
