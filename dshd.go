package main

// DSH supervises one DeepSeek Harness process tree.

import (
	"bufio"
	"context"
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	defaultRepo = "~/Desktop/deepseek-harness"
	maxRestarts = 5
	shutdownAck = 5 * time.Second
)

var dshURLLine = regexp.MustCompile(`^dsh web: (https?://\S+)$`)

func parseDSHWebURL(line string) (string, bool) {
	m := dshURLLine.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// DSH supervises one dsh process tree.
type DSH struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	closed  bool
	onReady func(url string)
	config  DSHConfig
}

// DSHConfig holds the resolved launch configuration.
type DSHConfig struct {
	Command []string
	Home    string
}

func defaultHomeDir() string {
	base := os.Getenv("DSH_HOME")
	if base != "" {
		return base
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "dsh-go", "dsh-home")
	}
	return filepath.Join(cfg, "dsh-go", "dsh-home")
}

func webFlags() []string {
	return []string{"--profile", "web", "--no-open", "--port", "0"}
}

func dshBinJS(root string) string {
	return filepath.Join(root, "node_modules", "@deepseek-ai", "dsh", "lib", "bin.js")
}

func bundledNodeName() string {
	if runtime.GOOS == "windows" {
		return "node.exe"
	}
	return "node"
}

func runtimeRoots() []string {
	var roots []string
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		roots = append(roots,
			filepath.Join(exeDir, "..", "Resources", "dsh-runtime"),
			filepath.Join(exeDir, "dsh-runtime"),
		)
	}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, filepath.Join(wd, "vendor", "dsh"))
	}
	return roots
}

func bundledNodeCommand() ([]string, error) {
	nodeName := bundledNodeName()
	for _, root := range runtimeRoots() {
		node := filepath.Join(root, nodeName)
		bin := dshBinJS(root)
		if fi, err := os.Stat(node); err == nil && !fi.IsDir() {
			if _, err := os.Stat(bin); err == nil {
				return append([]string{node, bin}, webFlags()...), nil
			}
		}
	}
	return nil, errors.New("no bundled node + @deepseek-ai/dsh runtime")
}

func (c DSHConfig) resolveCommand() ([]string, error) {
	if len(c.Command) > 0 {
		return c.Command, nil
	}
	if exe := os.Getenv("DSH_EXE"); exe != "" {
		if _, err := os.Stat(exe); err != nil {
			return nil, err
		}
		return append([]string{exe}, webFlags()...), nil
	}
	if bundled, err := bundledNodeCommand(); err == nil {
		return bundled, nil
	}
	repo := os.Getenv("DSH_REPO")
	if repo == "" {
		home, _ := os.UserHomeDir()
		repo = strings.Replace(defaultRepo, "~/", home+"/", 1)
	}
	bin := filepath.Join(repo, "apps", "cli", "lib", "bin.js")
	if _, err := os.Stat(bin); err == nil {
		return append([]string{"node", bin}, webFlags()...), nil
	}
	wrapper := filepath.Join(".", "scripts", "run-dsh.sh")
	if wd, err := os.Getwd(); err == nil {
		wrapper = filepath.Join(wd, "scripts", "run-dsh.sh")
	}
	if _, err := os.Stat(wrapper); err != nil {
		return nil, errors.New("no dsh available: run scripts/sync-dsh.sh or set DSH_EXE / DSH_REPO")
	}
	return append([]string{wrapper}, webFlags()...), nil
}

// NewDSH returns a supervisor. onReady is invoked with the authenticated
// startup URL as soon as dsh announces it.
func NewDSH(cfg DSHConfig, onReady func(url string)) *DSH {
	return &DSH{config: cfg, onReady: onReady}
}

// Start launches dsh and blocks supervising it until Close is called or the
// retry budget is exhausted.
func (d *DSH) Start(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		ready, err := d.launchOnce(ctx)
		if err != nil {
			return err
		}
		if d.isClosed() {
			return nil
		}
		if !ready && attempt >= maxRestarts {
			return errors.New("dsh exited repeatedly; giving up after " +
				itoa(attempt) + " restarts")
		}
		log.Printf("dsh exited (ready=%v, attempt=%d); restarting", ready, attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (d *DSH) launchOnce(ctx context.Context) (bool, error) {
	argv, err := d.config.resolveCommand()
	if err != nil {
		return false, err
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "DSH_HOME="+d.config.Home)
	applyProcAttr(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false, err
	}
	cmd.Stderr = os.Stderr

	d.mu.Lock()
	d.cmd = cmd
	d.mu.Unlock()

	if err := cmd.Start(); err != nil {
		return false, err
	}
	log.Printf("dsh started: %s", strings.Join(argv, " "))

	ready := false
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		line := sc.Text()
		if ready {
			continue
		}
		if url, ok := parseDSHWebURL(line); ok {
			ready = true
			log.Printf("dsh ready: %s", url)
			if d.onReady != nil {
				d.onReady(url)
			}
		}
	}
	err = cmd.Wait()
	if err != nil && ctx.Err() == nil {
		log.Printf("dsh exited with error: %v", err)
	}
	return ready, nil
}

// Close terminates the dsh process tree and waits for it to exit.
func (d *DSH) Close() {
	d.mu.Lock()
	d.closed = true
	cmd := d.cmd
	d.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	killProcess(cmd)
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	select {
	case <-done:
		log.Printf("dsh stopped")
	case <-time.After(shutdownAck):
		killProcessForce(cmd)
		log.Printf("dsh killed after grace period")
	}
}

func (d *DSH) isClosed() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.closed
}

func itoa(n int) string {
	var b [20]byte
	i := len(b)
	for {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
		if n == 0 {
			break
		}
	}
	return string(b[i:])
}
