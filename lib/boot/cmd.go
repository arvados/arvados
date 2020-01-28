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
	"git.arvados.org/arvados.git/lib/controller"
	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"github.com/sirupsen/logrus"
)

var Command cmd.Handler = &bootCommand{}

type bootCommand struct {
	sourcePath  string // e.g., /home/username/src/arvados
	libPath     string // e.g., /var/lib/arvados
	clusterType string // e.g., production

	cluster *arvados.Cluster
	stdout  io.Writer
	stderr  io.Writer

	tempdir string

	setupRubyOnce sync.Once
	setupRubyErr  error
	goMutex       sync.Mutex
}

func (boot *bootCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	boot.stdout = stdout
	boot.stderr = stderr
	log := ctxlog.New(stderr, "json", "info")

	var err error
	defer func() {
		if err != nil {
			log.WithError(err).Info("exiting")
		}
	}()

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.SetOutput(stderr)
	loader := config.NewLoader(stdin, log)
	loader.SetupFlags(flags)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	flags.StringVar(&boot.sourcePath, "source", ".", "arvados source tree `directory`")
	flags.StringVar(&boot.libPath, "lib", "/var/lib/arvados", "`directory` to install dependencies and library files")
	flags.StringVar(&boot.clusterType, "type", "production", "cluster `type`: development, test, or production")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	} else if boot.clusterType != "development" && boot.clusterType != "test" && boot.clusterType != "production" {
		err = fmt.Errorf("cluster type must be 'development', 'test', or 'production'")
		return 2
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 1
	}
	if !strings.HasPrefix(boot.sourcePath, "/") {
		boot.sourcePath = filepath.Join(cwd, boot.sourcePath)
	}
	boot.sourcePath, err = filepath.EvalSymlinks(boot.sourcePath)
	if err != nil {
		return 1
	}

	loader.SkipAPICalls = true
	cfg, err := loader.Load()
	if err != nil {
		return 1
	}

	boot.tempdir, err = ioutil.TempDir("", "arvados-server-boot-")
	if err != nil {
		return 1
	}
	defer os.RemoveAll(boot.tempdir)

	// Fill in any missing config keys, and write the resulting
	// config in the temp dir for child services to use.
	err = boot.autofillConfig(cfg, log)
	if err != nil {
		return 1
	}
	conffile, err := os.OpenFile(filepath.Join(boot.tempdir, "config.yml"), os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return 1
	}
	defer conffile.Close()
	err = json.NewEncoder(conffile).Encode(cfg)
	if err != nil {
		return 1
	}
	err = conffile.Close()
	if err != nil {
		return 1
	}
	os.Setenv("ARVADOS_CONFIG", conffile.Name())
	arvados.DefaultConfigFile = conffile.Name()
	os.Setenv("RAILS_ENV", boot.clusterType)

	// Now that we have the config, replace the bootstrap logger
	// with a new one according to the logging config.
	boot.cluster, err = cfg.GetCluster("")
	if err != nil {
		return 1
	}
	log = ctxlog.New(stderr, boot.cluster.SystemLogs.Format, boot.cluster.SystemLogs.LogLevel)
	logger := log.WithFields(logrus.Fields{
		"PID": os.Getpid(),
	})
	ctx := ctxlog.Context(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT)
	go func() {
		for sig := range ch {
			logger.WithField("signal", sig).Info("caught signal")
			cancel()
		}
	}()

	for _, dir := range []string{boot.libPath, filepath.Join(boot.libPath, "bin")} {
		if _, err = os.Stat(filepath.Join(dir, ".")); os.IsNotExist(err) {
			err = os.Mkdir(dir, 0755)
			if err != nil {
				return 1
			}
		} else if err != nil {
			return 1
		}
	}
	os.Setenv("PATH", filepath.Join(boot.libPath, "bin")+":"+os.Getenv("PATH"))

	err = boot.installGoProgram(ctx, "cmd/arvados-server")
	if err != nil {
		return 1
	}

	var wg sync.WaitGroup
	for _, cmpt := range []component{
		{name: "nginx", runFunc: runNginx},
		{name: "controller", cmdHandler: controller.Command},
		{name: "dispatchcloud", cmdHandler: dispatchcloud.Command, notIfTest: true},
		{name: "git-httpd", goProg: "services/arv-git-httpd"},
		{name: "health", goProg: "services/health"},
		{name: "keep-balance", goProg: "services/keep-balance", notIfTest: true},
		{name: "keepproxy", goProg: "services/keepproxy"},
		{name: "keepstore", goProg: "services/keepstore", svc: boot.cluster.Services.Keepstore},
		{name: "keep-web", goProg: "services/keep-web"},
		{name: "railsAPI", svc: boot.cluster.Services.RailsAPI, railsApp: "services/api"},
		{name: "ws", goProg: "services/ws"},
	} {
		cmpt := cmpt
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer cancel()
			logger.WithField("component", cmpt.name).Info("starting")
			err := cmpt.Run(ctx, boot, stdout, stderr)
			if err != nil {
				logger.WithError(err).WithField("component", cmpt.name).Info("exited")
			}
		}()
	}
	if boot.waitUntilReady(ctx) {
		fmt.Fprintln(stdout, boot.cluster.Services.Controller.ExternalURL)
	}
	<-ctx.Done()
	wg.Wait()
	return 0
}

func (boot *bootCommand) waitUntilReady(ctx context.Context) bool {
	agg := health.Aggregator{Cluster: boot.cluster}
	for waiting := true; waiting; {
		time.Sleep(time.Second)
		if ctx.Err() != nil {
			return false
		}
		resp := agg.ClusterHealth()
		// The overall health check (resp.Health=="OK") might
		// never pass due to missing components (like
		// arvados-dispatch-cloud in a test cluster), so
		// instead we wait for all configured components to
		// pass.
		waiting = false
		for _, check := range resp.Checks {
			if check.Health != "OK" {
				waiting = true
			}
		}
	}
	return true
}

func (boot *bootCommand) installGoProgram(ctx context.Context, srcpath string) error {
	boot.goMutex.Lock()
	defer boot.goMutex.Unlock()
	return boot.RunProgram(ctx, filepath.Join(boot.sourcePath, srcpath), nil, []string{"GOPATH=" + boot.libPath}, "go", "install")
}

func (boot *bootCommand) setupRubyEnv() error {
	boot.setupRubyOnce.Do(func() {
		buf, err := exec.Command("gem", "env", "gempath").Output() // /var/lib/arvados/.gem/ruby/2.5.0/bin:...
		if err != nil || len(buf) == 0 {
			boot.setupRubyErr = fmt.Errorf("gem env gempath: %v", err)
		}
		gempath := string(bytes.Split(buf, []byte{':'})[0])
		os.Setenv("PATH", gempath+"/bin:"+os.Getenv("PATH"))
		os.Setenv("GEM_HOME", gempath)
		os.Setenv("GEM_PATH", gempath)
	})
	return boot.setupRubyErr
}

// Run prog with args, using dir as working directory. If ctx is
// cancelled while the child is running, RunProgram terminates the
// child, waits for it to exit, then returns.
//
// Child's environment will have our env vars, plus any given in env.
//
// Child's stdout will be written to output if non-nil, otherwise the
// boot command's stderr.
func (boot *bootCommand) RunProgram(ctx context.Context, dir string, output io.Writer, env []string, prog string, args ...string) error {
	cmdline := fmt.Sprintf("%s", append([]string{prog}, args...))
	fmt.Fprintf(boot.stderr, "%s executing in %s\n", cmdline, dir)
	cmd := exec.Command(prog, args...)
	if output == nil {
		cmd.Stdout = boot.stderr
	} else {
		cmd.Stdout = output
	}
	cmd.Stderr = boot.stderr
	if strings.HasPrefix(dir, "/") {
		cmd.Dir = dir
	} else {
		cmd.Dir = filepath.Join(boot.sourcePath, dir)
	}
	if env != nil {
		cmd.Env = append(env, os.Environ()...)
	}
	go func() {
		<-ctx.Done()
		log := ctxlog.FromContext(ctx).WithFields(logrus.Fields{"dir": dir, "cmdline": cmdline})
		for cmd.ProcessState == nil {
			// Child hasn't exited yet
			if cmd.Process == nil {
				log.Infof("waiting for child process to start")
				time.Sleep(time.Second)
			} else {
				log.WithField("PID", cmd.Process.Pid).Info("sending SIGTERM")
				cmd.Process.Signal(syscall.SIGTERM)
				log.WithField("PID", cmd.Process.Pid).Info("waiting for child process to exit after SIGTERM")
				time.Sleep(5 * time.Second)
			}
		}
	}()
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("%s: error: %v", cmdline, err)
	}
	return nil
}

type component struct {
	name       string
	svc        arvados.Service
	cmdHandler cmd.Handler
	runFunc    func(ctx context.Context, boot *bootCommand, stdout, stderr io.Writer) error
	railsApp   string // source dir in arvados tree, e.g., "services/api"
	goProg     string // source dir in arvados tree, e.g., "services/keepstore"
	notIfTest  bool   // don't run this component on a test cluster
}

func (cmpt *component) Run(ctx context.Context, boot *bootCommand, stdout, stderr io.Writer) error {
	if cmpt.notIfTest && boot.clusterType == "test" {
		fmt.Fprintf(stderr, "skipping component %q in %s mode\n", cmpt.name, boot.clusterType)
		<-ctx.Done()
		return nil
	}
	fmt.Fprintf(stderr, "starting component %q\n", cmpt.name)
	if cmpt.cmdHandler != nil {
		errs := make(chan error, 1)
		go func() {
			defer close(errs)
			exitcode := cmpt.cmdHandler.RunCommand(cmpt.name, nil, bytes.NewBuffer(nil), stdout, stderr)
			if exitcode != 0 {
				errs <- fmt.Errorf("exit code %d", exitcode)
			}
		}()
		select {
		case err := <-errs:
			return err
		case <-ctx.Done():
			// cmpt.cmdHandler.RunCommand() doesn't have
			// access to our context, so it won't shut
			// down by itself. We just abandon it.
			return nil
		}
	}
	if cmpt.goProg != "" {
		boot.RunProgram(ctx, cmpt.goProg, nil, nil, "go", "install")
		if ctx.Err() != nil {
			return nil
		}
		_, basename := filepath.Split(cmpt.goProg)
		if len(cmpt.svc.InternalURLs) > 0 {
			// Run one for each URL
			var wg sync.WaitGroup
			for u := range cmpt.svc.InternalURLs {
				u := u
				wg.Add(1)
				go func() {
					defer wg.Done()
					boot.RunProgram(ctx, boot.tempdir, nil, []string{"ARVADOS_SERVICE_INTERNAL_URL=" + u.String()}, basename)
				}()
			}
			wg.Wait()
			return nil
		} else {
			// Just run one
			boot.RunProgram(ctx, boot.tempdir, nil, nil, basename)
		}
	}
	if cmpt.runFunc != nil {
		return cmpt.runFunc(ctx, boot, stdout, stderr)
	}
	if cmpt.railsApp != "" {
		port, err := internalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("bug: no InternalURLs for component %q: %v", cmpt.name, cmpt.svc.InternalURLs)
		}
		err = boot.setupRubyEnv()
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		err = boot.RunProgram(ctx, cmpt.railsApp, &buf, nil, "gem", "list", "--details", "bundler")
		if err != nil {
			return err
		}
		for _, version := range []string{"1.11.0", "1.17.3", "2.0.2"} {
			if !strings.Contains(buf.String(), "("+version+")") {
				err = boot.RunProgram(ctx, cmpt.railsApp, nil, nil, "gem", "install", "--user", "bundler:1.11", "bundler:1.17.3", "bundler:2.0.2")
				if err != nil {
					return err
				}
				break
			}
		}
		err = boot.RunProgram(ctx, cmpt.railsApp, nil, nil, "bundle", "install", "--jobs", "4", "--path", filepath.Join(os.Getenv("HOME"), ".gem"))
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.railsApp, nil, nil, "bundle", "exec", "passenger-config", "build-native-support")
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.railsApp, nil, nil, "bundle", "exec", "passenger-config", "install-standalone-runtime")
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.railsApp, nil, nil, "bundle", "exec", "passenger-config", "validate-install")
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.railsApp, nil, nil, "bundle", "exec", "passenger", "start", "-p", port)
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("bug: component %q has nothing to run", cmpt.name)
}

func (boot *bootCommand) autofillConfig(cfg *arvados.Config, log logrus.FieldLogger) error {
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}
	port := 9000
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
	} {
		if svc == &cluster.Services.DispatchCloud && boot.clusterType == "test" {
			continue
		}
		if len(svc.InternalURLs) == 0 {
			port++
			svc.InternalURLs = map[arvados.URL]arvados.ServiceInstance{
				arvados.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port)}: arvados.ServiceInstance{},
			}
		}
		if svc.ExternalURL.Host == "" && (svc == &cluster.Services.Controller ||
			svc == &cluster.Services.GitHTTP ||
			svc == &cluster.Services.Keepproxy ||
			svc == &cluster.Services.WebDAV ||
			svc == &cluster.Services.WebDAVDownload ||
			svc == &cluster.Services.Websocket) {
			port++
			svc.ExternalURL = arvados.URL{Scheme: "https", Host: fmt.Sprintf("localhost:%d", port)}
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
	if boot.clusterType != "production" && cluster.Containers.DispatchPrivateKey == "" {
		buf, err := ioutil.ReadFile(filepath.Join(boot.sourcePath, "lib", "dispatchcloud", "test", "sshkey_dispatch"))
		if err != nil {
			return err
		}
		cluster.Containers.DispatchPrivateKey = string(buf)
	}
	if boot.clusterType != "production" {
		cluster.TLS.Insecure = true
	}
	if boot.clusterType == "test" {
		// Add a second keepstore process.
		port++
		cluster.Services.Keepstore.InternalURLs[arvados.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port)}] = arvados.ServiceInstance{}

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
			cluster.Volumes[fmt.Sprintf("zzzzz-nyw5e-%015d", volnum)] = arvados.Volume{
				Driver:           "Directory",
				DriverParameters: json.RawMessage(fmt.Sprintf(`{"Root":%q}`, datadir)),
				AccessViaHosts: map[arvados.URL]arvados.VolumeAccess{
					url: {},
				},
			}
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
