// Package utils
package utils

import (
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"

	"spm/pkg/config"
)

const GlobalConfigFile = "/etc/spm.yml"

var RuntimeInfo, _ = debug.ReadBuildInfo()
var RuntimeModuleInfo = strings.Split(RuntimeInfo.Main.Path, "/")
var RuntimeModuleName = RuntimeModuleInfo[len(RuntimeModuleInfo)-1]

var SupervisorPid = os.Getpid()

var FinishChan = make(chan struct{}, 1)
var StopChan = make(chan os.Signal, 1)

func WriteDaemonPid(pid int) error {
	if err := os.WriteFile(
		config.GetConfig().PidFile,
		[]byte(strconv.Itoa(pid)),
		0644,
	); err != nil {
		return fmt.Errorf("error writing PID to file: %v", err)
	}

	return nil
}

func CheckPerm(tmpDir string) error {
	tmpFile, err := os.CreateTemp(tmpDir, "*")
	if err != nil {
		return err
	}

	return os.Remove(tmpFile.Name())
}

func ReadPid(pidFile string) (int, error) {
	var pidNum int
	if _, err := os.Stat(pidFile); err != nil {
		return -1, err
	}

	isNum := regexp.MustCompile("[0-9]+$")

	pid, err := os.ReadFile(pidFile)
	if err != nil {
		return -1, err
	}

	pidStr := strings.TrimSpace(string(pid))
	if isNum.MatchString(pidStr) {
		pidNum, err = strconv.Atoi(pidStr)
		if err != nil {
			return -1, err
		}
	} else {
		return -1, err
	}

	return pidNum, nil
}
