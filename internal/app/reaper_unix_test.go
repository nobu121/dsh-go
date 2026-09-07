//go:build unix

package app

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestDeathReaperKillsProcessGroup(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = cmd.Wait()
	})

	stop := startDeathReaper(pid)
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("sleep died before reaper fired: %v", err)
	}

	stop()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("sleep exited 0; expected signal")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("death reaper did not kill process group")
	}
}

func TestDeathReaperIgnoresInvalidPgid(t *testing.T) {
	stop := startDeathReaper(1)
	stop()
	stop = startDeathReaper(0)
	stop()
}
