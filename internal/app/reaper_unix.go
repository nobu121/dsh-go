//go:build unix

package app

import (
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
)

// startDeathReaper watches this process via a pipe. If the shell is force-killed,
// the kernel closes the write end; a tiny helper then SIGKILLs the dsh process group.
func startDeathReaper(pgid int) func() {
	if pgid <= 1 {
		return func() {}
	}
	r, w, err := os.Pipe()
	if err != nil {
		log.Printf("dsh death reaper: pipe: %v", err)
		return func() {}
	}
	cmd := exec.Command("/bin/sh", "-c",
		"IFS= read -r _ || true; kill -9 -"+strconv.Itoa(pgid)+" >/dev/null 2>&1 || true")
	cmd.Stdin = r
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	applyReaperProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		_ = r.Close()
		_ = w.Close()
		log.Printf("dsh death reaper: %v", err)
		return func() {}
	}
	_ = r.Close()
	return func() {
		_ = w.Close()
		go func() { _ = cmd.Wait() }()
	}
}
