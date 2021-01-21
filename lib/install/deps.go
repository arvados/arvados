// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package install

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/lib/pq"
)

var Command cmd.Handler = installCommand{}

const devtestDatabasePassword = "insecure_arvados_test"

type installCommand struct{}

func (installCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	ctx := ctxlog.Context(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error
	defer func() {
		if err != nil {
			logger.WithError(err).Info("exiting")
		}
	}()

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.SetOutput(stderr)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	clusterType := flags.String("type", "production", "cluster `type`: development, test, or production")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
		return 2
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	}

	var dev, test, prod bool
	switch *clusterType {
	case "development":
		dev = true
	case "test":
		test = true
	case "production":
		prod = true
	default:
		err = fmt.Errorf("invalid cluster type %q (must be 'development', 'test', or 'production')", *clusterType)
		return 2
	}

	if prod {
		err = errors.New("production install is not yet implemented")
		return 1
	}

	osv, err := identifyOS()
	if err != nil {
		return 1
	}

	listdir, err := os.Open("/var/lib/apt/lists")
	if err != nil {
		logger.Warnf("error while checking whether to run apt-get update: %s", err)
	} else if names, _ := listdir.Readdirnames(1); len(names) == 0 {
		// Special case for a base docker image where the
		// package cache has been deleted and all "apt-get
		// install" commands will fail unless we fetch repos.
		cmd := exec.CommandContext(ctx, "apt-get", "update")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return 1
		}
	}

	if dev || test {
		debs := []string{
			"bison",
			"bsdmainutils",
			"build-essential",
			"ca-certificates",
			"cadaver",
			"curl",
			"cython3",
			"daemontools", // lib/boot uses setuidgid to drop privileges when running as root
			"default-jdk-headless",
			"default-jre-headless",
			"fuse",
			"gettext",
			"git",
			"gitolite3",
			"graphviz",
			"haveged",
			"iceweasel",
			"libattr1-dev",
			"libcrypt-ssleay-perl",
			"libcrypt-ssleay-perl",
			"libcurl3-gnutls",
			"libcurl4-openssl-dev",
			"libfuse-dev",
			"libgnutls28-dev",
			"libjson-perl",
			"libjson-perl",
			"libpam-dev",
			"libpcre3-dev",
			"libpq-dev",
			"libreadline-dev",
			"libssl-dev",
			"libwww-perl",
			"libxml2-dev",
			"libxslt1.1",
			"linkchecker",
			"lsof",
			"net-tools",
			"nginx",
			"pandoc",
			"perl-modules",
			"pkg-config",
			"postgresql",
			"postgresql-contrib",
			"python3-dev",
			"python3-venv",
			"python3-virtualenv",
			"r-base",
			"r-cran-testthat",
			"r-cran-devtools",
			"r-cran-knitr",
			"r-cran-markdown",
			"r-cran-roxygen2",
			"r-cran-xml",
			"sudo",
			"wget",
			"xvfb",
			"zlib1g-dev",
		}
		switch {
		case osv.Debian && osv.Major >= 10:
			debs = append(debs, "libcurl4")
		default:
			debs = append(debs, "libcurl3")
		}
		cmd := exec.CommandContext(ctx, "apt-get", "install", "--yes", "--no-install-recommends")
		cmd.Args = append(cmd.Args, debs...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return 1
		}
	}

	os.Mkdir("/var/lib/arvados", 0755)
	rubyversion := "2.5.7"
	if haverubyversion, err := exec.Command("/var/lib/arvados/bin/ruby", "-v").CombinedOutput(); err == nil && bytes.HasPrefix(haverubyversion, []byte("ruby "+rubyversion)) {
		logger.Print("ruby " + rubyversion + " already installed")
	} else {
		err = runBash(`
mkdir -p /var/lib/arvados/tmp
tmp=/var/lib/arvados/tmp/ruby-`+rubyversion+`
trap "rm -r ${tmp}" ERR
wget --progress=dot:giga -O- https://cache.ruby-lang.org/pub/ruby/2.5/ruby-`+rubyversion+`.tar.gz | tar -C /var/lib/arvados/tmp -xzf -
cd ${tmp}
./configure --disable-install-doc --prefix /var/lib/arvados
make -j4
make install
/var/lib/arvados/bin/gem install bundler
rm -r ${tmp}
`, stdout, stderr)
		if err != nil {
			return 1
		}
	}

	if !prod {
		goversion := "1.14"
		if havegoversion, err := exec.Command("/usr/local/bin/go", "version").CombinedOutput(); err == nil && bytes.HasPrefix(havegoversion, []byte("go version go"+goversion+" ")) {
			logger.Print("go " + goversion + " already installed")
		} else {
			err = runBash(`
cd /tmp
wget --progress=dot:giga -O- https://storage.googleapis.com/golang/go`+goversion+`.linux-amd64.tar.gz | tar -C /var/lib/arvados -xzf -
ln -sf /var/lib/arvados/go/bin/* /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		pjsversion := "1.9.8"
		if havepjsversion, err := exec.Command("/usr/local/bin/phantomjs", "--version").CombinedOutput(); err == nil && string(havepjsversion) == "1.9.8\n" {
			logger.Print("phantomjs " + pjsversion + " already installed")
		} else {
			err = runBash(`
PJS=phantomjs-`+pjsversion+`-linux-x86_64
wget --progress=dot:giga -O- https://bitbucket.org/ariya/phantomjs/downloads/$PJS.tar.bz2 | tar -C /var/lib/arvados -xjf -
ln -sf /var/lib/arvados/$PJS/bin/phantomjs /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		geckoversion := "0.24.0"
		if havegeckoversion, err := exec.Command("/usr/local/bin/geckodriver", "--version").CombinedOutput(); err == nil && strings.Contains(string(havegeckoversion), " "+geckoversion+" ") {
			logger.Print("geckodriver " + geckoversion + " already installed")
		} else {
			err = runBash(`
GD=v`+geckoversion+`
wget --progress=dot:giga -O- https://github.com/mozilla/geckodriver/releases/download/$GD/geckodriver-$GD-linux64.tar.gz | tar -C /var/lib/arvados/bin -xzf - geckodriver
ln -sf /var/lib/arvados/bin/geckodriver /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		nodejsversion := "v10.23.1"
		if havenodejsversion, err := exec.Command("/usr/local/bin/node", "--version").CombinedOutput(); err == nil && string(havenodejsversion) == nodejsversion+"\n" {
			logger.Print("nodejs " + nodejsversion + " already installed")
		} else {
			err = runBash(`
NJS=`+nodejsversion+`
wget --progress=dot:giga -O- https://nodejs.org/dist/${NJS}/node-${NJS}-linux-x64.tar.xz | sudo tar -C /var/lib/arvados -xJf -
ln -sf /var/lib/arvados/node-${NJS}-linux-x64/bin/{node,npm} /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		gradleversion := "5.3.1"
		if havegradleversion, err := exec.Command("/usr/local/bin/gradle", "--version").CombinedOutput(); err == nil && strings.Contains(string(havegradleversion), "Gradle "+gradleversion+"\n") {
			logger.Print("gradle " + gradleversion + " already installed")
		} else {
			err = runBash(`
G=`+gradleversion+`
mkdir -p /var/lib/arvados/tmp
zip=/var/lib/arvados/tmp/gradle-${G}-bin.zip
trap "rm ${zip}" ERR
wget --progress=dot:giga -O${zip} https://services.gradle.org/distributions/gradle-${G}-bin.zip
unzip -o -d /var/lib/arvados ${zip}
ln -sf /var/lib/arvados/gradle-${G}/bin/gradle /usr/local/bin/
rm ${zip}
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		// The entry in /etc/locale.gen is "en_US.UTF-8"; once
		// it's installed, locale -a reports it as
		// "en_US.utf8".
		wantlocale := "en_US.UTF-8"
		if havelocales, err := exec.Command("locale", "-a").CombinedOutput(); err == nil && bytes.Contains(havelocales, []byte(strings.Replace(wantlocale+"\n", "UTF-", "utf", 1))) {
			logger.Print("locale " + wantlocale + " already installed")
		} else {
			err = runBash(`sed -i 's/^# *\(`+wantlocale+`\)/\1/' /etc/locale.gen && locale-gen`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		var pgc struct {
			Version       string
			Cluster       string
			Port          int
			Status        string
			Owner         string
			DataDirectory string
			LogFile       string
		}
		if pg_lsclusters, err2 := exec.Command("pg_lsclusters", "--no-header").CombinedOutput(); err2 != nil {
			err = fmt.Errorf("pg_lsclusters: %s", err2)
			return 1
		} else if pgclusters := strings.Split(strings.TrimSpace(string(pg_lsclusters)), "\n"); len(pgclusters) != 1 {
			logger.Warnf("pg_lsclusters returned %d postgresql clusters -- skipping postgresql initdb/startup, hope that's ok", len(pgclusters))
		} else if _, err = fmt.Sscanf(pgclusters[0], "%s %s %d %s %s %s %s", &pgc.Version, &pgc.Cluster, &pgc.Port, &pgc.Status, &pgc.Owner, &pgc.DataDirectory, &pgc.LogFile); err != nil {
			err = fmt.Errorf("error parsing pg_lsclusters output: %s", err)
			return 1
		} else if pgc.Status == "online" {
			logger.Infof("postgresql cluster %s-%s is online", pgc.Version, pgc.Cluster)
		} else {
			logger.Infof("postgresql cluster %s-%s is %s; trying to start", pgc.Version, pgc.Cluster, pgc.Status)
			cmd := exec.Command("pg_ctlcluster", "--foreground", pgc.Version, pgc.Cluster, "start")
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err = cmd.Start()
			if err != nil {
				return 1
			}
			defer func() {
				cmd.Process.Signal(syscall.SIGTERM)
				logger.Info("sent SIGTERM; waiting for postgres to shut down")
				cmd.Wait()
			}()
			err = waitPostgreSQLReady()
			if err != nil {
				return 1
			}
		}

		if os.Getpid() == 1 {
			// We are the init process (presumably in a
			// docker container) so although postgresql is
			// installed, it's not running, and initdb
			// might never have been run.
		}

		var needcoll []string
		// If the en_US.UTF-8 locale wasn't installed when
		// postgresql initdb ran, it needs to be added
		// explicitly before we can use it in our test suite.
		for _, collname := range []string{"en_US", "en_US.UTF-8"} {
			cmd := exec.Command("sudo", "-u", "postgres", "psql", "-t", "-c", "SELECT 1 FROM pg_catalog.pg_collation WHERE collname='"+collname+"' AND collcollate IN ('en_US.UTF-8', 'en_US.utf8')")
			cmd.Dir = "/"
			out, err2 := cmd.CombinedOutput()
			if err != nil {
				err = fmt.Errorf("error while checking postgresql collations: %s", err2)
				return 1
			}
			if strings.Contains(string(out), "1") {
				logger.Infof("postgresql supports collation %s", collname)
			} else {
				needcoll = append(needcoll, collname)
			}
		}
		if len(needcoll) > 0 && os.Getpid() != 1 {
			// In order for the CREATE COLLATION statement
			// below to work, the locale must have existed
			// when PostgreSQL started up. If we're
			// running as init, we must have started
			// PostgreSQL ourselves after installing the
			// locales. Otherwise, it might need a
			// restart, so we attempt to restart it with
			// systemd.
			if err = runBash(`sudo systemctl restart postgresql`, stdout, stderr); err != nil {
				logger.Warn("`systemctl restart postgresql` failed; hoping postgresql does not need to be restarted")
			} else if err = waitPostgreSQLReady(); err != nil {
				return 1
			}
		}
		for _, collname := range needcoll {
			cmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", "CREATE COLLATION \""+collname+"\" (LOCALE = \"en_US.UTF-8\")")
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			cmd.Dir = "/"
			err = cmd.Run()
			if err != nil {
				err = fmt.Errorf("error adding postgresql collation %s: %s", collname, err)
				return 1
			}
		}

		withstuff := "WITH LOGIN SUPERUSER ENCRYPTED PASSWORD " + pq.QuoteLiteral(devtestDatabasePassword)
		cmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", "ALTER ROLE arvados "+withstuff)
		cmd.Dir = "/"
		if err := cmd.Run(); err == nil {
			logger.Print("arvados role exists; superuser privileges added, password updated")
		} else {
			cmd := exec.Command("sudo", "-u", "postgres", "psql", "-c", "CREATE ROLE arvados "+withstuff)
			cmd.Dir = "/"
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err = cmd.Run()
			if err != nil {
				return 1
			}
		}
	}

	return 0
}

type osversion struct {
	Debian bool
	Ubuntu bool
	Major  int
}

func identifyOS() (osversion, error) {
	var osv osversion
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return osv, err
	}
	defer f.Close()

	kv := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		toks := strings.SplitN(line, "=", 2)
		if len(toks) != 2 {
			return osv, fmt.Errorf("invalid line in /etc/os-release: %q", line)
		}
		k := toks[0]
		v := strings.Trim(toks[1], `"`)
		if v == toks[1] {
			v = strings.Trim(v, `'`)
		}
		kv[k] = v
	}
	if err = scanner.Err(); err != nil {
		return osv, err
	}
	switch kv["ID"] {
	case "ubuntu":
		osv.Ubuntu = true
	case "debian":
		osv.Debian = true
	default:
		return osv, fmt.Errorf("unsupported ID in /etc/os-release: %q", kv["ID"])
	}
	vstr := kv["VERSION_ID"]
	if i := strings.Index(vstr, "."); i > 0 {
		vstr = vstr[:i]
	}
	osv.Major, err = strconv.Atoi(vstr)
	if err != nil {
		return osv, fmt.Errorf("incomprehensible VERSION_ID in /etc/os-release: %q", kv["VERSION_ID"])
	}
	return osv, nil
}

func waitPostgreSQLReady() error {
	for deadline := time.Now().Add(10 * time.Second); ; {
		output, err := exec.Command("pg_isready").CombinedOutput()
		if err == nil {
			return nil
		} else if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for pg_isready (%q)", output)
		} else {
			time.Sleep(time.Second)
		}
	}
}

func runBash(script string, stdout, stderr io.Writer) error {
	cmd := exec.Command("bash", "-")
	cmd.Stdin = bytes.NewBufferString("set -ex -o pipefail\n" + script)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
