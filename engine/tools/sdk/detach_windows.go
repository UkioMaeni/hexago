package sdk

import (
	"os/exec"
	"syscall"
)

// detach отвязывает процесс от консоли hexago: Ctrl+C в терминале его не остановит,
// и лишнего окна консоли не будет.
func detach(cmd *exec.Cmd) {
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow,
	}
}
