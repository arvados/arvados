// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package install

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/lib/pq"
)

var Command cmd.Handler = &installCommand{}

const goversion = "1.17.7"

const (
	rubyversion             = "2.7.5"
	bundlerversion          = "2.2.19"
	singularityversion      = "3.9.9"
	pjsversion              = "1.9.8"
	geckoversion            = "0.24.0"
	gradleversion           = "5.3.1"
	nodejsversion           = "v12.22.11"
	devtestDatabasePassword = "insecure_arvados_test"
	workbench2version       = "2454ac35292a79594c32a80430740317ed5005cf"
)

//go:embed arvados.service
var arvadosServiceFile []byte

type installCommand struct {
	ClusterType    string
	SourcePath     string
	PackageVersion string
	EatMyData      bool
}

func (inst *installCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
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
	flags.StringVar(&inst.ClusterType, "type", "production", "cluster `type`: development, test, production, or package")
	flags.StringVar(&inst.SourcePath, "source", "/arvados", "source tree location (required for -type=package)")
	flags.StringVar(&inst.PackageVersion, "package-version", "0.0.0", "version string to embed in executable files")
	flags.BoolVar(&inst.EatMyData, "eatmydata", false, "use eatmydata to speed up install")

	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return code
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	}

	var dev, test, prod, pkg bool
	switch inst.ClusterType {
	case "development":
		dev = true
	case "test":
		test = true
	case "production":
		prod = true
	case "package":
		pkg = true
	default:
		err = fmt.Errorf("invalid cluster type %q (must be 'development', 'test', 'production', or 'package')", inst.ClusterType)
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

	if inst.EatMyData {
		cmd := exec.CommandContext(ctx, "apt-get", "install", "--yes", "--no-install-recommends", "eatmydata")
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return 1
		}
	}

	pkgs := prodpkgs(osv)

	if pkg {
		pkgs = append(pkgs,
			"dpkg-dev",
			"eatmydata", // install it for later steps, even if we're not using it now
		)
	}

	if dev || test || pkg {
		pkgs = append(pkgs,
			"automake",
			"bison",
			"bsdmainutils",
			"build-essential",
			"cadaver",
			"curl",
			"cython3",
			"default-jdk-headless",
			"default-jre-headless",
			"gettext",
			"libattr1-dev",
			"libcrypt-ssleay-perl",
			"libfuse-dev",
			"libgnutls28-dev",
			"libjson-perl",
			"libpam-dev",
			"libpcre3-dev",
			"libpq-dev",
			"libreadline-dev",
			"libssl-dev",
			"libwww-perl",
			"libxml2-dev",
			"libxslt1-dev",
			"linkchecker",
			"lsof",
			"make",
			"net-tools",
			"pandoc",
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
			"rsync",
			"sudo",
			"uuid-dev",
			"wget",
			"xvfb",
			"zlib1g-dev", // services/api
		)
		if test {
			if osv.Debian && osv.Major <= 10 {
				pkgs = append(pkgs, "iceweasel")
			} else {
				pkgs = append(pkgs, "firefox")
			}
		}
		if dev || test {
			pkgs = append(pkgs, "squashfs-tools") // for singularity
			pkgs = append(pkgs, "gnupg")          // for docker install recipe
		}
		switch {
		case osv.Debian && osv.Major >= 11:
			pkgs = append(pkgs, "g++", "libcurl4", "libcurl4-openssl-dev", "perl-modules-5.32")
		case osv.Debian && osv.Major >= 10:
			pkgs = append(pkgs, "g++", "libcurl4", "libcurl4-openssl-dev", "perl-modules")
		case osv.Debian || osv.Ubuntu:
			pkgs = append(pkgs, "g++", "libcurl3", "libcurl3-openssl-dev", "perl-modules")
		case osv.Centos:
			pkgs = append(pkgs, "gcc", "gcc-c++", "libcurl-devel", "postgresql-devel")
		}
		cmd := exec.CommandContext(ctx, "apt-get")
		if inst.EatMyData {
			cmd = exec.CommandContext(ctx, "eatmydata", "apt-get")
		}
		cmd.Args = append(cmd.Args, "install", "--yes", "--no-install-recommends")
		cmd.Args = append(cmd.Args, pkgs...)
		cmd.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return 1
		}
	}

	if dev || test {
		if havedockerversion, err := exec.Command("docker", "--version").CombinedOutput(); err == nil {
			logger.Printf("%s installed, assuming that version is ok", bytes.TrimSuffix(havedockerversion, []byte("\n")))
		} else if osv.Debian {
			var codename string
			switch osv.Major {
			case 10:
				codename = "buster"
			case 11:
				codename = "bullseye"
			default:
				err = fmt.Errorf("don't know how to install docker-ce for debian %d", osv.Major)
				return 1
			}
			err = inst.runBash(`
rm -f /usr/share/keyrings/docker-archive-keyring.gpg
curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo 'deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian/ `+codename+` stable' | \
    tee /etc/apt/sources.list.d/docker.list
apt-get update
DEBIAN_FRONTEND=noninteractive apt-get --yes --no-install-recommends install docker-ce
`, stdout, stderr)
			if err != nil {
				return 1
			}
		} else {
			err = fmt.Errorf("don't know how to install docker for osversion %v", osv)
			return 1
		}
	}

	os.Mkdir("/var/lib/arvados", 0755)
	os.Mkdir("/var/lib/arvados/tmp", 0700)
	if prod || pkg {
		u, er := user.Lookup("www-data")
		if er != nil {
			err = fmt.Errorf("user.Lookup(%q): %w", "www-data", er)
			return 1
		}
		uid, _ := strconv.Atoi(u.Uid)
		gid, _ := strconv.Atoi(u.Gid)
		os.Mkdir("/var/lib/arvados/wwwtmp", 0700)
		err = os.Chown("/var/lib/arvados/wwwtmp", uid, gid)
		if err != nil {
			return 1
		}
	}
	rubymajorversion := rubyversion[:strings.LastIndex(rubyversion, ".")]
	if haverubyversion, err := exec.Command("/var/lib/arvados/bin/ruby", "-v").CombinedOutput(); err == nil && bytes.HasPrefix(haverubyversion, []byte("ruby "+rubyversion)) {
		logger.Print("ruby " + rubyversion + " already installed")
	} else {
		err = inst.runBash(`
tmp="$(mktemp -d)"
trap 'rm -r "${tmp}"' ERR EXIT
wget --progress=dot:giga -O- https://cache.ruby-lang.org/pub/ruby/`+rubymajorversion+`/ruby-`+rubyversion+`.tar.gz | tar -C "${tmp}" -xzf -
cd "${tmp}/ruby-`+rubyversion+`"
./configure --disable-install-static-library --enable-shared --disable-install-doc --prefix /var/lib/arvados
make -j8
make install
/var/lib/arvados/bin/gem install bundler --no-document
`, stdout, stderr)
		if err != nil {
			return 1
		}
	}

	if !prod {
		if havegoversion, err := exec.Command("/usr/local/bin/go", "version").CombinedOutput(); err == nil && bytes.HasPrefix(havegoversion, []byte("go version go"+goversion+" ")) {
			logger.Print("go " + goversion + " already installed")
		} else {
			err = inst.runBash(`
cd /tmp
rm -rf /var/lib/arvados/go/
wget --progress=dot:giga -O- https://storage.googleapis.com/golang/go`+goversion+`.linux-amd64.tar.gz | tar -C /var/lib/arvados -xzf -
ln -sfv /var/lib/arvados/go/bin/* /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}
	}

	if !prod && !pkg {
		if havepjsversion, err := exec.Command("/usr/local/bin/phantomjs", "--version").CombinedOutput(); err == nil && string(havepjsversion) == "1.9.8\n" {
			logger.Print("phantomjs " + pjsversion + " already installed")
		} else {
			err = inst.runBash(`
PJS=phantomjs-`+pjsversion+`-linux-x86_64
wget --progress=dot:giga -O- https://cache.arvados.org/$PJS.tar.bz2 | tar -C /var/lib/arvados -xjf -
ln -sfv /var/lib/arvados/$PJS/bin/phantomjs /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		if havegeckoversion, err := exec.Command("/usr/local/bin/geckodriver", "--version").CombinedOutput(); err == nil && strings.Contains(string(havegeckoversion), " "+geckoversion+" ") {
			logger.Print("geckodriver " + geckoversion + " already installed")
		} else {
			err = inst.runBash(`
GD=v`+geckoversion+`
wget --progress=dot:giga -O- https://github.com/mozilla/geckodriver/releases/download/$GD/geckodriver-$GD-linux64.tar.gz | tar -C /var/lib/arvados/bin -xzf - geckodriver
ln -sfv /var/lib/arvados/bin/geckodriver /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		if havegradleversion, err := exec.Command("/usr/local/bin/gradle", "--version").CombinedOutput(); err == nil && strings.Contains(string(havegradleversion), "Gradle "+gradleversion+"\n") {
			logger.Print("gradle " + gradleversion + " already installed")
		} else {
			err = inst.runBash(`
G=`+gradleversion+`
zip=/var/lib/arvados/tmp/gradle-${G}-bin.zip
trap "rm ${zip}" ERR
wget --progress=dot:giga -O${zip} https://services.gradle.org/distributions/gradle-${G}-bin.zip
unzip -o -d /var/lib/arvados ${zip}
ln -sfv /var/lib/arvados/gradle-${G}/bin/gradle /usr/local/bin/
rm ${zip}
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		if havesingularityversion, err := exec.Command("/var/lib/arvados/bin/singularity", "--version").CombinedOutput(); err == nil && strings.Contains(string(havesingularityversion), singularityversion) {
			logger.Print("singularity " + singularityversion + " already installed")
		} else if dev || test {
			err = inst.runBash(`
S=`+singularityversion+`
tmp=/var/lib/arvados/tmp/singularity
trap "rm -r ${tmp}" ERR EXIT
cd /var/lib/arvados/tmp
git clone https://github.com/sylabs/singularity
cd singularity
git checkout v${S}
./mconfig --prefix=/var/lib/arvados
make -C ./builddir
make -C ./builddir install
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		err = inst.runBash(`
install /usr/bin/nsenter /var/lib/arvados/bin/nsenter
setcap "cap_sys_admin+pei cap_sys_chroot+pei" /var/lib/arvados/bin/nsenter
`, stdout, stderr)
		if err != nil {
			return 1
		}

		// The entry in /etc/locale.gen is "en_US.UTF-8"; once
		// it's installed, locale -a reports it as
		// "en_US.utf8".
		wantlocale := "en_US.UTF-8"
		if havelocales, err := exec.Command("locale", "-a").CombinedOutput(); err == nil && bytes.Contains(havelocales, []byte(strings.Replace(wantlocale+"\n", "UTF-", "utf", 1))) {
			logger.Print("locale " + wantlocale + " already installed")
		} else {
			err = inst.runBash(`sed -i 's/^# *\(`+wantlocale+`\)/\1/' /etc/locale.gen && locale-gen`, stdout, stderr)
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
		if pgLsclusters, err2 := exec.Command("pg_lsclusters", "--no-header").CombinedOutput(); err2 != nil {
			err = fmt.Errorf("pg_lsclusters: %s", err2)
			return 1
		} else if pgclusters := strings.Split(strings.TrimSpace(string(pgLsclusters)), "\n"); len(pgclusters) != 1 {
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
			if err = inst.runBash(`sudo systemctl restart postgresql`, stdout, stderr); err != nil {
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

	if !prod {
		if havenodejsversion, err := exec.Command("/usr/local/bin/node", "--version").CombinedOutput(); err == nil && string(havenodejsversion) == nodejsversion+"\n" {
			logger.Print("nodejs " + nodejsversion + " already installed")
		} else {
			err = inst.runBash(`
NJS=`+nodejsversion+`
wget --progress=dot:giga -O- https://nodejs.org/dist/${NJS}/node-${NJS}-linux-x64.tar.xz | sudo tar -C /var/lib/arvados -xJf -
ln -sfv /var/lib/arvados/node-${NJS}-linux-x64/bin/{node,npm} /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		if haveyarnversion, err := exec.Command("/usr/local/bin/yarn", "--version").CombinedOutput(); err == nil && len(haveyarnversion) > 0 {
			logger.Print("yarn " + strings.TrimSpace(string(haveyarnversion)) + " already installed")
		} else {
			err = inst.runBash(`
npm install -g yarn
ln -sfv /var/lib/arvados/node-`+nodejsversion+`-linux-x64/bin/{yarn,yarnpkg} /usr/local/bin/
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		if havewb2version, err := exec.Command("git", "--git-dir=/var/lib/arvados/arvados-workbench2/.git", "log", "-n1", "--format=%H").CombinedOutput(); err == nil && string(havewb2version) == workbench2version+"\n" {
			logger.Print("workbench2 repo is already at " + workbench2version)
		} else {
			err = inst.runBash(`
V=`+workbench2version+`
cd /var/lib/arvados
if [[ ! -e arvados-workbench2 ]]; then
  git clone https://git.arvados.org/arvados-workbench2.git
  cd arvados-workbench2
  git checkout $V
else
  cd arvados-workbench2
  if ! git checkout $V; then
    git fetch
    git checkout yarn.lock
    git checkout $V
  fi
fi
rm -rf build
`, stdout, stderr)
			if err != nil {
				return 1
			}
		}

		if err = inst.runBash(`
cd /var/lib/arvados/arvados-workbench2
yarn install
`, stdout, stderr); err != nil {
			return 1
		}
	}

	if prod || pkg {
		// Install Go programs to /var/lib/arvados/bin/
		for _, srcdir := range []string{
			"cmd/arvados-client",
			"cmd/arvados-server",
		} {
			fmt.Fprintf(stderr, "building %s...\n", srcdir)
			cmd := exec.Command("go", "install", "-ldflags", "-X git.arvados.org/arvados.git/lib/cmd.version="+inst.PackageVersion+" -X main.version="+inst.PackageVersion+" -s -w")
			cmd.Env = append(cmd.Env, os.Environ()...)
			cmd.Env = append(cmd.Env, "GOBIN=/var/lib/arvados/bin")
			cmd.Dir = filepath.Join(inst.SourcePath, srcdir)
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err = cmd.Run()
			if err != nil {
				return 1
			}
		}

		// Copy assets from source tree to /var/lib/arvados/share
		cmd := exec.Command("install", "-v", "-t", "/var/lib/arvados/share", filepath.Join(inst.SourcePath, "sdk/python/tests/nginx.conf"))
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			return 1
		}

		// Install python SDK and arv-mount in
		// /var/lib/arvados/lib/python.
		//
		// setup.py writes a file in the source directory in
		// order to include the version number in the package
		// itself.  We don't want to write to the source tree
		// (in "arvados-package" context it's mounted
		// readonly) so we run setup.py in a temporary copy of
		// the source dir.
		if err = inst.runBash(`
v=/var/lib/arvados/lib/python
tmp=/var/lib/arvados/tmp/python
python3 -m venv "$v"
. "$v/bin/activate"
pip3 install --no-cache-dir 'setuptools>=18.5' 'pip>=7'
export ARVADOS_BUILDING_VERSION="`+inst.PackageVersion+`"
for src in "`+inst.SourcePath+`/sdk/python" "`+inst.SourcePath+`/services/fuse"; do
  rsync -a --delete-after "$src/" "$tmp/"
  cd "$tmp"
  python3 setup.py install
  cd ..
  rm -rf "$tmp"
done
`, stdout, stderr); err != nil {
			return 1
		}

		// Install Rails apps to /var/lib/arvados/{railsapi,workbench1}/
		for dstdir, srcdir := range map[string]string{
			"railsapi":   "services/api",
			"workbench1": "apps/workbench",
		} {
			fmt.Fprintf(stderr, "building %s...\n", srcdir)
			cmd := exec.Command("rsync",
				"-a", "--no-owner", "--no-group", "--delete-after", "--delete-excluded",
				"--exclude", "/coverage",
				"--exclude", "/log",
				"--exclude", "/node_modules",
				"--exclude", "/tmp",
				"--exclude", "/public/assets",
				"--exclude", "/vendor",
				"--exclude", "/config/environments",
				"./", "/var/lib/arvados/"+dstdir+"/")
			cmd.Dir = filepath.Join(inst.SourcePath, srcdir)
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err = cmd.Run()
			if err != nil {
				return 1
			}
			for _, cmdline := range [][]string{
				{"mkdir", "-p", "log", "public/assets", "tmp", "vendor", ".bundle", "/var/www/.bundle", "/var/www/.gem", "/var/www/.npm", "/var/www/.passenger"},
				{"touch", "log/production.log"},
				{"chown", "-R", "--from=root", "www-data:www-data", "/var/www/.bundle", "/var/www/.gem", "/var/www/.npm", "/var/www/.passenger", "log", "tmp", "vendor", ".bundle", "Gemfile.lock", "config.ru", "config/environment.rb"},
				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/gem", "install", "--user", "--conservative", "--no-document", "bundler:" + bundlerversion},
				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "config", "set", "--local", "deployment", "true"},
				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "config", "set", "--local", "path", "/var/www/.gem"},
				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "config", "set", "--local", "without", "development test diagnostics performance"},
				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "install", "--jobs", fmt.Sprintf("%d", runtime.NumCPU())},

				{"chown", "www-data:www-data", ".", "public/assets"},
				// {"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "config", "set", "--local", "system", "true"},
				{"sudo", "-u", "www-data", "ARVADOS_CONFIG=none", "RAILS_GROUPS=assets", "RAILS_ENV=production", "PATH=/var/lib/arvados/bin:" + os.Getenv("PATH"), "/var/lib/arvados/bin/bundle", "exec", "rake", "npm:install"},
				{"sudo", "-u", "www-data", "ARVADOS_CONFIG=none", "RAILS_GROUPS=assets", "RAILS_ENV=production", "PATH=/var/lib/arvados/bin:" + os.Getenv("PATH"), "/var/lib/arvados/bin/bundle", "exec", "rake", "assets:precompile"},
				{"chown", "root:root", "."},
				{"chown", "-R", "root:root", "public/assets", "vendor"},

				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "exec", "passenger-config", "build-native-support"},
				{"sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "exec", "passenger-config", "install-standalone-runtime"},
			} {
				if cmdline[len(cmdline)-2] == "rake" && dstdir != "workbench1" {
					continue
				}
				cmd = exec.Command(cmdline[0], cmdline[1:]...)
				cmd.Dir = "/var/lib/arvados/" + dstdir
				cmd.Stdout = stdout
				cmd.Stderr = stderr
				fmt.Fprintf(stderr, "... %s\n", cmd.Args)
				err = cmd.Run()
				if err != nil {
					return 1
				}
			}
			cmd = exec.Command("sudo", "-u", "www-data", "/var/lib/arvados/bin/bundle", "exec", "passenger-config", "validate-install")
			cmd.Dir = "/var/lib/arvados/" + dstdir
			cmd.Stdout = stdout
			cmd.Stderr = stderr
			err = cmd.Run()
			if err != nil && !strings.Contains(err.Error(), "exit status 2") {
				// Exit code 2 indicates there were warnings (like
				// "other passenger installations have been detected",
				// which we can't expect to avoid) but no errors.
				// Other non-zero exit codes (1, 9) indicate errors.
				return 1
			}
		}

		// Install workbench2 app to /var/lib/arvados/workbench2/
		if err = inst.runBash(`
cd /var/lib/arvados/arvados-workbench2
VERSION="`+inst.PackageVersion+`" BUILD_NUMBER=1 GIT_COMMIT="`+workbench2version[:9]+`" yarn build
rsync -a --delete-after build/ /var/lib/arvados/workbench2/
`, stdout, stderr); err != nil {
			return 1
		}

		// Install arvados-cli gem (binaries go in
		// /var/lib/arvados/bin)
		if err = inst.runBash(`
/var/lib/arvados/bin/gem install --conservative --no-document arvados-cli
`, stdout, stderr); err != nil {
			return 1
		}

		err = os.WriteFile("/lib/systemd/system/arvados.service", arvadosServiceFile, 0777)
		if err != nil {
			return 1
		}
		if prod {
			// (fpm will do this for us in the pkg case)
			// This is equivalent to "systemd enable", but
			// does not depend on the systemctl program
			// being available:
			symlink := "/etc/systemd/system/multi-user.target.wants/arvados.service"
			err = os.Remove(symlink)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return 1
			}
			err = os.Symlink("/lib/systemd/system/arvados.service", symlink)
			if err != nil {
				return 1
			}
		}

		// Add symlinks in /usr/bin for user-facing programs
		for _, srcdst := range [][]string{
			// go
			{"bin/arvados-client"},
			{"bin/arvados-client", "arv"},
			{"bin/arvados-server"},
			// sdk/cli
			{"bin/arv", "arv-ruby"},
			{"bin/arv-tag"},
			// sdk/python
			{"lib/python/bin/arv-copy"},
			{"lib/python/bin/arv-federation-migrate"},
			{"lib/python/bin/arv-get"},
			{"lib/python/bin/arv-keepdocker"},
			{"lib/python/bin/arv-ls"},
			{"lib/python/bin/arv-migrate-docker19"},
			{"lib/python/bin/arv-normalize"},
			{"lib/python/bin/arv-put"},
			{"lib/python/bin/arv-ws"},
			// services/fuse
			{"lib/python/bin/arv-mount"},
		} {
			src := "/var/lib/arvados/" + srcdst[0]
			if _, err = os.Stat(src); err != nil {
				return 1
			}
			dst := srcdst[len(srcdst)-1]
			_, dst = filepath.Split(dst)
			dst = "/usr/bin/" + dst
			err = os.Remove(dst)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return 1
			}
			err = os.Symlink(src, dst)
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
	Centos bool
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
	case "centos":
		osv.Centos = true
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

func (inst *installCommand) runBash(script string, stdout, stderr io.Writer) error {
	cmd := exec.Command("bash", "-")
	if inst.EatMyData {
		cmd = exec.Command("eatmydata", "bash", "-")
	}
	cmd.Stdin = bytes.NewBufferString("set -ex -o pipefail\n" + script)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func prodpkgs(osv osversion) []string {
	pkgs := []string{
		"ca-certificates",
		"curl",
		"fuse",
		"git",
		"gitolite3",
		"graphviz",
		"haveged",
		"libcurl3-gnutls",
		"libxslt1.1",
		"nginx",
		"python",
		"sudo",
	}
	if osv.Debian || osv.Ubuntu {
		if osv.Debian && osv.Major == 8 {
			pkgs = append(pkgs, "libgnutls-deb0-28") // sdk/cwl
		} else if osv.Debian && osv.Major >= 10 || osv.Ubuntu && osv.Major >= 16 {
			pkgs = append(pkgs, "python3-distutils") // sdk/cwl
		}
		return append(pkgs,
			"mime-support", // keep-web
		)
	} else if osv.Centos {
		return append(pkgs,
			"fuse-libs", // services/fuse
			"mailcap",   // keep-web
		)
	} else {
		panic("os version not supported")
	}
}

func ProductionDependencies() ([]string, error) {
	osv, err := identifyOS()
	if err != nil {
		return nil, err
	}
	return prodpkgs(osv), nil
}
