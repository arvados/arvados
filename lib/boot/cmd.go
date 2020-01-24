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
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

var Command cmd.Handler = &bootCommand{}

type bootCommand struct {
	sourcePath  string // e.g., /home/username/src/arvados
	libPath     string // e.g., /var/lib/arvados
	clusterType string // e.g., production

	stdout io.Writer
	stderr io.Writer

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

	tempdir, err := ioutil.TempDir("", "arvados-server-boot-")
	if err != nil {
		return 1
	}
	defer os.RemoveAll(tempdir)

	// Fill in any missing config keys, and write the resulting
	// config in the temp dir for child services to use.
	autofillConfig(cfg, log)
	conffile, err := os.OpenFile(filepath.Join(tempdir, "config.yml"), os.O_CREATE|os.O_WRONLY, 0777)
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

	os.Setenv("RAILS_ENV", boot.clusterType)

	// Now that we have the config, replace the bootstrap logger
	// with a new one according to the logging config.
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}
	log = ctxlog.New(stderr, cluster.SystemLogs.Format, cluster.SystemLogs.LogLevel)
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
		{name: "controller", svc: cluster.Services.Controller, cmdArgs: []string{"-config", conffile.Name()}, cmdHandler: controller.Command},
		// {name: "dispatchcloud", cmdArgs: []string{"-config", conffile.Name()}, cmdHandler: dispatchcloud.Command},
		{name: "railsAPI", svc: cluster.Services.RailsAPI, src: "services/api"},
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
	<-ctx.Done()
	wg.Wait()
	return 0
}

func (boot *bootCommand) installGoProgram(ctx context.Context, srcpath string) error {
	boot.goMutex.Lock()
	defer boot.goMutex.Unlock()
	env := append([]string{"GOPATH=" + boot.libPath}, os.Environ()...)
	return boot.RunProgram(ctx, filepath.Join(boot.sourcePath, srcpath), nil, env, "go", "install")
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
		cmd.Env = env
	}
	go func() {
		<-ctx.Done()
		cmd.Process.Signal(syscall.SIGINT)
		for range time.Tick(5 * time.Second) {
			if cmd.ProcessState != nil {
				break
			}
			ctxlog.FromContext(ctx).WithField("process", cmd.Process).Infof("waiting for child process to exit after SIGINT")
			cmd.Process.Signal(syscall.SIGINT)
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
	cmdArgs    []string
	src        string // source dir in arvados tree, e.g., "services/keepstore"
}

func (cmpt *component) Run(ctx context.Context, boot *bootCommand, stdout, stderr io.Writer) error {
	fmt.Fprintf(stderr, "starting component %q\n", cmpt.name)
	if cmpt.cmdHandler != nil {
		errs := make(chan error, 1)
		go func() {
			defer close(errs)
			exitcode := cmpt.cmdHandler.RunCommand(cmpt.name, cmpt.cmdArgs, bytes.NewBuffer(nil), stdout, stderr)
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
	if cmpt.src != "" {
		port := "-"
		for u := range cmpt.svc.InternalURLs {
			if _, p, err := net.SplitHostPort(u.Host); err != nil {
				return err
			} else if p != "" {
				port = p
			} else if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
			break
		}
		if port == "-" {
			return fmt.Errorf("bug: no InternalURLs for component %q: %v", cmpt.name, cmpt.svc.InternalURLs)
		}

		err := boot.setupRubyEnv()
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		err = boot.RunProgram(ctx, cmpt.src, &buf, nil, "gem", "list", "--details", "bundler")
		if err != nil {
			return err
		}
		for _, version := range []string{"1.11.0", "1.17.3", "2.0.2"} {
			if !strings.Contains(buf.String(), "("+version+")") {
				err = boot.RunProgram(ctx, cmpt.src, nil, nil, "gem", "install", "--user", "bundler:1.11", "bundler:1.17.3", "bundler:2.0.2")
				if err != nil {
					return err
				}
				break
			}
		}
		err = boot.RunProgram(ctx, cmpt.src, nil, nil, "bundle", "install", "--jobs", "4", "--path", filepath.Join(os.Getenv("HOME"), ".gem"))
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.src, nil, nil, "bundle", "exec", "passenger-config", "build-native-support")
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.src, nil, nil, "bundle", "exec", "passenger-config", "install-standalone-runtime")
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.src, nil, nil, "bundle", "exec", "passenger-config", "validate-install")
		if err != nil {
			return err
		}
		err = boot.RunProgram(ctx, cmpt.src, nil, nil, "bundle", "exec", "passenger", "start", "-p", port)
		if err != nil {
			return err
		}
	}
	return fmt.Errorf("bug: component %q has nothing to run", cmpt.name)
}

func autofillConfig(cfg *arvados.Config, log logrus.FieldLogger) {
	cluster, err := cfg.GetCluster("")
	if err != nil {
		panic(err)
	}
	port := 9000
	for _, svc := range []*arvados.Service{
		&cluster.Services.Controller,
		&cluster.Services.DispatchCloud,
		&cluster.Services.RailsAPI,
	} {
		if len(svc.InternalURLs) == 0 {
			port++
			svc.InternalURLs = map[arvados.URL]arvados.ServiceInstance{
				arvados.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", port)}: arvados.ServiceInstance{},
			}
		}
	}
	if cluster.Services.Controller.ExternalURL.Host == "" {
		for k := range cluster.Services.Controller.InternalURLs {
			cluster.Services.Controller.ExternalURL = k
		}
	}
	if cluster.SystemRootToken == "" {
		cluster.SystemRootToken = randomHexString(64)
	}
	if cluster.API.RailsSessionSecretToken == "" {
		cluster.API.RailsSessionSecretToken = randomHexString(64)
	}
	if cluster.Collections.BlobSigningKey == "" {
		cluster.Collections.BlobSigningKey = randomHexString(64)
	}
	cfg.Clusters[cluster.ClusterID] = *cluster
}

func randomHexString(chars int) string {
	b := make([]byte, chars/2)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}
