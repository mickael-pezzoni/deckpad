//go:build !windows

package stats

import "os/exec"

func hideWindow(*exec.Cmd) {}
