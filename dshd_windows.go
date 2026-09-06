//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"syscall"

	"golang.org/x/sys/windows"
)

func applyProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = hiddenProcAttr(windows.CREATE_NEW_PROCESS_GROUP)
}

func hiddenProcAttr(extra uint32) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW | extra,
	}
}

func runHidden(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = hiddenProcAttr(0)
	_ = cmd.Run()
}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	runHidden("taskkill", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
}

func killProcessForce(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	runHidden("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
}
