package main

// DSH supervises one DeepSeek Harness process tree.

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"log"
	"net/url"
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
)

var (
	dshURLLine = regexp.MustCompile(`dsh web: (https?://\S+)`)
	lookPath   = exec.LookPath
	errNoDSH   = errors.New("no dsh available")
)

type dshSourceKind string

const (
	sourceManual  dshSourceKind = "manual"
	sourcePath    dshSourceKind = "path"
	sourceCache   dshSourceKind = "cache"
	sourceBundled dshSourceKind = "bundled"
	sourceRepo    dshSourceKind = "repo"
	sourceNpx     dshSourceKind = "npx"
)

type resolvedDSH struct {
	Argv []string
	Kind dshSourceKind
	Path string
}

func parseDSHWebURL(line string) (string, bool) {
	m := dshURLLine.FindStringSubmatch(line)
	if m == nil {
		return "", false
	}
	raw := strings.TrimRight(m[1], ".,;)")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	if u.Query().Get("token") == "" {
		return "", false
	}
	return u.String(), true
}

// DSH supervises one dsh process tree.
type DSH struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	closed  bool
	onReady func(url string)
	config  DSHConfig
	source  resolvedDSH
	lastURL string
}

// DSHConfig holds the resolved launch configuration.
type DSHConfig struct {
	Command []string
	Home    string
	OnPrep  PrepReporter
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
	for _, root := range runtimeRoots() {
		if runtimeLooksValid(root) {
			return runtimeNodeCommand(root), nil
		}
	}
	return nil, errors.New("no bundled node + @deepseek-ai/dsh runtime")
}

func resolveLaunch(c DSHConfig) (resolvedDSH, error) {
	if len(c.Command) > 0 {
		return resolvedDSH{Argv: c.Command, Kind: sourceManual}, nil
	}
	if exe := os.Getenv("DSH_EXE"); exe != "" {
		if _, err := os.Stat(exe); err != nil {
			return resolvedDSH{}, err
		}
		return resolvedDSH{
			Argv: append([]string{exe}, webFlags()...),
			Kind: sourceManual,
			Path: exe,
		}, nil
	}
	if exe, err := lookNamed("dsh"); err == nil && exe != "" {
		if _, err := os.Stat(exe); err == nil {
			return resolvedDSH{
				Argv: append([]string{exe}, webFlags()...),
				Kind: sourcePath,
				Path: exe,
			}, nil
		}
	}
	if argv, ok := cacheRuntimeCommand(); ok {
		return resolvedDSH{Argv: argv, Kind: sourceCache, Path: runtimeCacheDir()}, nil
	}
	if bundled, err := bundledNodeCommand(); err == nil {
		return resolvedDSH{Argv: bundled, Kind: sourceBundled}, nil
	}
	return resolvedDSH{}, errNoDSH
}

func resolveDevFallback() (resolvedDSH, bool) {
	repo := os.Getenv("DSH_REPO")
	if repo == "" {
		home, _ := os.UserHomeDir()
		repo = strings.Replace(defaultRepo, "~/", home+"/", 1)
	}
	bin := filepath.Join(repo, "apps", "cli", "lib", "bin.js")
	if _, err := os.Stat(bin); err == nil {
		node := "node"
		if p, err := lookNamed("node"); err == nil {
			node = p
		}
		return resolvedDSH{
			Argv: append([]string{node, bin}, webFlags()...),
			Kind: sourceRepo,
			Path: repo,
		}, true
	}
	if argv, ok := hotNpxArgv(currentVersion()); ok {
		return resolvedDSH{Argv: argv, Kind: sourceNpx}, true
	}
	return resolvedDSH{}, false
}

func hotNpxArgv(pin string) ([]string, bool) {
	npx, err := lookNamed("npx")
	if err != nil || npx == "" || pin == "" {
		return nil, false
	}
	if !npxHasCachedPin(pin) {
		return nil, false
	}
	return append([]string{npx, "-y", "@deepseek-ai/dsh@" + pin}, webFlags()...), true
}

func npxHasCachedPin(pin string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	root := filepath.Join(home, ".npm", "_npx")
	if _, err := os.Stat(root); err != nil {
		return false
	}
	found := false
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found {
			return err
		}
		if d.Name() != "package.json" {
			return nil
		}
		if filepath.Base(filepath.Dir(path)) != "dsh" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(string(b), `"version": "`+pin+`"`) {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}

func (c DSHConfig) resolveCommand() ([]string, error) {
	r, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return r.Argv, nil
}

func (c DSHConfig) resolve() (resolvedDSH, error) {
	r, err := resolveLaunch(c)
	if err == nil {
		return r, nil
	}
	if !errors.Is(err, errNoDSH) {
		return resolvedDSH{}, err
	}
	if r, ok := resolveDevFallback(); ok {
		return r, nil
	}
	return resolvedDSH{}, errors.New("no dsh available: install dsh, set DSH_EXE / DSH_REPO, or set DSH_RUNTIME_BASE_URL")
}

func (d *DSH) ensureCommand(ctx context.Context) ([]string, error) {
	d.report(PrepProgress{Stage: "detect", Message: "正在检测本机 dsh…"})
	r, err := resolveLaunch(d.config)
	if err == nil {
		d.setSource(r)
		return r.Argv, nil
	}
	if !errors.Is(err, errNoDSH) {
		return nil, err
	}
	if base := runtimeBaseURL(); base != "" {
		if ferr := fetchCachedRuntime(ctx, d.config.OnPrep); ferr != nil {
			d.report(PrepProgress{Stage: "error", Message: ferr.Error()})
		} else if r, err := resolveLaunch(d.config); err == nil {
			d.setSource(r)
			return r.Argv, nil
		}
	}
	if r, ok := resolveDevFallback(); ok {
		d.setSource(r)
		return r.Argv, nil
	}
	return nil, errors.New("no dsh available: install dsh, set DSH_EXE / DSH_REPO, or set DSH_RUNTIME_BASE_URL")
}

func (d *DSH) setSource(r resolvedDSH) {
	d.mu.Lock()
	d.source = r
	d.mu.Unlock()
}

func (d *DSH) Source() resolvedDSH {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.source
}

func (d *DSH) setLastURL(url string) {
	d.mu.Lock()
	d.lastURL = url
	d.mu.Unlock()
}

func (d *DSH) LastURL() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastURL
}

func (d *DSH) report(p PrepProgress) {
	reportPrep(d.config.OnPrep, p)
}

// killCurrent stops the running dsh process without marking the supervisor closed,
// so the Start loop relaunches.
func (d *DSH) killCurrent() {
	d.mu.Lock()
	cmd := d.cmd
	d.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	killProcess(cmd)
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
	argv, err := d.ensureCommand(ctx)
	if err != nil {
		d.report(PrepProgress{Stage: "error", Message: err.Error()})
		return false, err
	}
	d.report(PrepProgress{Stage: "start", Message: "正在启动 DeepSeek Harness…"})
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), "DSH_HOME="+d.config.Home)
	applyProcAttr(cmd)

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	d.mu.Lock()
	d.cmd = cmd
	d.mu.Unlock()

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return false, err
	}
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
		_ = pw.Close()
	}()
	stopReap := startDeathReaper(cmd.Process.Pid)
	defer stopReap()
	log.Printf("dsh started: %s", strings.Join(argv, " "))

	ready := false
	sc := bufio.NewScanner(pr)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		line := sc.Text()
		if ready {
			continue
		}
		if url, ok := parseDSHWebURL(line); ok {
			ready = true
			d.setLastURL(url)
			log.Printf("dsh ready: %s", url)
			if d.onReady != nil {
				d.onReady(url)
			}
		}
	}
	err = <-waitErr
	if err != nil && ctx.Err() == nil {
		log.Printf("dsh exited with error: %v", err)
	}
	return ready, nil
}

// Close marks the supervisor stopped and kills the dsh process group.
// launchOnce already Wait()s; a second Wait here raced and blocked Quit
// for up to several seconds after the window closed.
func (d *DSH) Close() {
	d.mu.Lock()
	d.closed = true
	cmd := d.cmd
	d.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return
	}
	killProcessForce(cmd)
	log.Printf("dsh stopped")
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
