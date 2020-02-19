// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/sirupsen/logrus"
)

var Command cmd.Handler = bootCommand{}

type bootTask interface {
	// Execute the task. Run should return nil when the task is
	// done enough to satisfy a dependency relationship (e.g., the
	// service is running and ready). If the task starts a
	// goroutine that fails after Run returns (e.g., the service
	// shuts down), it should call cancel.
	Run(ctx context.Context, fail func(error), boot *Booter) error
	String() string
}

type bootCommand struct{}

func (bootCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	boot := &Booter{
		Stderr: stderr,
		logger: ctxlog.New(stderr, "json", "info"),
	}

	ctx := ctxlog.Context(context.Background(), boot.logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range ch {
			boot.logger.WithField("signal", sig).Info("caught signal")
			cancel()
		}
	}()

	var err error
	defer func() {
		if err != nil {
			boot.logger.WithError(err).Info("exiting")
		}
	}()

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader := config.NewLoader(stdin, boot.logger)
	loader.SetupFlags(flags)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	flags.StringVar(&boot.SourcePath, "source", ".", "arvados source tree `directory`")
	flags.StringVar(&boot.LibPath, "lib", "/var/lib/arvados", "`directory` to install dependencies and library files")
	flags.StringVar(&boot.ClusterType, "type", "production", "cluster `type`: development, test, or production")
	flags.StringVar(&boot.ListenHost, "listen-host", "localhost", "host name or interface address for service listeners")
	flags.StringVar(&boot.ControllerAddr, "controller-address", ":0", "desired controller address, `host:port` or `:port`")
	flags.BoolVar(&boot.OwnTemporaryDatabase, "own-temporary-database", false, "bring up a postgres server and create a temporary database")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	} else if boot.ClusterType != "development" && boot.ClusterType != "test" && boot.ClusterType != "production" {
		err = fmt.Errorf("cluster type must be 'development', 'test', or 'production'")
		return 2
	}

	loader.SkipAPICalls = true
	cfg, err := loader.Load()
	if err != nil {
		return 1
	}

	boot.Start(ctx, cfg)
	defer boot.Stop()
	if url, ok := boot.WaitReady(); ok {
		fmt.Fprintln(stdout, url)
		<-ctx.Done() // wait for signal
		return 0
	} else {
		return 1
	}
}

type Booter struct {
	SourcePath           string // e.g., /home/username/src/arvados
	LibPath              string // e.g., /var/lib/arvados
	ClusterType          string // e.g., production
	ListenHost           string // e.g., localhost
	ControllerAddr       string // e.g., 127.0.0.1:8000
	OwnTemporaryDatabase bool
	Stderr               io.Writer

	logger  logrus.FieldLogger
	cluster *arvados.Cluster

	ctx           context.Context
	cancel        context.CancelFunc
	done          chan struct{}
	healthChecker *health.Aggregator
	tasksReady    map[string]chan bool

	tempdir    string
	configfile string
	environ    []string // for child processes

	setupRubyOnce sync.Once
	setupRubyErr  error
	goMutex       sync.Mutex
}

func (boot *Booter) Start(ctx context.Context, cfg *arvados.Config) {
	boot.ctx, boot.cancel = context.WithCancel(ctx)
	boot.done = make(chan struct{})
	go func() {
		err := boot.run(cfg)
		if err != nil {
			fmt.Fprintln(boot.Stderr, err)
		}
		close(boot.done)
	}()
}

func (boot *Booter) run(cfg *arvados.Config) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(boot.SourcePath, "/") {
		boot.SourcePath = filepath.Join(cwd, boot.SourcePath)
	}
	boot.SourcePath, err = filepath.EvalSymlinks(boot.SourcePath)
	if err != nil {
		return err
	}

	boot.tempdir, err = ioutil.TempDir("", "arvados-server-boot-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(boot.tempdir)

	// Fill in any missing config keys, and write the resulting
	// config in the temp dir for child services to use.
	err = boot.autofillConfig(cfg, boot.logger)
	if err != nil {
		return err
	}
	conffile, err := os.OpenFile(filepath.Join(boot.tempdir, "config.yml"), os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer conffile.Close()
	err = json.NewEncoder(conffile).Encode(cfg)
	if err != nil {
		return err
	}
	err = conffile.Close()
	if err != nil {
		return err
	}
	boot.configfile = conffile.Name()

	boot.environ = os.Environ()
	boot.setEnv("ARVADOS_CONFIG", boot.configfile)
	boot.setEnv("RAILS_ENV", boot.ClusterType)
	boot.prependEnv("PATH", filepath.Join(boot.LibPath, "bin")+":")

	boot.cluster, err = cfg.GetCluster("")
	if err != nil {
		return err
	}
	// Now that we have the config, replace the bootstrap logger
	// with a new one according to the logging config.
	loglevel := boot.cluster.SystemLogs.LogLevel
	if s := os.Getenv("ARVADOS_DEBUG"); s != "" && s != "0" {
		loglevel = "debug"
	}
	boot.logger = ctxlog.New(boot.Stderr, boot.cluster.SystemLogs.Format, loglevel).WithFields(logrus.Fields{
		"PID": os.Getpid(),
	})
	boot.healthChecker = &health.Aggregator{Cluster: boot.cluster}

	for _, dir := range []string{boot.LibPath, filepath.Join(boot.LibPath, "bin")} {
		if _, err = os.Stat(filepath.Join(dir, ".")); os.IsNotExist(err) {
			err = os.Mkdir(dir, 0755)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	err = boot.installGoProgram(boot.ctx, "cmd/arvados-server")
	if err != nil {
		return err
	}
	err = boot.setupRubyEnv()
	if err != nil {
		return err
	}

	tasks := []bootTask{
		createCertificates{},
		runPostgreSQL{},
		runNginx{},
		runServiceCommand{name: "controller", svc: boot.cluster.Services.Controller, depends: []bootTask{runPostgreSQL{}}},
		runGoProgram{src: "services/arv-git-httpd"},
		runGoProgram{src: "services/health"},
		runGoProgram{src: "services/keepproxy", depends: []bootTask{runPassenger{src: "services/api"}}},
		runGoProgram{src: "services/keepstore", svc: boot.cluster.Services.Keepstore},
		runGoProgram{src: "services/keep-web"},
		runGoProgram{src: "services/ws", depends: []bootTask{runPostgreSQL{}}},
		installPassenger{src: "services/api"},
		runPassenger{src: "services/api", svc: boot.cluster.Services.RailsAPI, depends: []bootTask{createCertificates{}, runPostgreSQL{}, installPassenger{src: "services/api"}}},
		installPassenger{src: "apps/workbench", depends: []bootTask{installPassenger{src: "services/api"}}}, // dependency ensures workbench doesn't delay api startup
		runPassenger{src: "apps/workbench", svc: boot.cluster.Services.Workbench1, depends: []bootTask{installPassenger{src: "apps/workbench"}}},
		seedDatabase{},
	}
	if boot.ClusterType != "test" {
		tasks = append(tasks,
			runServiceCommand{name: "dispatch-cloud", svc: boot.cluster.Services.Controller},
			runGoProgram{src: "services/keep-balance"},
		)
	}
	boot.tasksReady = map[string]chan bool{}
	for _, task := range tasks {
		boot.tasksReady[task.String()] = make(chan bool)
	}
	for _, task := range tasks {
		task := task
		fail := func(err error) {
			if boot.ctx.Err() != nil {
				return
			}
			boot.cancel()
			boot.logger.WithField("task", task.String()).WithError(err).Error("task failed")
		}
		go func() {
			boot.logger.WithField("task", task.String()).Info("starting")
			err := task.Run(boot.ctx, fail, boot)
			if err != nil {
				fail(err)
				return
			}
			close(boot.tasksReady[task.String()])
		}()
	}
	err = boot.wait(boot.ctx, tasks...)
	if err != nil {
		return err
	}
	<-boot.ctx.Done()
	return boot.ctx.Err()
}

func (boot *Booter) wait(ctx context.Context, tasks ...bootTask) error {
	for _, task := range tasks {
		ch, ok := boot.tasksReady[task.String()]
		if !ok {
			return fmt.Errorf("no such task: %s", task)
		}
		boot.logger.WithField("task", task.String()).Info("waiting")
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (boot *Booter) Stop() {
	boot.cancel()
	<-boot.done
}

func (boot *Booter) WaitReady() (*arvados.URL, bool) {
	for waiting := true; waiting; {
		time.Sleep(time.Second)
		if boot.ctx.Err() != nil {
			return nil, false
		}
		if boot.healthChecker == nil {
			// not set up yet
			continue
		}
		resp := boot.healthChecker.ClusterHealth()
		// The overall health check (resp.Health=="OK") might
		// never pass due to missing components (like
		// arvados-dispatch-cloud in a test cluster), so
		// instead we wait for all configured components to
		// pass.
		waiting = false
		for target, check := range resp.Checks {
			if check.Health != "OK" {
				waiting = true
				boot.logger.WithField("target", target).Debug("waiting")
			}
		}
	}
	u := boot.cluster.Services.Controller.ExternalURL
	return &u, true
}

func (boot *Booter) prependEnv(key, prepend string) {
	for i, s := range boot.environ {
		if strings.HasPrefix(s, key+"=") {
			boot.environ[i] = key + "=" + prepend + s[len(key)+1:]
			return
		}
	}
	boot.environ = append(boot.environ, key+"="+prepend)
}

func (boot *Booter) setEnv(key, val string) {
	for i, s := range boot.environ {
		if strings.HasPrefix(s, key+"=") {
			boot.environ[i] = key + "=" + val
			return
		}
	}
	boot.environ = append(boot.environ, key+"="+val)
}

func (boot *Booter) installGoProgram(ctx context.Context, srcpath string) error {
	boot.goMutex.Lock()
	defer boot.goMutex.Unlock()
	return boot.RunProgram(ctx, filepath.Join(boot.SourcePath, srcpath), nil, []string{"GOPATH=" + boot.LibPath}, "go", "install")
}

func (boot *Booter) setupRubyEnv() error {
	buf, err := exec.Command("gem", "env", "gempath").Output() // /var/lib/arvados/.gem/ruby/2.5.0/bin:...
	if err != nil || len(buf) == 0 {
		return fmt.Errorf("gem env gempath: %v", err)
	}
	gempath := string(bytes.Split(buf, []byte{':'})[0])
	boot.prependEnv("PATH", gempath+"/bin:")
	boot.setEnv("GEM_HOME", gempath)
	boot.setEnv("GEM_PATH", gempath)
	return nil
}

func (boot *Booter) lookPath(prog string) string {
	for _, val := range boot.environ {
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

// Run prog with args, using dir as working directory. If ctx is
// cancelled while the child is running, RunProgram terminates the
// child, waits for it to exit, then returns.
//
// Child's environment will have our env vars, plus any given in env.
//
// Child's stdout will be written to output if non-nil, otherwise the
// boot command's stderr.
func (boot *Booter) RunProgram(ctx context.Context, dir string, output io.Writer, env []string, prog string, args ...string) error {
	cmdline := fmt.Sprintf("%s", append([]string{prog}, args...))
	fmt.Fprintf(boot.Stderr, "%s executing in %s\n", cmdline, dir)

	logprefix := prog
	if prog == "bundle" && len(args) > 2 && args[0] == "exec" {
		logprefix = args[1]
	}
	if !strings.HasPrefix(dir, "/") {
		logprefix = dir + ": " + logprefix
	}
	stderr := &logPrefixer{Writer: boot.Stderr, Prefix: []byte("[" + logprefix + "] ")}

	cmd := exec.Command(boot.lookPath(prog), args...)
	if output == nil {
		cmd.Stdout = stderr
	} else {
		cmd.Stdout = output
	}
	cmd.Stderr = stderr
	if strings.HasPrefix(dir, "/") {
		cmd.Dir = dir
	} else {
		cmd.Dir = filepath.Join(boot.SourcePath, dir)
	}
	cmd.Env = append(env, boot.environ...)

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
					log.WithField("PID", cmd.Process.Pid).Warn("still waiting for child process to exit 5s after SIGTERM")
				}
			}
		}
	}()

	err := cmd.Run()
	if err != nil && ctx.Err() == nil {
		// Only report errors that happen before the context ends.
		return fmt.Errorf("%s: error: %v", cmdline, err)
	}
	return nil
}

func (boot *Booter) autofillConfig(cfg *arvados.Config, log logrus.FieldLogger) error {
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}
	usedPort := map[string]bool{}
	nextPort := func() string {
		for {
			port, err := availablePort(":0")
			if err != nil {
				panic(err)
			}
			if usedPort[port] {
				continue
			}
			usedPort[port] = true
			return port
		}
	}
	if cluster.Services.Controller.ExternalURL.Host == "" {
		h, p, err := net.SplitHostPort(boot.ControllerAddr)
		if err != nil {
			return err
		}
		if h == "" {
			h = boot.ListenHost
		}
		if p == "0" {
			p, err = availablePort(":0")
			if err != nil {
				return err
			}
			usedPort[p] = true
		}
		cluster.Services.Controller.ExternalURL = arvados.URL{Scheme: "https", Host: net.JoinHostPort(h, p)}
	}
	for _, svc := range []*arvados.Service{
		&cluster.Services.Controller,
		&cluster.Services.DispatchCloud,
		&cluster.Services.GitHTTP,
		&cluster.Services.Health,
		&cluster.Services.Keepproxy,
		&cluster.Services.Keepstore,
		&cluster.Services.RailsAPI,
		&cluster.Services.WebDAV,
		&cluster.Services.WebDAVDownload,
		&cluster.Services.Websocket,
		&cluster.Services.Workbench1,
	} {
		if svc == &cluster.Services.DispatchCloud && boot.ClusterType == "test" {
			continue
		}
		if svc.ExternalURL.Host == "" && (svc == &cluster.Services.Controller ||
			svc == &cluster.Services.GitHTTP ||
			svc == &cluster.Services.Keepproxy ||
			svc == &cluster.Services.WebDAV ||
			svc == &cluster.Services.WebDAVDownload ||
			svc == &cluster.Services.Websocket ||
			svc == &cluster.Services.Workbench1) {
			svc.ExternalURL = arvados.URL{Scheme: "https", Host: fmt.Sprintf("%s:%s", boot.ListenHost, nextPort())}
		}
		if len(svc.InternalURLs) == 0 {
			svc.InternalURLs = map[arvados.URL]arvados.ServiceInstance{
				arvados.URL{Scheme: "http", Host: fmt.Sprintf("%s:%s", boot.ListenHost, nextPort())}: arvados.ServiceInstance{},
			}
		}
	}
	if cluster.SystemRootToken == "" {
		cluster.SystemRootToken = randomHexString(64)
	}
	if cluster.ManagementToken == "" {
		cluster.ManagementToken = randomHexString(64)
	}
	if cluster.API.RailsSessionSecretToken == "" {
		cluster.API.RailsSessionSecretToken = randomHexString(64)
	}
	if cluster.Collections.BlobSigningKey == "" {
		cluster.Collections.BlobSigningKey = randomHexString(64)
	}
	if boot.ClusterType != "production" && cluster.Containers.DispatchPrivateKey == "" {
		buf, err := ioutil.ReadFile(filepath.Join(boot.SourcePath, "lib", "dispatchcloud", "test", "sshkey_dispatch"))
		if err != nil {
			return err
		}
		cluster.Containers.DispatchPrivateKey = string(buf)
	}
	if boot.ClusterType != "production" {
		cluster.TLS.Insecure = true
	}
	if boot.ClusterType == "test" {
		// Add a second keepstore process.
		cluster.Services.Keepstore.InternalURLs[arvados.URL{Scheme: "http", Host: fmt.Sprintf("%s:%s", boot.ListenHost, nextPort())}] = arvados.ServiceInstance{}

		// Create a directory-backed volume for each keepstore
		// process.
		cluster.Volumes = map[string]arvados.Volume{}
		for url := range cluster.Services.Keepstore.InternalURLs {
			volnum := len(cluster.Volumes)
			datadir := fmt.Sprintf("%s/keep%d.data", boot.tempdir, volnum)
			if _, err = os.Stat(datadir + "/."); err == nil {
			} else if !os.IsNotExist(err) {
				return err
			} else if err = os.Mkdir(datadir, 0777); err != nil {
				return err
			}
			cluster.Volumes[fmt.Sprintf(cluster.ClusterID+"-nyw5e-%015d", volnum)] = arvados.Volume{
				Driver:           "Directory",
				DriverParameters: json.RawMessage(fmt.Sprintf(`{"Root":%q}`, datadir)),
				AccessViaHosts: map[arvados.URL]arvados.VolumeAccess{
					url: {},
				},
			}
		}
	}
	if boot.OwnTemporaryDatabase {
		cluster.PostgreSQL.Connection = arvados.PostgreSQLConnection{
			"client_encoding": "utf8",
			"host":            "localhost",
			"port":            nextPort(),
			"dbname":          "arvados_test",
			"user":            "arvados",
			"password":        "insecure_arvados_test",
		}
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

func randomHexString(chars int) string {
	b := make([]byte, chars/2)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func internalPort(svc arvados.Service) (string, error) {
	for u := range svc.InternalURLs {
		if _, p, err := net.SplitHostPort(u.Host); err != nil {
			return "", err
		} else if p != "" {
			return p, nil
		} else if u.Scheme == "https" {
			return "443", nil
		} else {
			return "80", nil
		}
	}
	return "", fmt.Errorf("service has no InternalURLs")
}

func externalPort(svc arvados.Service) (string, error) {
	if _, p, err := net.SplitHostPort(svc.ExternalURL.Host); err != nil {
		return "", err
	} else if p != "" {
		return p, nil
	} else if svc.ExternalURL.Scheme == "https" {
		return "443", nil
	} else {
		return "80", nil
	}
}

func availablePort(addr string) (string, error) {
	ln, err := net.Listen("tcp", addr)
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
