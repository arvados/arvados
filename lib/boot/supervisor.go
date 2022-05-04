// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

type Supervisor struct {
	// Config file location like "/etc/arvados/config.yml", or "-"
	// to read from Stdin (see below).
	ConfigPath string
	// Literal config file (useful for test suites). If non-empty,
	// this is used instead of ConfigPath.
	ConfigYAML string
	// Path to arvados source tree. Only used for dev/test
	// clusters.
	SourcePath string
	// Version number to build into binaries. Only used for
	// dev/test clusters.
	SourceVersion string
	// "production", "development", or "test".
	ClusterType string
	// Listening address for external services, and internal
	// services whose InternalURLs are not explicitly configured.
	// If blank, listen on the configured controller ExternalURL
	// host; if that is also blank, listen on all addresses
	// (0.0.0.0).
	ListenHost string
	// Default host:port for controller ExternalURL if not
	// explicitly configured in config file. If blank, use a
	// random port on ListenHost.
	ControllerAddr string
	// Path to arvados-workbench2 source tree checkout.
	Workbench2Source     string
	NoWorkbench1         bool
	NoWorkbench2         bool
	OwnTemporaryDatabase bool
	Stdin                io.Reader
	Stderr               io.Writer

	logger   logrus.FieldLogger
	cluster  *arvados.Cluster       // nil if this is a multi-cluster supervisor
	children map[string]*Supervisor // nil if this is a single-cluster supervisor

	ctx           context.Context
	cancel        context.CancelFunc
	done          chan struct{}      // closed when child procs/services have shut down
	err           error              // error that caused shutdown (valid when done is closed)
	healthChecker *health.Aggregator // nil if this is a multi-cluster supervisor, or still booting
	tasksReady    map[string]chan bool
	waitShutdown  sync.WaitGroup

	bindir     string
	tempdir    string // in production mode, this is accessible only to root
	wwwtempdir string // in production mode, this is accessible only to www-data
	configfile string
	environ    []string // for child processes
}

func (super *Supervisor) Clusters() map[string]*arvados.Cluster {
	m := map[string]*arvados.Cluster{}
	if super.cluster != nil {
		m[super.cluster.ClusterID] = super.cluster
	}
	for id, super2 := range super.children {
		m[id] = super2.Cluster("")
	}
	return m
}

func (super *Supervisor) Cluster(id string) *arvados.Cluster {
	if super.children != nil {
		return super.children[id].Cluster(id)
	} else {
		return super.cluster
	}
}

func (super *Supervisor) Start(ctx context.Context) {
	super.logger = ctxlog.FromContext(ctx)
	super.ctx, super.cancel = context.WithCancel(ctx)
	super.done = make(chan struct{})

	sigch := make(chan os.Signal)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigch)
	go func() {
		for sig := range sigch {
			super.logger.WithField("signal", sig).Info("caught signal")
			if super.err == nil {
				super.err = fmt.Errorf("caught signal %s", sig)
			}
			super.cancel()
		}
	}()

	hupch := make(chan os.Signal)
	signal.Notify(hupch, syscall.SIGHUP)
	defer signal.Stop(hupch)
	go func() {
		for sig := range hupch {
			super.logger.WithField("signal", sig).Info("caught signal")
			if super.err == nil {
				super.err = errNeedConfigReload
			}
			super.cancel()
		}
	}()

	loaderStdin := super.Stdin
	if super.ConfigYAML != "" {
		loaderStdin = bytes.NewBufferString(super.ConfigYAML)
	}
	loader := config.NewLoader(loaderStdin, super.logger)
	loader.SkipLegacy = true
	loader.SkipAPICalls = true
	loader.Path = super.ConfigPath
	if super.ConfigYAML != "" {
		loader.Path = "-"
	}
	cfg, err := loader.Load()
	if err != nil {
		super.err = err
		close(super.done)
		super.cancel()
		return
	}

	if super.ConfigPath != "" && super.ConfigPath != "-" && cfg.AutoReloadConfig {
		go watchConfig(super.ctx, super.logger, super.ConfigPath, copyConfig(cfg), func() {
			if super.err == nil {
				super.err = errNeedConfigReload
			}
			super.cancel()
		})
	}

	if len(cfg.Clusters) > 1 {
		super.startFederation(cfg)
		go func() {
			defer super.cancel()
			defer close(super.done)
			for _, super2 := range super.children {
				err := super2.Wait()
				if super.err == nil {
					super.err = err
				}
			}
		}()
	} else {
		go func() {
			defer super.cancel()
			defer close(super.done)
			super.cluster, super.err = cfg.GetCluster("")
			if super.err != nil {
				return
			}
			err := super.runCluster()
			if err != nil {
				super.logger.WithError(err).Info("supervisor shut down")
				if super.err == nil {
					super.err = err
				}
			}
		}()
	}
}

// Wait returns when all child processes and goroutines have exited.
func (super *Supervisor) Wait() error {
	<-super.done
	return super.err
}

// startFederation starts a child Supervisor for each cluster in the
// given config. Each is a copy of the original/parent with the
// original config reduced to a single cluster.
func (super *Supervisor) startFederation(cfg *arvados.Config) {
	super.children = map[string]*Supervisor{}
	for id, cc := range cfg.Clusters {
		super2 := *super
		yaml, err := json.Marshal(arvados.Config{Clusters: map[string]arvados.Cluster{id: cc}})
		if err != nil {
			panic(fmt.Sprintf("json.Marshal partial config: %s", err))
		}
		super2.ConfigYAML = string(yaml)
		super2.ConfigPath = "-"
		super2.children = nil

		if super2.ClusterType == "test" {
			super2.Stderr = &service.LogPrefixer{
				Writer: super.Stderr,
				Prefix: []byte("[" + id + "] "),
			}
		}
		super2.Start(super.ctx)
		super.children[id] = &super2
	}
}

func (super *Supervisor) runCluster() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if super.ClusterType == "test" && super.SourcePath == "" {
		// When invoked by test suite, default to current
		// source tree
		buf, err := exec.Command("git", "rev-parse", "--show-toplevel").CombinedOutput()
		if err != nil {
			return fmt.Errorf("git rev-parse: %w", err)
		}
		super.SourcePath = strings.TrimSuffix(string(buf), "\n")
	} else if !strings.HasPrefix(super.SourcePath, "/") {
		super.SourcePath = filepath.Join(cwd, super.SourcePath)
	}
	super.SourcePath, err = filepath.EvalSymlinks(super.SourcePath)
	if err != nil {
		return err
	}

	if super.ListenHost == "" {
		if urlhost := super.cluster.Services.Controller.ExternalURL.Host; urlhost != "" {
			if h, _, _ := net.SplitHostPort(urlhost); h != "" {
				super.ListenHost = h
			} else {
				super.ListenHost = urlhost
			}
		} else {
			super.ListenHost = "0.0.0.0"
		}
	}

	// Choose bin and temp dirs: /var/lib/arvados/... in
	// production, transient tempdir otherwise.
	if super.ClusterType == "production" {
		// These dirs have already been created by
		// "arvados-server install" (or by extracting a
		// package).
		super.tempdir = "/var/lib/arvados/tmp"
		super.wwwtempdir = "/var/lib/arvados/wwwtmp"
		super.bindir = "/var/lib/arvados/bin"
	} else {
		super.tempdir, err = ioutil.TempDir("", "arvados-server-boot-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(super.tempdir)
		super.wwwtempdir = super.tempdir
		super.bindir = filepath.Join(super.tempdir, "bin")
		if err := os.Mkdir(super.bindir, 0755); err != nil {
			return err
		}
	}

	// Fill in any missing config keys, and write the resulting
	// config in the temp dir for child services to use.
	err = super.autofillConfig()
	if err != nil {
		return err
	}
	conffile, err := os.OpenFile(filepath.Join(super.wwwtempdir, "config.yml"), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer conffile.Close()
	err = json.NewEncoder(conffile).Encode(arvados.Config{
		Clusters: map[string]arvados.Cluster{
			super.cluster.ClusterID: *super.cluster}})
	if err != nil {
		return err
	}
	err = conffile.Close()
	if err != nil {
		return err
	}
	super.configfile = conffile.Name()

	super.environ = os.Environ()
	super.cleanEnv([]string{"ARVADOS_"})
	super.setEnv("ARVADOS_CONFIG", super.configfile)
	super.setEnv("RAILS_ENV", super.ClusterType)
	super.setEnv("TMPDIR", super.tempdir)
	super.prependEnv("PATH", "/var/lib/arvados/bin:")
	if super.ClusterType != "production" {
		super.prependEnv("PATH", super.tempdir+"/bin:")
	}

	// Now that we have the config, replace the bootstrap logger
	// with a new one according to the logging config.
	loglevel := super.cluster.SystemLogs.LogLevel
	if s := os.Getenv("ARVADOS_DEBUG"); s != "" && s != "0" {
		loglevel = "debug"
	}
	super.logger = ctxlog.New(super.Stderr, super.cluster.SystemLogs.Format, loglevel).WithFields(logrus.Fields{
		"PID": os.Getpid(),
	})

	if super.SourceVersion == "" && super.ClusterType == "production" {
		// don't need SourceVersion
	} else if super.SourceVersion == "" {
		// Find current source tree version.
		var buf bytes.Buffer
		err = super.RunProgram(super.ctx, ".", runOptions{output: &buf}, "git", "diff", "--shortstat")
		if err != nil {
			return err
		}
		dirty := buf.Len() > 0
		buf.Reset()
		err = super.RunProgram(super.ctx, ".", runOptions{output: &buf}, "git", "log", "-n1", "--format=%H")
		if err != nil {
			return err
		}
		super.SourceVersion = strings.TrimSpace(buf.String())
		if dirty {
			super.SourceVersion += "+uncommitted"
		}
	} else {
		return errors.New("specifying a version to run is not yet supported")
	}

	_, err = super.installGoProgram(super.ctx, "cmd/arvados-server")
	if err != nil {
		return err
	}
	err = super.setupRubyEnv()
	if err != nil {
		return err
	}

	tasks := []supervisedTask{
		createCertificates{},
		runPostgreSQL{},
		runNginx{},
		runServiceCommand{name: "controller", svc: super.cluster.Services.Controller, depends: []supervisedTask{seedDatabase{}}},
		runServiceCommand{name: "git-httpd", svc: super.cluster.Services.GitHTTP},
		runGoProgram{src: "services/health", svc: super.cluster.Services.Health},
		runServiceCommand{name: "keepproxy", svc: super.cluster.Services.Keepproxy, depends: []supervisedTask{runPassenger{src: "services/api"}}},
		runServiceCommand{name: "keepstore", svc: super.cluster.Services.Keepstore},
		runServiceCommand{name: "keep-web", svc: super.cluster.Services.WebDAV},
		runServiceCommand{name: "ws", svc: super.cluster.Services.Websocket, depends: []supervisedTask{seedDatabase{}}},
		installPassenger{src: "services/api", varlibdir: "railsapi"},
		runPassenger{src: "services/api", varlibdir: "railsapi", svc: super.cluster.Services.RailsAPI, depends: []supervisedTask{createCertificates{}, seedDatabase{}, installPassenger{src: "services/api", varlibdir: "railsapi"}}},
		seedDatabase{},
	}
	if !super.NoWorkbench1 {
		tasks = append(tasks,
			installPassenger{src: "apps/workbench", varlibdir: "workbench1", depends: []supervisedTask{seedDatabase{}}}, // dependency ensures workbench doesn't delay api install/startup
			runPassenger{src: "apps/workbench", varlibdir: "workbench1", svc: super.cluster.Services.Workbench1, depends: []supervisedTask{installPassenger{src: "apps/workbench", varlibdir: "workbench1"}}},
		)
	}
	if !super.NoWorkbench2 {
		tasks = append(tasks,
			runWorkbench2{svc: super.cluster.Services.Workbench2},
		)
	}
	if super.ClusterType != "test" {
		tasks = append(tasks,
			runServiceCommand{name: "keep-balance", svc: super.cluster.Services.Keepbalance},
		)
	}
	if super.cluster.Containers.CloudVMs.Enable {
		tasks = append(tasks,
			runServiceCommand{name: "dispatch-cloud", svc: super.cluster.Services.DispatchCloud},
		)
	}
	super.tasksReady = map[string]chan bool{}
	for _, task := range tasks {
		super.tasksReady[task.String()] = make(chan bool)
	}
	for _, task := range tasks {
		task := task
		fail := func(err error) {
			if super.ctx.Err() != nil {
				return
			}
			super.cancel()
			super.logger.WithField("task", task.String()).WithError(err).Error("task failed")
		}
		go func() {
			super.logger.WithField("task", task.String()).Info("starting")
			err := task.Run(super.ctx, fail, super)
			if err != nil {
				fail(err)
				return
			}
			close(super.tasksReady[task.String()])
		}()
	}
	err = super.wait(super.ctx, tasks...)
	if err != nil {
		return err
	}
	super.logger.Info("all startup tasks are complete; starting health checks")
	super.healthChecker = &health.Aggregator{Cluster: super.cluster}
	<-super.ctx.Done()
	super.logger.Info("shutting down")
	super.waitShutdown.Wait()
	return super.ctx.Err()
}

func (super *Supervisor) wait(ctx context.Context, tasks ...supervisedTask) error {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for _, task := range tasks {
		ch, ok := super.tasksReady[task.String()]
		if !ok {
			return fmt.Errorf("no such task: %s", task)
		}
		super.logger.WithField("task", task.String()).Info("waiting")
		for {
			select {
			case <-ch:
				super.logger.WithField("task", task.String()).Info("ready")
			case <-ctx.Done():
				super.logger.WithField("task", task.String()).Info("task was never ready")
				return ctx.Err()
			case <-ticker.C:
				super.logger.WithField("task", task.String()).Info("still waiting...")
				continue
			}
			break
		}
	}
	return nil
}

// Stop shuts down all child processes and goroutines, and returns
// when all of them have exited.
func (super *Supervisor) Stop() {
	super.cancel()
	<-super.done
}

// WaitReady waits for the cluster(s) to be ready to handle requests,
// then returns true. If startup fails, it returns false.
func (super *Supervisor) WaitReady() bool {
	if super.children != nil {
		for id, super2 := range super.children {
			super.logger.Infof("waiting for %s to be ready", id)
			if !super2.WaitReady() {
				super.logger.Infof("%s startup failed", id)
				return false
			}
			super.logger.Infof("%s is ready", id)
		}
		super.logger.Info("all clusters are ready")
		return true
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for waiting := "all"; waiting != ""; {
		select {
		case <-ticker.C:
		case <-super.ctx.Done():
			return false
		}
		if super.healthChecker == nil {
			// not set up yet
			continue
		}
		resp := super.healthChecker.ClusterHealth()
		// The overall health check (resp.Health=="OK") might
		// never pass due to missing components (like
		// arvados-dispatch-cloud in a test cluster), so
		// instead we wait for all configured components to
		// pass.
		waiting = ""
		for target, check := range resp.Checks {
			if check.Health != "OK" {
				waiting += " " + target
			}
		}
		if waiting != "" {
			super.logger.WithField("targets", waiting[1:]).Info("waiting")
		}
	}
	return true
}

func (super *Supervisor) prependEnv(key, prepend string) {
	for i, s := range super.environ {
		if strings.HasPrefix(s, key+"=") {
			super.environ[i] = key + "=" + prepend + s[len(key)+1:]
			return
		}
	}
	super.environ = append(super.environ, key+"="+prepend)
}

func (super *Supervisor) cleanEnv(prefixes []string) {
	var cleaned []string
	for _, s := range super.environ {
		drop := false
		for _, p := range prefixes {
			if strings.HasPrefix(s, p) {
				drop = true
				break
			}
		}
		if !drop {
			cleaned = append(cleaned, s)
		}
	}
	super.environ = cleaned
}

func (super *Supervisor) setEnv(key, val string) {
	for i, s := range super.environ {
		if strings.HasPrefix(s, key+"=") {
			super.environ[i] = key + "=" + val
			return
		}
	}
	super.environ = append(super.environ, key+"="+val)
}

// Remove all but the first occurrence of each env var.
func dedupEnv(in []string) []string {
	saw := map[string]bool{}
	var out []string
	for _, kv := range in {
		if split := strings.Index(kv, "="); split < 1 {
			panic("invalid environment var: " + kv)
		} else if saw[kv[:split]] {
			continue
		} else {
			saw[kv[:split]] = true
			out = append(out, kv)
		}
	}
	return out
}

func (super *Supervisor) installGoProgram(ctx context.Context, srcpath string) (string, error) {
	_, basename := filepath.Split(srcpath)
	binfile := filepath.Join(super.bindir, basename)
	if super.ClusterType == "production" {
		return binfile, nil
	}
	err := super.RunProgram(ctx, filepath.Join(super.SourcePath, srcpath), runOptions{env: []string{"GOBIN=" + super.bindir}}, "go", "install", "-ldflags", "-X git.arvados.org/arvados.git/lib/cmd.version="+super.SourceVersion+" -X main.version="+super.SourceVersion)
	return binfile, err
}

func (super *Supervisor) usingRVM() bool {
	return os.Getenv("rvm_path") != ""
}

func (super *Supervisor) setupRubyEnv() error {
	if !super.usingRVM() {
		// (If rvm is in use, assume the caller has everything
		// set up as desired)
		super.cleanEnv([]string{
			"GEM_HOME=",
			"GEM_PATH=",
		})
		gem := "gem"
		if _, err := os.Stat("/var/lib/arvados/bin/gem"); err == nil || super.ClusterType == "production" {
			gem = "/var/lib/arvados/bin/gem"
		}
		cmd := exec.Command(gem, "env", "gempath")
		if super.ClusterType == "production" {
			cmd.Args = append([]string{"sudo", "-u", "www-data", "-E", "HOME=/var/www"}, cmd.Args...)
			path, err := exec.LookPath("sudo")
			if err != nil {
				return fmt.Errorf("LookPath(\"sudo\"): %w", err)
			}
			cmd.Path = path
		}
		cmd.Stderr = super.Stderr
		cmd.Env = super.environ
		buf, err := cmd.Output() // /var/lib/arvados/.gem/ruby/2.5.0/bin:...
		if err != nil || len(buf) == 0 {
			return fmt.Errorf("gem env gempath: %w", err)
		}
		gempath := string(bytes.Split(buf, []byte{':'})[0])
		super.prependEnv("PATH", gempath+"/bin:")
		super.setEnv("GEM_HOME", gempath)
		super.setEnv("GEM_PATH", gempath)
	}
	// Passenger install doesn't work unless $HOME is ~user
	u, err := user.Current()
	if err != nil {
		return err
	}
	super.setEnv("HOME", u.HomeDir)
	return nil
}

func (super *Supervisor) lookPath(prog string) string {
	for _, val := range super.environ {
		if strings.HasPrefix(val, "PATH=") {
			for _, dir := range filepath.SplitList(val[5:]) {
				path := filepath.Join(dir, prog)
				if fi, err := os.Stat(path); err == nil && fi.Mode()&0111 != 0 {
					return path
				}
			}
		}
	}
	return prog
}

type runOptions struct {
	output io.Writer // attach stdout
	env    []string  // add/replace environment variables
	user   string    // run as specified user
	stdin  io.Reader
}

// RunProgram runs prog with args, using dir as working directory. If ctx is
// cancelled while the child is running, RunProgram terminates the child, waits
// for it to exit, then returns.
//
// Child's environment will have our env vars, plus any given in env.
//
// Child's stdout will be written to output if non-nil, otherwise the
// boot command's stderr.
func (super *Supervisor) RunProgram(ctx context.Context, dir string, opts runOptions, prog string, args ...string) error {
	cmdline := fmt.Sprintf("%s", append([]string{prog}, args...))
	super.logger.WithField("command", cmdline).WithField("dir", dir).Info("executing")

	logprefix := prog
	{
		innerargs := args
		if logprefix == "sudo" {
			for i := 0; i < len(args); i++ {
				if args[i] == "-u" {
					i++
				} else if args[i] == "-E" || strings.Contains(args[i], "=") {
				} else {
					logprefix = args[i]
					innerargs = args[i+1:]
					break
				}
			}
		}
		logprefix = strings.TrimPrefix(logprefix, "/var/lib/arvados/bin/")
		logprefix = strings.TrimPrefix(logprefix, super.tempdir+"/bin/")
		if logprefix == "bundle" && len(innerargs) > 2 && innerargs[0] == "exec" {
			_, dirbase := filepath.Split(dir)
			logprefix = innerargs[1] + "@" + dirbase
		} else if logprefix == "arvados-server" && len(args) > 1 {
			logprefix = args[0]
		}
		if !strings.HasPrefix(dir, "/") {
			logprefix = dir + ": " + logprefix
		}
	}

	cmd := exec.Command(super.lookPath(prog), args...)
	cmd.Stdin = opts.stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	logwriter := &service.LogPrefixer{Writer: super.Stderr, Prefix: []byte("[" + logprefix + "] ")}
	var copiers sync.WaitGroup
	copiers.Add(1)
	go func() {
		io.Copy(logwriter, stderr)
		copiers.Done()
	}()
	copiers.Add(1)
	go func() {
		if opts.output == nil {
			io.Copy(logwriter, stdout)
		} else {
			io.Copy(opts.output, stdout)
		}
		copiers.Done()
	}()

	if strings.HasPrefix(dir, "/") {
		cmd.Dir = dir
	} else {
		cmd.Dir = filepath.Join(super.SourcePath, dir)
	}
	env := append([]string(nil), opts.env...)
	env = append(env, super.environ...)
	cmd.Env = dedupEnv(env)

	if opts.user != "" {
		// Note: We use this approach instead of "sudo"
		// because in certain circumstances (we are pid 1 in a
		// docker container, and our passenger child process
		// changes to pgid 1) the intermediate sudo process
		// notices we have the same pgid as our child and
		// refuses to propagate signals from us to our child,
		// so we can't signal/shutdown our passenger/rails
		// apps. "chpst" or "setuidgid" would work, but these
		// few lines avoid depending on runit/daemontools.
		u, err := user.Lookup(opts.user)
		if err != nil {
			return fmt.Errorf("user.Lookup(%q): %w", opts.user, err)
		}
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid),
				Gid: uint32(gid),
			},
		}
	}

	exited := false
	defer func() { exited = true }()
	go func() {
		<-ctx.Done()
		log := ctxlog.FromContext(ctx).WithFields(logrus.Fields{"dir": dir, "cmdline": cmdline})
		for !exited {
			if cmd.Process == nil {
				log.Debug("waiting for child process to start")
				time.Sleep(time.Second / 2)
			} else {
				log.WithField("PID", cmd.Process.Pid).Debug("sending SIGTERM")
				cmd.Process.Signal(syscall.SIGTERM)
				time.Sleep(5 * time.Second)
				if !exited {
					stdout.Close()
					stderr.Close()
					log.WithField("PID", cmd.Process.Pid).Warn("still waiting for child process to exit 5s after SIGTERM")
				}
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return err
	}
	copiers.Wait()
	err = cmd.Wait()
	if ctx.Err() != nil {
		// Return "context canceled", instead of the "killed"
		// error that was probably caused by the context being
		// canceled.
		return ctx.Err()
	} else if err != nil {
		return fmt.Errorf("%s: error: %v", cmdline, err)
	}
	return nil
}

func (super *Supervisor) autofillConfig() error {
	usedPort := map[string]bool{}
	nextPort := func(host string) (string, error) {
		for {
			port, err := availablePort(host)
			if err != nil {
				port, err = availablePort(super.ListenHost)
			}
			if err != nil {
				return "", err
			}
			if usedPort[port] {
				continue
			}
			usedPort[port] = true
			return port, nil
		}
	}
	if super.cluster.Services.Controller.ExternalURL.Host == "" {
		h, p, err := net.SplitHostPort(super.ControllerAddr)
		if err != nil && super.ControllerAddr != "" {
			return fmt.Errorf("SplitHostPort(ControllerAddr %q): %w", super.ControllerAddr, err)
		}
		if h == "" {
			h = super.ListenHost
		}
		if p == "0" || p == "" {
			p, err = nextPort(h)
			if err != nil {
				return err
			}
		}
		super.cluster.Services.Controller.ExternalURL = arvados.URL{Scheme: "https", Host: net.JoinHostPort(h, p), Path: "/"}
	}
	u := url.URL(super.cluster.Services.Controller.ExternalURL)
	defaultExtHost := u.Hostname()
	for _, svc := range []*arvados.Service{
		&super.cluster.Services.Controller,
		&super.cluster.Services.DispatchCloud,
		&super.cluster.Services.GitHTTP,
		&super.cluster.Services.Health,
		&super.cluster.Services.Keepproxy,
		&super.cluster.Services.Keepstore,
		&super.cluster.Services.RailsAPI,
		&super.cluster.Services.WebDAV,
		&super.cluster.Services.WebDAVDownload,
		&super.cluster.Services.Websocket,
		&super.cluster.Services.Workbench1,
		&super.cluster.Services.Workbench2,
	} {
		if svc.ExternalURL.Host == "" {
			port, err := nextPort(defaultExtHost)
			if err != nil {
				return err
			}
			host := net.JoinHostPort(defaultExtHost, port)
			if svc == &super.cluster.Services.Controller ||
				svc == &super.cluster.Services.GitHTTP ||
				svc == &super.cluster.Services.Health ||
				svc == &super.cluster.Services.Keepproxy ||
				svc == &super.cluster.Services.WebDAV ||
				svc == &super.cluster.Services.WebDAVDownload ||
				svc == &super.cluster.Services.Workbench1 ||
				svc == &super.cluster.Services.Workbench2 {
				svc.ExternalURL = arvados.URL{Scheme: "https", Host: host, Path: "/"}
			} else if svc == &super.cluster.Services.Websocket {
				svc.ExternalURL = arvados.URL{Scheme: "wss", Host: host, Path: "/websocket"}
			}
		}
		if super.NoWorkbench1 && svc == &super.cluster.Services.Workbench1 ||
			super.NoWorkbench2 && svc == &super.cluster.Services.Workbench2 {
			// When workbench1 is disabled, it gets an
			// ExternalURL (so we have a valid listening
			// port to write in our Nginx config) but no
			// InternalURLs (so health checker doesn't
			// complain).
			continue
		}
		if len(svc.InternalURLs) == 0 {
			port, err := nextPort(super.ListenHost)
			if err != nil {
				return err
			}
			host := net.JoinHostPort(super.ListenHost, port)
			svc.InternalURLs = map[arvados.URL]arvados.ServiceInstance{
				{Scheme: "http", Host: host, Path: "/"}: {},
			}
		}
	}
	if super.ClusterType != "production" {
		if super.cluster.SystemRootToken == "" {
			super.cluster.SystemRootToken = randomHexString(64)
		}
		if super.cluster.ManagementToken == "" {
			super.cluster.ManagementToken = randomHexString(64)
		}
		if super.cluster.Collections.BlobSigningKey == "" {
			super.cluster.Collections.BlobSigningKey = randomHexString(64)
		}
		if super.cluster.Users.AnonymousUserToken == "" {
			super.cluster.Users.AnonymousUserToken = randomHexString(64)
		}
		if super.cluster.Containers.DispatchPrivateKey == "" {
			buf, err := ioutil.ReadFile(filepath.Join(super.SourcePath, "lib", "dispatchcloud", "test", "sshkey_dispatch"))
			if err != nil {
				return err
			}
			super.cluster.Containers.DispatchPrivateKey = string(buf)
		}
		super.cluster.TLS.Insecure = true
	}
	if super.ClusterType == "test" {
		// Add a second keepstore process.
		port, err := nextPort(super.ListenHost)
		if err != nil {
			return err
		}
		host := net.JoinHostPort(super.ListenHost, port)
		super.cluster.Services.Keepstore.InternalURLs[arvados.URL{Scheme: "http", Host: host, Path: "/"}] = arvados.ServiceInstance{}

		// Create a directory-backed volume for each keepstore
		// process.
		super.cluster.Volumes = map[string]arvados.Volume{}
		for url := range super.cluster.Services.Keepstore.InternalURLs {
			volnum := len(super.cluster.Volumes)
			datadir := fmt.Sprintf("%s/keep%d.data", super.tempdir, volnum)
			if _, err = os.Stat(datadir + "/."); err == nil {
			} else if !os.IsNotExist(err) {
				return err
			} else if err = os.Mkdir(datadir, 0755); err != nil {
				return err
			}
			super.cluster.Volumes[fmt.Sprintf(super.cluster.ClusterID+"-nyw5e-%015d", volnum)] = arvados.Volume{
				Driver:           "Directory",
				DriverParameters: json.RawMessage(fmt.Sprintf(`{"Root":%q}`, datadir)),
				AccessViaHosts: map[arvados.URL]arvados.VolumeAccess{
					url: {},
				},
				StorageClasses: map[string]bool{
					"default": true,
					"foo":     true,
					"bar":     true,
				},
			}
		}
		super.cluster.StorageClasses = map[string]arvados.StorageClassConfig{
			"default": {Default: true},
			"foo":     {},
			"bar":     {},
		}
	}
	if super.OwnTemporaryDatabase {
		port, err := nextPort("localhost")
		if err != nil {
			return err
		}
		super.cluster.PostgreSQL.Connection = arvados.PostgreSQLConnection{
			"client_encoding": "utf8",
			"host":            "localhost",
			"port":            port,
			"dbname":          "arvados_test",
			"user":            "arvados",
			"password":        "insecure_arvados_test",
		}
	}
	return nil
}

func addrIsLocal(addr string) (bool, error) {
	if h, _, err := net.SplitHostPort(addr); err != nil {
		return false, err
	} else {
		addr = net.JoinHostPort(h, "0")
	}
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		listener.Close()
		return true, nil
	} else if strings.Contains(err.Error(), "cannot assign requested address") {
		return false, nil
	} else {
		return false, err
	}
}

func randomHexString(chars int) string {
	b := make([]byte, chars/2)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func internalPort(svc arvados.Service) (host, port string, err error) {
	if len(svc.InternalURLs) > 1 {
		return "", "", errors.New("internalPort() doesn't work with multiple InternalURLs")
	}
	for u := range svc.InternalURLs {
		u := url.URL(u)
		host, port = u.Hostname(), u.Port()
		switch {
		case port != "":
		case u.Scheme == "https", u.Scheme == "ws":
			port = "443"
		default:
			port = "80"
		}
		return
	}
	return "", "", fmt.Errorf("service has no InternalURLs")
}

func externalPort(svc arvados.Service) (string, error) {
	u := url.URL(svc.ExternalURL)
	if p := u.Port(); p != "" {
		return p, nil
	} else if u.Scheme == "https" || u.Scheme == "wss" {
		return "443", nil
	} else {
		return "80", nil
	}
}

func availablePort(host string) (string, error) {
	ln, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return "", err
	}
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		return "", err
	}
	return port, nil
}

// Try to connect to addr until it works, then close ch. Give up if
// ctx cancels.
func waitForConnect(ctx context.Context, addr string) error {
	ctxlog.FromContext(ctx).WithField("addr", addr).Info("waitForConnect")
	dialer := net.Dialer{Timeout: time.Second}
	for ctx.Err() == nil {
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err != nil {
			time.Sleep(time.Second / 10)
			continue
		}
		conn.Close()
		return nil
	}
	return ctx.Err()
}

func copyConfig(cfg *arvados.Config) *arvados.Config {
	pr, pw := io.Pipe()
	go func() {
		err := json.NewEncoder(pw).Encode(cfg)
		if err != nil {
			panic(err)
		}
		pw.Close()
	}()
	cfg2 := new(arvados.Config)
	err := json.NewDecoder(pr).Decode(cfg2)
	if err != nil {
		panic(err)
	}
	return cfg2
}

func watchConfig(ctx context.Context, logger logrus.FieldLogger, cfgPath string, prevcfg *arvados.Config, fn func()) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.WithError(err).Error("fsnotify setup failed")
		return
	}
	defer watcher.Close()

	err = watcher.Add(cfgPath)
	if err != nil {
		logger.WithError(err).Error("fsnotify watcher failed")
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.WithError(err).Warn("fsnotify watcher reported error")
		case _, ok := <-watcher.Events:
			if !ok {
				return
			}
			for len(watcher.Events) > 0 {
				<-watcher.Events
			}
			loader := config.NewLoader(&bytes.Buffer{}, &logrus.Logger{Out: ioutil.Discard})
			loader.Path = cfgPath
			loader.SkipAPICalls = true
			cfg, err := loader.Load()
			if err != nil {
				logger.WithError(err).Warn("error reloading config file after change detected; ignoring new config for now")
			} else if reflect.DeepEqual(cfg, prevcfg) {
				logger.Debug("config file changed but is still DeepEqual to the existing config")
			} else {
				logger.Debug("config changed, notifying supervisor")
				fn()
				prevcfg = cfg
			}
		}
	}
}
