#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

. `dirname "$(readlink -f "$0")"`/libcloud-pin.sh

COLUMNS=80
. `dirname "$(readlink -f "$0")"`/run-library.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Install and test Arvados components.

Exit non-zero if any tests fail.

Syntax:
        $(basename $0) WORKSPACE=/path/to/arvados [options]

Options:

--skip FOO     Do not test the FOO component.
--skip sanity  Skip initial dev environment sanity checks.
--skip install Do not run any install steps. Just run tests.
               You should provide GOPATH, GEMHOME, and VENVDIR options
               from a previous invocation if you use this option.
--only FOO     Do not test anything except the FOO component.
--temp DIR     Install components and dependencies under DIR instead of
               making a new temporary directory. Implies --leave-temp.
--leave-temp   Do not remove GOPATH, virtualenv, and other temp dirs at exit.
               Instead, show the path to give as --temp to reuse them in
               subsequent invocations.
--repeat N     Repeat each install/test step until it succeeds N times.
--retry        Prompt to retry if an install or test suite fails.
--only-install Run specific install step
--short        Skip (or scale down) some slow tests.
--interactive  Set up, then prompt for test/install steps to perform.
WORKSPACE=path Arvados source tree to test.
CONFIGSRC=path Dir with config.yml file containing PostgreSQL section for use by tests. (required)
services/api_test="TEST=test/functional/arvados/v1/collections_controller_test.rb"
               Restrict apiserver tests to the given file
sdk/python_test="--test-suite tests.test_keep_locator"
               Restrict Python SDK tests to the given class
apps/workbench_test="TEST=test/integration/pipeline_instances_test.rb"
               Restrict Workbench tests to the given file
services/arv-git-httpd_test="-check.vv"
               Show all log messages, even when tests pass (also works
               with services/keepstore_test etc.)
ARVADOS_DEBUG=1
               Print more debug messages
envvar=value   Set \$envvar to value. Primarily useful for WORKSPACE,
               *_test, and other examples shown above.

Assuming "--skip install" is not given, all components are installed
into \$GOPATH, \$VENDIR, and \$GEMHOME before running any tests. Many
test suites depend on other components being installed, and installing
everything tends to be quicker than debugging dependencies.

As a special concession to the current CI server config, CONFIGSRC
defaults to $HOME/arvados-api-server if that directory exists.

More information and background:

https://arvados.org/projects/arvados/wiki/Running_tests

Available tests:

apps/workbench (*)
apps/workbench_units (*)
apps/workbench_functionals (*)
apps/workbench_integration (*)
apps/workbench_benchmark
apps/workbench_profile
cmd/arvados-client
cmd/arvados-server
doc
lib/cli
lib/cmd
lib/controller
lib/controller/federation
lib/controller/railsproxy
lib/controller/router
lib/controller/rpc
lib/crunchstat
lib/cloud
lib/cloud/azure
lib/cloud/cloudtest
lib/dispatchcloud
lib/dispatchcloud/container
lib/dispatchcloud/scheduler
lib/dispatchcloud/ssh_executor
lib/dispatchcloud/worker
lib/service
services/api
services/arv-git-httpd
services/crunchstat
services/dockercleaner
services/fuse
services/fuse:py3
services/health
services/keep-web
services/keepproxy
services/keepstore
services/keep-balance
services/login-sync
services/nodemanager
services/nodemanager_integration
services/crunch-run
services/crunch-dispatch-local
services/crunch-dispatch-slurm
services/ws
sdk/cli
sdk/pam
sdk/pam:py3
sdk/python
sdk/python:py3
sdk/ruby
sdk/go/arvados
sdk/go/arvadosclient
sdk/go/auth
sdk/go/dispatch
sdk/go/keepclient
sdk/go/health
sdk/go/httpserver
sdk/go/manifest
sdk/go/blockdigest
sdk/go/asyncbuf
sdk/go/stats
sdk/go/crunchrunner
sdk/cwl
sdk/R
sdk/java-v2
tools/sync-groups
tools/crunchstat-summary
tools/crunchstat-summary:py3
tools/keep-exercise
tools/keep-rsync
tools/keep-block-check

(*) apps/workbench is shorthand for apps/workbench_units +
    apps/workbench_functionals + apps/workbench_integration

EOF

# First make sure to remove any ARVADOS_ variables from the calling
# environment that could interfere with the tests.
unset $(env | cut -d= -f1 | grep \^ARVADOS_)

# Reset other variables that could affect our [tests'] behavior by
# accident.
GITDIR=
GOPATH=
VENVDIR=
VENV3DIR=
PYTHONPATH=
GEMHOME=
PERLINSTALLBASE=
R_LIBS=
export LANG=en_US.UTF-8

short=
only_install=
temp=
temp_preserve=

clear_temp() {
    if [[ -z "$temp" ]]; then
        # we didn't even get as far as making a temp dir
        :
    elif [[ -z "$temp_preserve" ]]; then
        rm -rf "$temp"
    else
        echo "Leaving behind temp dirs in $temp"
    fi
}

fatal() {
    clear_temp
    echo >&2 "Fatal: $* (encountered in ${FUNCNAME[1]} at ${BASH_SOURCE[1]} line ${BASH_LINENO[0]})"
    exit 1
}

exit_cleanly() {
    trap - INT
    if which create-plot-data-from-log.sh >/dev/null; then
        create-plot-data-from-log.sh $BUILD_NUMBER "$WORKSPACE/apps/workbench/log/test.log" "$WORKSPACE/apps/workbench/log/"
    fi
    rotate_logfile "$WORKSPACE/apps/workbench/log/" "test.log"
    stop_services
    rotate_logfile "$WORKSPACE/services/api/log/" "test.log"
    report_outcomes
    clear_temp
    exit ${#failures}
}

sanity_checks() {
    [[ -n "${skip[sanity]}" ]] && return 0
    ( [[ -n "$WORKSPACE" ]] && [[ -d "$WORKSPACE/services" ]] ) \
        || fatal "WORKSPACE environment variable not set to a source directory (see: $0 --help)"
    [[ -n "$CONFIGSRC" ]] \
	|| fatal "CONFIGSRC environment not set (see: $0 --help)"
    [[ -s "$CONFIGSRC/config.yml" ]] \
	|| fatal "'$CONFIGSRC/config.yml' is empty or not found (see: $0 --help)"
    echo Checking dependencies:
    echo "locale: ${LANG}"
    [[ "$(locale charmap)" = "UTF-8" ]] \
        || fatal "Locale '${LANG}' is broken/missing. Try: echo ${LANG} | sudo tee -a /etc/locale.gen && sudo locale-gen"
    echo -n 'virtualenv: '
    virtualenv --version \
        || fatal "No virtualenv. Try: apt-get install virtualenv (on ubuntu: python-virtualenv)"
    echo -n 'ruby: '
    ruby -v \
        || fatal "No ruby. Install >=2.1.9 (using rbenv, rvm, or source)"
    echo -n 'go: '
    go version \
        || fatal "No go binary. See http://golang.org/doc/install"
    [[ $(go version) =~ go1.([0-9]+) ]] && [[ ${BASH_REMATCH[1]} -ge 12 ]] \
        || fatal "Go >= 1.12 required. See http://golang.org/doc/install"
    echo -n 'gcc: '
    gcc --version | egrep ^gcc \
        || fatal "No gcc. Try: apt-get install build-essential"
    echo -n 'fuse.h: '
    find /usr/include -path '*fuse/fuse.h' | egrep --max-count=1 . \
        || fatal "No fuse/fuse.h. Try: apt-get install libfuse-dev"
    echo -n 'gnutls.h: '
    find /usr/include -path '*gnutls/gnutls.h' | egrep --max-count=1 . \
        || fatal "No gnutls/gnutls.h. Try: apt-get install libgnutls28-dev"
    echo -n 'Python2 pyconfig.h: '
    find /usr/include -path '*/python2*/pyconfig.h' | egrep --max-count=1 . \
        || fatal "No Python2 pyconfig.h. Try: apt-get install python2.7-dev"
    echo -n 'Python3 pyconfig.h: '
    find /usr/include -path '*/python3*/pyconfig.h' | egrep --max-count=1 . \
        || fatal "No Python3 pyconfig.h. Try: apt-get install python3-dev"
    which netstat \
        || fatal "No netstat. Try: apt-get install net-tools"
    echo -n 'nginx: '
    PATH="$PATH:/sbin:/usr/sbin:/usr/local/sbin" nginx -v \
        || fatal "No nginx. Try: apt-get install nginx"
    echo -n 'perl: '
    perl -v | grep version \
        || fatal "No perl. Try: apt-get install perl"
    for mod in ExtUtils::MakeMaker JSON LWP Net::SSL; do
        echo -n "perl $mod: "
        perl -e "use $mod; print \"\$$mod::VERSION\\n\"" \
            || fatal "No $mod. Try: apt-get install perl-modules libcrypt-ssleay-perl libjson-perl libwww-perl"
    done
    echo -n 'gitolite: '
    which gitolite \
        || fatal "No gitolite. Try: apt-get install gitolite3"
    echo -n 'npm: '
    npm --version \
        || fatal "No npm. Try: wget -O- https://nodejs.org/dist/v6.11.2/node-v6.11.2-linux-x64.tar.xz | sudo tar -C /usr/local -xJf - && sudo ln -s ../node-v6.11.2-linux-x64/bin/{node,npm} /usr/local/bin/"
    echo -n 'cadaver: '
    cadaver --version | grep -w cadaver \
          || fatal "No cadaver. Try: apt-get install cadaver"
    echo -n 'libattr1 xattr.h: '
    find /usr/include -path '*/attr/xattr.h' | egrep --max-count=1 . \
        || fatal "No libattr1 xattr.h. Try: apt-get install libattr1-dev"
    echo -n 'libcurl curl.h: '
    find /usr/include -path '*/curl/curl.h' | egrep --max-count=1 . \
        || fatal "No libcurl curl.h. Try: apt-get install libcurl4-gnutls-dev"
    echo -n 'libpq libpq-fe.h: '
    find /usr/include -path '*/postgresql/libpq-fe.h' | egrep --max-count=1 . \
        || fatal "No libpq libpq-fe.h. Try: apt-get install libpq-dev"
    echo -n 'postgresql: '
    psql --version || fatal "No postgresql. Try: apt-get install postgresql postgresql-client-common"
    echo -n 'phantomjs: '
    phantomjs --version || fatal "No phantomjs. Try: apt-get install phantomjs"
    echo -n 'xvfb: '
    which Xvfb || fatal "No xvfb. Try: apt-get install xvfb"
    echo -n 'graphviz: '
    dot -V || fatal "No graphviz. Try: apt-get install graphviz"
    echo -n 'geckodriver: '
    geckodriver --version | grep ^geckodriver || echo "No geckodriver. Try: wget -O- https://github.com/mozilla/geckodriver/releases/download/v0.23.0/geckodriver-v0.23.0-linux64.tar.gz | sudo tar -C /usr/local/bin -xzf - geckodriver"

    if [[ "$NEED_SDK_R" = true ]]; then
      # R SDK stuff
      echo -n 'R: '
      which Rscript || fatal "No Rscript. Try: apt-get install r-base"
      echo -n 'testthat: '
      Rscript -e "library('testthat')" || fatal "No testthat. Try: apt-get install r-cran-testthat"
      # needed for roxygen2, needed for devtools, needed for R sdk
      pkg-config --exists libxml-2.0 || fatal "No libxml2. Try: apt-get install libxml2-dev"
      # needed for pkgdown, builds R SDK doc pages
      which pandoc || fatal "No pandoc. Try: apt-get install pandoc"
    fi
    echo 'procs with /dev/fuse open:'
    find /proc/*/fd -lname /dev/fuse 2>/dev/null | cut -d/ -f3 | xargs --no-run-if-empty ps -lywww
    echo 'grep fuse /proc/self/mountinfo:'
    grep fuse /proc/self/mountinfo
}

rotate_logfile() {
  # i.e.  rotate_logfile "$WORKSPACE/apps/workbench/log/" "test.log"
  # $BUILD_NUMBER is set by Jenkins if this script is being called as part of a Jenkins run
  if [[ -f "$1/$2" ]]; then
    THEDATE=`date +%Y%m%d%H%M%S`
    mv "$1/$2" "$1/$THEDATE-$BUILD_NUMBER-$2"
    gzip "$1/$THEDATE-$BUILD_NUMBER-$2"
  fi
}

declare -a failures
declare -A skip
declare -A only
declare -A testargs
skip[apps/workbench_profile]=1
# nodemanager_integration tests are not reliable, see #12061.
skip[services/nodemanager_integration]=1

while [[ -n "$1" ]]
do
    arg="$1"; shift
    case "$arg" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --skip)
            skip[$1]=1; shift
            ;;
        --only)
            only[$1]=1; skip[$1]=""; shift
            ;;
        --short)
            short=1
            ;;
        --interactive)
            interactive=1
            ;;
        --skip-install)
            skip[install]=1
            ;;
        --only-install)
            only_install="$1"; shift
            ;;
        --temp)
            temp="$1"; shift
            temp_preserve=1
            ;;
        --leave-temp)
            temp_preserve=1
            ;;
        --repeat)
            repeat=$((${1}+0)); shift
            ;;
        --retry)
            retry=1
            ;;
        *_test=*)
            suite="${arg%%_test=*}"
            args="${arg#*=}"
            testargs["$suite"]="$args"
            ;;
        *=*)
            eval export $(echo $arg | cut -d= -f1)=\"$(echo $arg | cut -d= -f2-)\"
            ;;
        *)
            echo >&2 "$0: Unrecognized option: '$arg'. Try: $0 --help"
            exit 1
            ;;
    esac
done

# R SDK installation is very slow (~360s in a clean environment) and only
# required when testing it. Skip that step if it is not needed.
NEED_SDK_R=true

if [[ ${#only[@]} -ne 0 ]] &&
   [[ -z "${only['sdk/R']}" && -z "${only['doc']}" ]]; then
  NEED_SDK_R=false
fi

if [[ ${skip["sdk/R"]} == 1 && ${skip["doc"]} == 1 ]]; then
  NEED_SDK_R=false
fi

if [[ $NEED_SDK_R == false ]]; then
	echo "R SDK not needed, it will not be installed."
fi

checkpidfile() {
    svc="$1"
    pid="$(cat "$WORKSPACE/tmp/${svc}.pid")"
    if [[ -z "$pid" ]] || ! kill -0 "$pid"; then
        tail $WORKSPACE/tmp/${1}*.log
        echo "${svc} pid ${pid} not running"
        return 1
    fi
    echo "${svc} pid ${pid} ok"
}

checkhealth() {
    svc="$1"
    base=$(python -c "import yaml; print list(yaml.safe_load(file('$ARVADOS_CONFIG'))['Clusters']['zzzzz']['Services']['$1']['InternalURLs'].keys())[0]")
    url="$base/_health/ping"
    if ! curl -Ss -H "Authorization: Bearer e687950a23c3a9bceec28c6223a06c79" "${url}" | tee -a /dev/stderr | grep '"OK"'; then
        echo "${url} failed"
        return 1
    fi
}

checkdiscoverydoc() {
    dd="https://${1}/discovery/v1/apis/arvados/v1/rest"
    if ! (set -o pipefail; curl -fsk "$dd" | grep -q ^{ ); then
        echo >&2 "ERROR: could not retrieve discovery doc from RailsAPI at $dd"
        tail -v $WORKSPACE/services/api/log/test.log
        return 1
    fi
    echo "${dd} ok"
}

start_services() {
    if [[ -n "$ARVADOS_TEST_API_HOST" ]]; then
        return 0
    fi
    . "$VENVDIR/bin/activate"
    echo 'Starting API, controller, keepproxy, keep-web, arv-git-httpd, ws, and nginx ssl proxy...'
    if [[ ! -d "$WORKSPACE/services/api/log" ]]; then
	mkdir -p "$WORKSPACE/services/api/log"
    fi
    # Remove empty api.pid file if it exists
    if [[ -f "$WORKSPACE/tmp/api.pid" && ! -s "$WORKSPACE/tmp/api.pid" ]]; then
	rm -f "$WORKSPACE/tmp/api.pid"
    fi
    all_services_stopped=
    fail=1

    cd "$WORKSPACE" \
        && eval $(python sdk/python/tests/run_test_server.py start --auth admin) \
        && export ARVADOS_TEST_API_HOST="$ARVADOS_API_HOST" \
        && export ARVADOS_TEST_API_INSTALLED="$$" \
        && checkpidfile api \
        && checkdiscoverydoc $ARVADOS_API_HOST \
        && eval $(python sdk/python/tests/run_test_server.py start_nginx) \
        && checkpidfile nginx \
        && python sdk/python/tests/run_test_server.py start_controller \
        && checkpidfile controller \
        && checkhealth Controller \
        && checkdiscoverydoc $ARVADOS_API_HOST \
        && python sdk/python/tests/run_test_server.py start_keep_proxy \
        && checkpidfile keepproxy \
        && python sdk/python/tests/run_test_server.py start_keep-web \
        && checkpidfile keep-web \
        && checkhealth WebDAV \
        && python sdk/python/tests/run_test_server.py start_arv-git-httpd \
        && checkpidfile arv-git-httpd \
        && checkhealth GitHTTP \
        && python sdk/python/tests/run_test_server.py start_ws \
        && checkpidfile ws \
        && export ARVADOS_TEST_PROXY_SERVICES=1 \
        && (env | egrep ^ARVADOS) \
        && fail=0
    deactivate
    if [[ $fail != 0 ]]; then
        unset ARVADOS_TEST_API_HOST
    fi
    return $fail
}

stop_services() {
    if [[ -n "$all_services_stopped" ]]; then
        return
    fi
    unset ARVADOS_TEST_API_HOST ARVADOS_TEST_PROXY_SERVICES
    . "$VENVDIR/bin/activate" || return
    cd "$WORKSPACE" \
        && python sdk/python/tests/run_test_server.py stop_nginx \
        && python sdk/python/tests/run_test_server.py stop_arv-git-httpd \
        && python sdk/python/tests/run_test_server.py stop_ws \
        && python sdk/python/tests/run_test_server.py stop_keep-web \
        && python sdk/python/tests/run_test_server.py stop_keep_proxy \
        && python sdk/python/tests/run_test_server.py stop_controller \
        && python sdk/python/tests/run_test_server.py stop \
        && all_services_stopped=1
    deactivate
    unset ARVADOS_CONFIG
}

interrupt() {
    failures+=("($(basename $0) interrupted)")
    exit_cleanly
}
trap interrupt INT

setup_ruby_environment() {
    if [[ -s "$HOME/.rvm/scripts/rvm" ]] ; then
        source "$HOME/.rvm/scripts/rvm"
        using_rvm=true
    elif [[ -s "/usr/local/rvm/scripts/rvm" ]] ; then
        source "/usr/local/rvm/scripts/rvm"
        using_rvm=true
    else
        using_rvm=false
    fi

    if [[ "$using_rvm" == true ]]; then
        # If rvm is in use, we can't just put separate "dependencies"
        # and "gems-under-test" paths to GEM_PATH: passenger resets
        # the environment to the "current gemset", which would lose
        # our GEM_PATH and prevent our test suites from running ruby
        # programs (for example, the Workbench test suite could not
        # boot an API server or run arv). Instead, we have to make an
        # rvm gemset and use it for everything.

        [[ `type rvm | head -n1` == "rvm is a function" ]] \
            || fatal 'rvm check'

        # Put rvm's favorite path back in first place (overriding
        # virtualenv, which just put itself there). Ignore rvm's
        # complaint about not being in first place already.
        rvm use @default 2>/dev/null

        # Create (if needed) and switch to an @arvados-tests-* gemset,
        # salting the gemset name so it doesn't interfere with
        # concurrent builds in other workspaces. Leave the choice of
        # ruby to the caller.
        gemset="arvados-tests-$(echo -n "${WORKSPACE}" | md5sum | head -c16)"
        rvm use "@${gemset}" --create \
            || fatal 'rvm gemset setup'

        rvm env
    else
        # When our "bundle install"s need to install new gems to
        # satisfy dependencies, we want them to go where "gem install
        # --user-install" would put them. (However, if the caller has
        # already set GEM_HOME, we assume that's where dependencies
        # should be installed, and we should leave it alone.)

        if [ -z "$GEM_HOME" ]; then
            user_gempath="$(gem env gempath)"
            export GEM_HOME="${user_gempath%%:*}"
        fi
        PATH="$(gem env gemdir)/bin:$PATH"

        # When we build and install our own gems, we install them in our
        # $GEMHOME tmpdir, and we want them to be at the front of GEM_PATH and
        # PATH so integration tests prefer them over other versions that
        # happen to be installed in $user_gempath, system dirs, etc.

        tmpdir_gem_home="$(env - PATH="$PATH" HOME="$GEMHOME" gem env gempath | cut -f1 -d:)"
        PATH="$tmpdir_gem_home/bin:$PATH"
        export GEM_PATH="$tmpdir_gem_home"

        echo "Will install dependencies to $(gem env gemdir)"
        echo "Will install arvados gems to $tmpdir_gem_home"
        echo "Gem search path is GEM_PATH=$GEM_PATH"
    fi
    bundle config || gem install bundler \
        || fatal 'install bundler'
}

with_test_gemset() {
    if [[ "$using_rvm" == true ]]; then
        "$@"
    else
        GEM_HOME="$tmpdir_gem_home" GEM_PATH="$tmpdir_gem_home" "$@"
    fi
}

gem_uninstall_if_exists() {
    if gem list "$1\$" | egrep '^\w'; then
        gem uninstall --force --all --executables "$1"
    fi
}

setup_virtualenv() {
    local venvdest="$1"; shift
    if ! [[ -e "$venvdest/bin/activate" ]] || ! [[ -e "$venvdest/bin/pip" ]]; then
        virtualenv --setuptools "$@" "$venvdest" || fatal "virtualenv $venvdest failed"
    elif [[ -n "$short" ]]; then
        return
    fi
    if [[ $("$venvdest/bin/python" --version 2>&1) =~ \ 3\.[012]\. ]]; then
        # pip 8.0.0 dropped support for python 3.2, e.g., debian wheezy
        "$venvdest/bin/pip" install --no-cache-dir 'setuptools>=18.5' 'pip>=7,<8'
    else
        "$venvdest/bin/pip" install --no-cache-dir 'setuptools>=18.5' 'pip>=7'
    fi
}

initialize() {
    sanity_checks

    echo "WORKSPACE=$WORKSPACE"

    # Clean up .pyc files that may exist in the workspace
    cd "$WORKSPACE"
    find -name '*.pyc' -delete

    if [[ -z "$temp" ]]; then
        temp="$(mktemp -d)"
    fi

    # Set up temporary install dirs (unless existing dirs were supplied)
    for tmpdir in VENVDIR VENV3DIR GOPATH GEMHOME PERLINSTALLBASE R_LIBS
    do
        if [[ -z "${!tmpdir}" ]]; then
            eval "$tmpdir"="$temp/$tmpdir"
        fi
        if ! [[ -d "${!tmpdir}" ]]; then
            mkdir "${!tmpdir}" || fatal "can't create ${!tmpdir} (does $temp exist?)"
        fi
    done

    rm -vf "${WORKSPACE}/tmp/*.log"

    export PERLINSTALLBASE
    export PERL5LIB="$PERLINSTALLBASE/lib/perl5${PERL5LIB:+:$PERL5LIB}"

    export R_LIBS

    export GOPATH
    # Make sure our compiled binaries under test override anything
    # else that might be in the environment.
    export PATH=$GOPATH/bin:$PATH

    # Jenkins config requires that glob tmp/*.log match something. Ensure
    # that happens even if we don't end up running services that set up
    # logging.
    mkdir -p "${WORKSPACE}/tmp/" || fatal "could not mkdir ${WORKSPACE}/tmp"
    touch "${WORKSPACE}/tmp/controller.log" || fatal "could not touch ${WORKSPACE}/tmp/controller.log"

    unset http_proxy https_proxy no_proxy

    # Note: this must be the last time we change PATH, otherwise rvm will
    # whine a lot.
    setup_ruby_environment

    echo "PATH is $PATH"
}

install_env() {
    (
        set -e
        mkdir -p "$GOPATH/src/git.curoverse.com"
        if [[ ! -h "$GOPATH/src/git.curoverse.com/arvados.git" ]]; then
            for d in \
                "$GOPATH/src/git.curoverse.com/arvados.git/tmp/GOPATH" \
                    "$GOPATH/src/git.curoverse.com/arvados.git/tmp" \
                    "$GOPATH/src/git.curoverse.com/arvados.git/arvados" \
                    "$GOPATH/src/git.curoverse.com/arvados.git"; do
                [[ -h "$d" ]] && rm "$d"
                [[ -d "$d" ]] && rmdir "$d"
            done
        fi
        ln -vsfT "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"
        go get -v github.com/kardianos/govendor
        cd "$GOPATH/src/git.curoverse.com/arvados.git"
        go get -v -d ...
        "$GOPATH/bin/govendor" sync
        which goimports >/dev/null || go get golang.org/x/tools/cmd/goimports
    ) || fatal "Go setup failed"

    setup_virtualenv "$VENVDIR" --python python2.7
    . "$VENVDIR/bin/activate"

    # Needed for run_test_server.py which is used by certain (non-Python) tests.
    pip install --no-cache-dir PyYAML future \
        || fatal "pip install PyYAML failed"

    # Preinstall libcloud if using a fork; otherwise nodemanager "pip
    # install" won't pick it up by default.
    if [[ -n "$LIBCLOUD_PIN_SRC" ]]; then
        pip freeze 2>/dev/null | egrep ^apache-libcloud==$LIBCLOUD_PIN \
            || pip install --pre --ignore-installed --no-cache-dir "$LIBCLOUD_PIN_SRC" >/dev/null \
            || fatal "pip install apache-libcloud failed"
    fi

    # Deactivate Python 2 virtualenv
    deactivate

    # If Python 3 is available, set up its virtualenv in $VENV3DIR.
    # Otherwise, skip dependent tests.
    PYTHON3=$(which python3)
    if [[ ${?} = 0 ]]; then
        setup_virtualenv "$VENV3DIR" --python python3
    else
        PYTHON3=
        cat >&2 <<EOF

Warning: python3 could not be found. Python 3 tests will be skipped.

EOF
    fi

    if ! which bundler >/dev/null
    then
        gem install --user-install bundler || fatal 'Could not install bundler'
    fi
}

retry() {
    remain="${repeat}"
    while :
    do
        if ${@}; then
            if [[ "$remain" -gt 1 ]]; then
                remain=$((${remain}-1))
                title "(repeating ${remain} more times)"
            else
                break
            fi
        elif [[ "$retry" == 1 ]]; then
            read -p 'Try again? [Y/n] ' x
            if [[ "$x" != "y" ]] && [[ "$x" != "" ]]
            then
                break
            fi
        else
            break
        fi
    done
}

do_test() {
    case "${1}" in
        apps/workbench_units | apps/workbench_functionals | apps/workbench_integration)
            suite=apps/workbench
            ;;
        services/nodemanager | services/nodemanager_integration)
            suite=services/nodemanager_suite
            ;;
        *)
            suite="${1}"
            ;;
    esac
    if [[ -n "${skip[$suite]}" || \
              -n "${skip[$1]}" || \
              (${#only[@]} -ne 0 && ${only[$suite]} -eq 0 && ${only[$1]} -eq 0) ]]; then
        return 0
    fi
    case "${1}" in
        services/api)
            stop_services
            check_arvados_config "$1"
            ;;
        gofmt | govendor | doc | lib/cli | lib/cloud/azure | lib/cloud/ec2 | lib/cloud/cloudtest | lib/cmd | lib/dispatchcloud/ssh_executor | lib/dispatchcloud/worker)
            check_arvados_config "$1"
            # don't care whether services are running
            ;;
        *)
            check_arvados_config "$1"
            if ! start_services; then
                checkexit 1 "$1 tests"
                title "test $1 -- failed to start services"
                return 1
            fi
            ;;
    esac
    retry do_test_once ${@}
}

go_ldflags() {
    version=${ARVADOS_VERSION:-$(git log -n1 --format=%H)-dev}
    echo "-X git.curoverse.com/arvados.git/lib/cmd.version=${version} -X main.version=${version}"
}

do_test_once() {
    unset result

    title "test $1"
    timer_reset

    result=
    if which deactivate >/dev/null; then deactivate; fi
    if ! . "$VENVDIR/bin/activate"
    then
        result=1
    elif [[ "$2" == "go" ]]
    then
        covername="coverage-$(echo "$1" | sed -e 's/\//_/g')"
        coverflags=("-covermode=count" "-coverprofile=$WORKSPACE/tmp/.$covername.tmp")
        # We do "go get -t" here to catch compilation errors
        # before trying "go test". Otherwise, coverage-reporting
        # mode makes Go show the wrong line numbers when reporting
        # compilation errors.
        go get -ldflags "$(go_ldflags)" -t "git.curoverse.com/arvados.git/$1" && \
            cd "$GOPATH/src/git.curoverse.com/arvados.git/$1" && \
            if [[ -n "${testargs[$1]}" ]]
        then
            # "go test -check.vv giturl" doesn't work, but this
            # does:
            go test ${short:+-short} ${testargs[$1]}
        else
            # The above form gets verbose even when testargs is
            # empty, so use this form in such cases:
            go test ${short:+-short} ${coverflags[@]} "git.curoverse.com/arvados.git/$1"
        fi
        result=${result:-$?}
        if [[ -f "$WORKSPACE/tmp/.$covername.tmp" ]]
        then
            go tool cover -html="$WORKSPACE/tmp/.$covername.tmp" -o "$WORKSPACE/tmp/$covername.html"
            rm "$WORKSPACE/tmp/.$covername.tmp"
        fi
        [[ $result = 0 ]] && gofmt -e -d *.go
    elif [[ "$2" == "pip" ]]
    then
        tries=0
        cd "$WORKSPACE/$1" && while :
        do
            tries=$((${tries}+1))
            # $3 can name a path directory for us to use, including trailing
            # slash; e.g., the bin/ subdirectory of a virtualenv.
            if [[ -e "${3}activate" ]]; then
                . "${3}activate"
            fi
            python setup.py ${short:+--short-tests-only} test ${testargs[$1]}
            result=$?
            if [[ ${tries} < 3 && ${result} == 137 ]]
            then
                printf '\n*****\n%s tests killed -- retrying\n*****\n\n' "$1"
                continue
            else
                break
            fi
        done
    elif [[ "$2" != "" ]]
    then
        "test_$2"
    else
        "test_$1"
    fi
    result=${result:-$?}
    checkexit $result "$1 tests"
    title "test $1 -- `timer`"
    return $result
}

check_arvados_config() {
    if [[ "$1" = "env" ]] ; then
	return
    fi
    if [[ -z "$ARVADOS_CONFIG" ]] ; then
	# Create config file.  The run_test_server script requires PyYAML,
	# so virtualenv needs to be active.  Downstream steps like
	# workbench install which require a valid config.yml.
	if [[ ! -s "$VENVDIR/bin/activate" ]] ; then
	    install_env
	fi
	. "$VENVDIR/bin/activate"
        cd "$WORKSPACE"
	eval $(python sdk/python/tests/run_test_server.py setup_config)
	deactivate
    fi
}

do_install() {
    if [[ -n "${skip[install]}" || ( -n "${only_install}" && "${only_install}" != "${1}" && "${only_install}" != "${2}" ) ]]; then
        return 0
    fi
    check_arvados_config "$1"
    retry do_install_once ${@}
}

do_install_once() {
    title "install $1"
    timer_reset

    result=
    if which deactivate >/dev/null; then deactivate; fi
    if [[ "$1" != "env" ]] && ! . "$VENVDIR/bin/activate"; then
        result=1
    elif [[ "$2" == "go" ]]
    then
        go get -ldflags "$(go_ldflags)" -t "git.curoverse.com/arvados.git/$1"
    elif [[ "$2" == "pip" ]]
    then
        # $3 can name a path directory for us to use, including trailing
        # slash; e.g., the bin/ subdirectory of a virtualenv.

        # Need to change to a different directory after creating
        # the source dist package to avoid a pip bug.
        # see https://arvados.org/issues/5766 for details.

        # Also need to install twice, because if it believes the package is
        # already installed, pip it won't install it.  So the first "pip
        # install" ensures that the dependencies are met, the second "pip
        # install" ensures that we've actually installed the local package
        # we just built.
        cd "$WORKSPACE/$1" \
            && "${3}python" setup.py sdist rotate --keep=1 --match .tar.gz \
            && cd "$WORKSPACE" \
            && "${3}pip" install --no-cache-dir --quiet "$WORKSPACE/$1/dist"/*.tar.gz \
            && "${3}pip" install --no-cache-dir --quiet --no-deps --ignore-installed "$WORKSPACE/$1/dist"/*.tar.gz
    elif [[ "$2" != "" ]]
    then
        "install_$2"
    else
        "install_$1"
    fi
    result=${result:-$?}
    checkexit $result "$1 install"
    title "install $1 -- `timer`"
    return $result
}

bundle_install_trylocal() {
    (
        set -e
        echo "(Running bundle install --local. 'could not find package' messages are OK.)"
        if ! bundle install --local --no-deployment; then
            echo "(Running bundle install again, without --local.)"
            bundle install --no-deployment
        fi
        bundle package --all
    )
}

install_doc() {
    cd "$WORKSPACE/doc" \
        && bundle_install_trylocal \
        && rm -rf .site
}

install_gem() {
    gemname=$1
    srcpath=$2
    with_test_gemset gem_uninstall_if_exists "$gemname" \
        && cd "$WORKSPACE/$srcpath" \
        && bundle_install_trylocal \
        && gem build "$gemname.gemspec" \
        && with_test_gemset gem install --no-document $(ls -t "$gemname"-*.gem|head -n1)
}

install_sdk/ruby() {
    install_gem arvados sdk/ruby
}

install_sdk/R() {
  if [[ "$NEED_SDK_R" = true ]]; then
    cd "$WORKSPACE/sdk/R" \
       && Rscript --vanilla install_deps.R
  fi
}

install_sdk/perl() {
    cd "$WORKSPACE/sdk/perl" \
        && perl Makefile.PL INSTALL_BASE="$PERLINSTALLBASE" \
        && make install INSTALLDIRS=perl
}

install_sdk/cli() {
    install_gem arvados-cli sdk/cli
}

install_services/login-sync() {
    install_gem arvados-login-sync services/login-sync
}

install_services/api() {
    stop_services
    cd "$WORKSPACE/services/api" \
        && RAILS_ENV=test bundle_install_trylocal

    rm -f config/environments/test.rb
    cp config/environments/test.rb.example config/environments/test.rb

    # Clear out any lingering postgresql connections to the test
    # database, so that we can drop it. This assumes the current user
    # is a postgresql superuser.
    cd "$WORKSPACE/services/api" \
        && test_database=$(python -c "import yaml; print yaml.safe_load(file('$ARVADOS_CONFIG'))['Clusters']['zzzzz']['PostgreSQL']['Connection']['dbname']") \
        && psql "$test_database" -c "SELECT pg_terminate_backend (pg_stat_activity.pid::int) FROM pg_stat_activity WHERE pg_stat_activity.datname = '$test_database';" 2>/dev/null

    mkdir -p "$WORKSPACE/services/api/tmp/pids"

    cert="$WORKSPACE/services/api/tmp/self-signed"
    if [[ ! -e "$cert.pem" || "$(date -r "$cert.pem" +%s)" -lt 1512659226 ]]; then
        (
            dir="$WORKSPACE/services/api/tmp"
            set -ex
            openssl req -newkey rsa:2048 -nodes -subj '/C=US/ST=State/L=City/CN=localhost' -out "$cert.csr" -keyout "$cert.key" </dev/null
            openssl x509 -req -in "$cert.csr" -signkey "$cert.key" -out "$cert.pem" -days 3650 -extfile <(printf 'subjectAltName=DNS:localhost,DNS:::1,DNS:0.0.0.0,DNS:127.0.0.1,IP:::1,IP:0.0.0.0,IP:127.0.0.1')
        ) || return 1
    fi

    cd "$WORKSPACE/services/api" \
        && rm -rf tmp/git \
        && mkdir -p tmp/git \
        && cd tmp/git \
        && tar xf ../../test/test.git.tar \
        && mkdir -p internal.git \
        && git --git-dir internal.git init \
            || return 1


    (cd "$WORKSPACE/services/api"
     export RAILS_ENV=test
     if bundle exec rails db:environment:set ; then
        bundle exec rake db:drop
     fi
     bundle exec rake db:setup \
	 && bundle exec rake db:fixtures:load
    )
}

declare -a pythonstuff
pythonstuff=(
    sdk/pam
    sdk/python
    sdk/python:py3
    sdk/cwl
    sdk/cwl:py3
    services/dockercleaner:py3
    services/fuse
    services/fuse:py3
    services/nodemanager
    tools/crunchstat-summary
    tools/crunchstat-summary:py3
)

declare -a gostuff
gostuff=($(cd "$WORKSPACE" && git grep -lw func | grep \\.go | sed -e 's/\/[^\/]*$//' | sort -u))

install_apps/workbench() {
    cd "$WORKSPACE/apps/workbench" \
        && mkdir -p tmp/cache \
        && RAILS_ENV=test bundle_install_trylocal \
        && RAILS_ENV=test RAILS_GROUPS=assets bundle exec rake npm:install
}

test_doc() {
    (
        set -e
        cd "$WORKSPACE/doc"
        ARVADOS_API_HOST=qr1hi.arvadosapi.com
        # Make sure python-epydoc is installed or the next line won't
        # do much good!
        PYTHONPATH=$WORKSPACE/sdk/python/ bundle exec rake linkchecker baseurl=file://$WORKSPACE/doc/.site/ arvados_workbench_host=https://workbench.$ARVADOS_API_HOST arvados_api_host=$ARVADOS_API_HOST
    )
}

test_gofmt() {
    cd "$WORKSPACE" || return 1
    dirs=$(ls -d */ | egrep -v 'vendor|tmp')
    [[ -z "$(gofmt -e -d $dirs | tee -a /dev/stderr)" ]]
}

test_govendor() {
    (
        set -e
        cd "$GOPATH/src/git.curoverse.com/arvados.git"
        # Remove cached source dirs in workdir. Otherwise, they will
        # not qualify as +missing or +external below, and we won't be
        # able to detect that they're missing from vendor/vendor.json.
        rm -rf vendor/*/
        go get -v -d ...
        "$GOPATH/bin/govendor" sync
        if [[ -n $("$GOPATH/bin/govendor" list +unused +missing +external | tee /dev/stderr) ]]; then
            echo >&2 "vendor/vendor.json has unused or missing dependencies -- try:

(export GOPATH=\"${GOPATH}\"; cd \$GOPATH/src/git.curoverse.com/arvados.git && \$GOPATH/bin/govendor add +missing +external && \$GOPATH/bin/govendor remove +unused)

"
            return 1
        fi
    )
}

test_services/api() {
    rm -f "$WORKSPACE/services/api/git-commit.version"
    cd "$WORKSPACE/services/api" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} bundle exec rake test TESTOPTS=\'-v -d\' ${testargs[services/api]}
}

test_sdk/ruby() {
    cd "$WORKSPACE/sdk/ruby" \
        && bundle exec rake test TESTOPTS=-v ${testargs[sdk/ruby]}
}

test_sdk/R() {
  if [[ "$NEED_SDK_R" = true ]]; then
    cd "$WORKSPACE/sdk/R" \
        && Rscript --vanilla run_test.R
  fi
}

test_sdk/cli() {
    cd "$WORKSPACE/sdk/cli" \
        && mkdir -p /tmp/keep \
        && KEEP_LOCAL_STORE=/tmp/keep bundle exec rake test TESTOPTS=-v ${testargs[sdk/cli]}
}

test_sdk/java-v2() {
    cd "$WORKSPACE/sdk/java-v2" && gradle test
}

test_services/login-sync() {
    cd "$WORKSPACE/services/login-sync" \
        && bundle exec rake test TESTOPTS=-v ${testargs[services/login-sync]}
}

test_services/nodemanager_integration() {
    cd "$WORKSPACE/services/nodemanager" \
        && tests/integration_test.py ${testargs[services/nodemanager_integration]}
}

test_apps/workbench_units() {
    local TASK="test:units"
    cd "$WORKSPACE/apps/workbench" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} bundle exec rake ${TASK} TESTOPTS=\'-v -d\' ${testargs[apps/workbench]} ${testargs[apps/workbench_units]}
}

test_apps/workbench_functionals() {
    local TASK="test:functionals"
    cd "$WORKSPACE/apps/workbench" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} bundle exec rake ${TASK} TESTOPTS=\'-v -d\' ${testargs[apps/workbench]} ${testargs[apps/workbench_functionals]}
}

test_apps/workbench_integration() {
    local TASK="test:integration"
    cd "$WORKSPACE/apps/workbench" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} bundle exec rake ${TASK} TESTOPTS=\'-v -d\' ${testargs[apps/workbench]} ${testargs[apps/workbench_integration]} 
}

test_apps/workbench_benchmark() {
    local TASK="test:benchmark"
    cd "$WORKSPACE/apps/workbench" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} bundle exec rake ${TASK} ${testargs[apps/workbench_benchmark]}
}

test_apps/workbench_profile() {
    local TASK="test:profile"
    cd "$WORKSPACE/apps/workbench" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} bundle exec rake ${TASK} ${testargs[apps/workbench_profile]}
}

install_deps() {
    # Install parts needed by test suites
    do_install env
    do_install cmd/arvados-server go
    do_install sdk/cli
    do_install sdk/perl
    do_install sdk/python pip
    do_install sdk/ruby
    do_install services/api
    do_install services/arv-git-httpd go
    do_install services/keepproxy go
    do_install services/keepstore go
    do_install services/keep-web go
    do_install services/ws go
}

install_all() {
    do_install env
    do_install doc
    do_install sdk/ruby
    do_install sdk/R
    do_install sdk/perl
    do_install sdk/cli
    do_install services/login-sync
    for p in "${pythonstuff[@]}"
    do
        dir=${p%:py3}
        if [[ ${dir} = ${p} ]]; then
            if [[ -z ${skip[python2]} ]]; then
                do_install ${dir} pip
            fi
        elif [[ -n ${PYTHON3} ]]; then
            if [[ -z ${skip[python3]} ]]; then
                do_install ${dir} pip "$VENV3DIR/bin/"
            fi
        fi
    done
    for g in "${gostuff[@]}"
    do
        do_install "$g" go
    done
    do_install services/api
    do_install apps/workbench
}

test_all() {
    stop_services
    do_test services/api

    # Shortcut for when we're only running apiserver tests. This saves a bit of time,
    # because we don't need to start up the api server for subsequent tests.
    if [ ! -z "$only" ] && [ "$only" == "services/api" ]; then
        rotate_logfile "$WORKSPACE/services/api/log/" "test.log"
        exit_cleanly
    fi

    do_test gofmt
    do_test govendor
    do_test doc
    do_test sdk/ruby
    do_test sdk/R
    do_test sdk/cli
    do_test services/login-sync
    do_test sdk/java-v2
    do_test services/nodemanager_integration
    for p in "${pythonstuff[@]}"
    do
        dir=${p%:py3}
        if [[ ${dir} = ${p} ]]; then
            if [[ -z ${skip[python2]} ]]; then
                do_test ${dir} pip
            fi
        elif [[ -n ${PYTHON3} ]]; then
            if [[ -z ${skip[python3]} ]]; then
                do_test ${dir} pip "$VENV3DIR/bin/"
            fi
        fi
    done

    for g in "${gostuff[@]}"
    do
        do_test "$g" go
    done
    do_test apps/workbench_units
    do_test apps/workbench_functionals
    do_test apps/workbench_integration
    do_test apps/workbench_benchmark
    do_test apps/workbench_profile
}

help_interactive() {
    echo "== Interactive commands:"
    echo "TARGET                 (short for 'test DIR')"
    echo "test TARGET"
    echo "test TARGET:py3        (test with python3)"
    echo "test TARGET -check.vv  (pass arguments to test)"
    echo "install TARGET"
    echo "install env            (go/python libs)"
    echo "install deps           (go/python libs + arvados components needed for integration tests)"
    echo "reset                  (...services used by integration tests)"
    echo "exit"
    echo "== Test targets:"
    echo "${!testfuncargs[@]}" | tr ' ' '\n' | sort | column
}

initialize

declare -A testfuncargs=()
for g in "${gostuff[@]}"; do
    testfuncargs[$g]="$g go"
done
for p in "${pythonstuff[@]}"; do
    dir=${p%:py3}
    testfuncargs[$dir]="$dir pip $VENVDIR/bin/"
    testfuncargs[$dir:py3]="$dir pip $VENV3DIR/bin/"
done

testfuncargs["sdk/cli"]="sdk/cli"
testfuncargs["sdk/R"]="sdk/R"
testfuncargs["sdk/java-v2"]="sdk/java-v2"
testfuncargs["apps/workbench_units"]="apps/workbench_units"
testfuncargs["apps/workbench_functionals"]="apps/workbench_functionals"
testfuncargs["apps/workbench_integration"]="apps/workbench_integration"
testfuncargs["apps/workbench_benchmark"]="apps/workbench_benchmark"
testfuncargs["apps/workbench_profile"]="apps/workbench_profile"

if [[ -z ${interactive} ]]; then
    install_all
    test_all
else
    skip=()
    only=()
    only_install=()
    if [[ -e "$VENVDIR/bin/activate" ]]; then stop_services; fi
    setnextcmd() {
        if [[ "$TERM" = dumb ]]; then
            # assume emacs, or something, is offering a history buffer
            # and pre-populating the command will only cause trouble
            nextcmd=
        elif [[ ! -e "$VENVDIR/bin/activate" ]]; then
            nextcmd="install deps"
        else
            nextcmd=""
        fi
    }
    echo
    help_interactive
    nextcmd="install deps"
    setnextcmd
    HISTFILE="$WORKSPACE/tmp/.history"
    history -r
    while read -p 'What next? ' -e -i "$nextcmd" nextcmd; do
        history -s "$nextcmd"
        history -w
        read verb target opts <<<"${nextcmd}"
        target="${target%/}"
        target="${target/\/:/:}"
        case "${verb}" in
            "exit" | "quit")
                exit_cleanly
                ;;
            "reset")
                stop_services
                ;;
            "test" | "install")
                case "$target" in
                    "")
                        help_interactive
                        ;;
                    all | deps)
                        ${verb}_${target}
                        ;;
                    *)
                        testargs["$target"]="${opts}"
                        tt="${testfuncargs[${target}]}"
                        tt="${tt:-$target}"
                        do_$verb $tt
                        ;;
                esac
                ;;
            "" | "help" | *)
                help_interactive
                ;;
        esac
        if [[ ${#successes[@]} -gt 0 || ${#failures[@]} -gt 0 ]]; then
            report_outcomes
            successes=()
            failures=()
        fi
        cd "$WORKSPACE"
        setnextcmd
    done
    echo
fi
exit_cleanly
