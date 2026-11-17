//go:build !windows

package sdk

import (
	"os/exec"
	"syscall"
)

// detach отвязывает процесс от терминала hexago: своя сессия, Ctrl+C его не остановит.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
