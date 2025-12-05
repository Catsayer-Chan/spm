package cmd

import (
	"os"
	"os/exec"
	"syscall"

	"spm/pkg/utils"
	"spm/pkg/utils/constants"
)

func isDaemonRunning() bool {
	daemonPid, err := utils.ReadPid(constants.DaemonPidFilePath)
	if err != nil {
		return false
	}

	if daemonPid < 0 {
		return false
	}

	return isPidActive(daemonPid)
}

func isPidActive(p int) bool {
	_, err := syscall.Getpgid(p)

	return err == nil
}

func tryRunDaemon() error {
	var cmd *exec.Cmd
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := make([]string, 0)
	args = append(args, "daemon")
	args = append(args, os.Args[2:]...)

	cmd = exec.Command(exe, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err = cmd.Start()

	return err
}
