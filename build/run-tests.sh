#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

COLUMNS=80
. `dirname "$(readlink -f "$0")"`/run-library.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Install and test Arvados components.

Exit non-zero if any tests fail.

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

Options:

--skip FOO     Do not test the FOO component.
--skip sanity  Skip initial dev environment sanity checks.
--skip install Do not run any install steps. Just run tests.
               You should provide GOPATH, GEMHOME, and VENVDIR options
               from a previous invocation if you use this option.
--only FOO     Do not test anything except the FOO component. If given
               more than once, all specified test suites are run.
--temp DIR     Install components and dependencies under DIR instead of
               making a new temporary directory. Implies --leave-temp.
--leave-temp   Do not remove GOPATH, virtualenv, and other temp dirs at exit.
               Instead, show the path to give as --temp to reuse them in
               subsequent invocations.
--repeat N     Repeat each install/test step until it succeeds N times.
--retry        Prompt to retry if an install or test suite fails.
--only-install Run specific install step. If given more than once,
               all but the last are ignored.
--short        Skip (or scale down) some slow tests.
--interactive  Set up, then prompt for test/install steps to perform.
services/api_test="TEST=test/functional/arvados/v1/collections_controller_test.rb"
               Restrict apiserver tests to the given file
sdk/python_test="tests/test_api.py::ArvadosApiTest"
               Restrict Python SDK tests to the given class
lib/dispatchcloud_test="-check.vv"
               Show all log messages, even when tests pass (also works
               with services/keepstore_test etc.)
ARVADOS_DEBUG=1
               Print more debug messages
ARVADOS_...=...
               Set other ARVADOS_* env vars (note ARVADOS_* vars are
               removed from the environment by this script when it
               starts, so the usual way of passing them will not work)

Assuming "--skip install" is not given, all components are installed
into \$GOPATH, \$VENDIR, and \$GEMHOME before running any tests. Many
test suites depend on other components being installed, and installing
everything tends to be quicker than debugging dependencies.

Environment variables:

WORKSPACE=path Arvados source tree to test.
CONFIGSRC=path Dir with config.yml file containing PostgreSQL section
               for use by tests.  As a special concession to the
               current CI server config, CONFIGSRC defaults to
               $HOME/arvados-api-server if that directory exists.

More information and background:

https://dev.arvados.org/projects/arvados/wiki/Running_tests
EOF

# First make sure to remove any ARVADOS_ variables from the calling
# environment that could interfere with the tests.
unset $(env | cut -d= -f1 | grep \^ARVADOS_)

# Reset other variables that could affect our [tests'] behavior by
# accident.
GITDIR=
GOPATH=
VENV3DIR=
PYTHONPATH=
GEMHOME=
R_LIBS=
export LANG=en_US.UTF-8

# setup_ruby_environment will set this to the path of the `bundle` executable
# it installs. This stub will cause commands to fail if they try to run before
# that.
BUNDLE=false

short=
only_install=
temp=
temp_preserve=

ignore_sigint=

clear_temp() {
    if [[ -z "$temp" ]]; then
        # we did not even get as far as making a temp dir
        :
    elif [[ -z "$temp_preserve" ]]; then
        # Go creates readonly dirs in the module cache, which cause
        # "rm -rf" to fail unless we chmod first.
        chmod -R u+w "$temp"
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
    [[ -z "$CONFIGSRC" ]] || [[ -s "$CONFIGSRC/config.yml" ]] \
        || fatal "CONFIGSRC is $CONFIGSRC but '$CONFIGSRC/config.yml' is empty or not found (see: $0 --help)"
    echo Checking dependencies:
    echo "locale: ${LANG}"
    [[ "$(locale charmap)" = "UTF-8" ]] \
        || fatal "Locale '${LANG}' is broken/missing. Try: echo ${LANG} | sudo tee -a /etc/locale.gen && sudo locale-gen"
    echo -n 'ruby: '
    ruby -v \
        || fatal "No ruby. Install >=2.7 from package or source"
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
    echo -n 'virtualenv: '
    python3 -m venv --help | grep -q '^usage: venv ' \
        && echo "venv module found" \
        || fatal "No virtualenv. Try: apt-get install python3-venv"
    echo -n 'Python3 pyconfig.h: '
    find /usr/include -path '*/python3*/pyconfig.h' | egrep --max-count=1 . \
        || fatal "No Python3 pyconfig.h. Try: apt-get install python3-dev"
    which netstat \
        || fatal "No netstat. Try: apt-get install net-tools"
    echo -n 'nginx: '
    PATH="$PATH:/sbin:/usr/sbin:/usr/local/sbin" nginx -v \
        || fatal "No nginx. Try: apt-get install nginx"
    echo -n 'npm: '
    npm --version \
        || fatal "No npm. Try: wget -O- https://nodejs.org/dist/v14.21.3/node-v14.21.3-linux-x64.tar.xz | sudo tar -C /usr/local -xJf - && sudo ln -s ../node-v14.21.3-linux-x64/bin/{node,npm} /usr/local/bin/"
    echo -n 'cadaver: '
    cadaver --version | grep -w cadaver \
          || fatal "No cadaver. Try: apt-get install cadaver"
    echo -n "jq: "
    jq --version ||
        fatal "No jq. Try: apt-get install jq"
    echo -n 'libcurl curl.h: '
    find /usr/include -path '*/curl/curl.h' | egrep --max-count=1 . \
        || fatal "No libcurl curl.h. Try: apt-get install libcurl4-gnutls-dev"
    echo -n 'libpq libpq-fe.h: '
    find /usr/include -path '*/postgresql/libpq-fe.h' | egrep --max-count=1 . \
        || fatal "No libpq libpq-fe.h. Try: apt-get install libpq-dev"
    echo -n 'libpam pam_appl.h: '
    find /usr/include -path '*/security/pam_appl.h' | egrep --max-count=1 . \
        || fatal "No libpam pam_appl.h. Try: apt-get install libpam0g-dev"
    echo -n 'postgresql: '
    psql --version || fatal "No postgresql. Try: apt-get install postgresql postgresql-client-common"
    echo -n 'xvfb: '
    which Xvfb || fatal "No xvfb. Try: apt-get install xvfb"
    echo -n 'singularity: '
    singularity --version || fatal "No singularity. Try: arvados-server install"
    echo -n 'docker client: '
    docker --version || echo "No docker client. Try: arvados-server install"
    echo -n 'docker server: '
    docker info --format='{{.ServerVersion}}' || echo "No docker server. Try: arvados-server install"

    if [[ "$NEED_SDK_R" = true ]]; then
      # R SDK stuff
      echo -n 'R: '
      which Rscript || fatal "No Rscript. Try: apt-get install r-base"
      echo -n 'testthat: '
      Rscript -e "library('testthat')" || fatal "No testthat. Try: apt-get install r-cran-testthat"
      # needed for roxygen2, needed for devtools, needed for R sdk
      pkg-config --exists libxml-2.0 || fatal "No libxml2. Try: apt-get install libxml2-dev"
    fi
    echo 'procs with /dev/fuse open:'
    find /proc/*/fd -lname /dev/fuse 2>/dev/null | cut -d/ -f3 | xargs --no-run-if-empty ps -lywww
    echo 'grep fuse /proc/self/mountinfo:'
    grep fuse /proc/self/mountinfo
}

rotate_logfile() {
  # i.e.  rotate_logfile "$WORKSPACE/services/api/log/" "test.log"
  # $BUILD_NUMBER is set by Jenkins if this script is being called as part of a Jenkins run
  if [[ -f "$1/$2" ]]; then
    THEDATE=`date +%Y%m%d%H%M%S`
    mv "$1/$2" "$1/$THEDATE-$BUILD_NUMBER-$2"
    gzip "$1/$THEDATE-$BUILD_NUMBER-$2"
  fi
}

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
    base="$(yq -r "(.Clusters.zzzzz.Services.$svc.InternalURLs | keys)[0]" "$ARVADOS_CONFIG")"
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
        tail -v $WORKSPACE/tmp/railsapi.log
        return 1
    fi
    echo "${dd} ok"
}

start_services() {
    if [[ -n "$ARVADOS_TEST_API_HOST" ]]; then
        return 0
    fi
    echo 'Starting API, controller, keepproxy, keep-web, ws, and nginx ssl proxy...'
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
        && eval $(python3 sdk/python/tests/run_test_server.py start --auth admin) \
        && export ARVADOS_TEST_API_HOST="$ARVADOS_API_HOST" \
        && export ARVADOS_TEST_API_INSTALLED="$$" \
        && checkpidfile api \
        && checkdiscoverydoc $ARVADOS_API_HOST \
        && eval $(python3 sdk/python/tests/run_test_server.py start_nginx) \
        && checkpidfile nginx \
        && python3 sdk/python/tests/run_test_server.py start_controller \
        && checkpidfile controller \
        && checkhealth Controller \
        && checkdiscoverydoc $ARVADOS_API_HOST \
        && python3 sdk/python/tests/run_test_server.py start_keep_proxy \
        && checkpidfile keepproxy \
        && python3 sdk/python/tests/run_test_server.py start_keep-web \
        && checkpidfile keep-web \
        && checkhealth WebDAV \
        && python3 sdk/python/tests/run_test_server.py start_ws \
        && checkpidfile ws \
        && export ARVADOS_TEST_PROXY_SERVICES=1 \
        && (env | egrep ^ARVADOS) \
        && fail=0
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
    cd "$WORKSPACE" \
        && python3 sdk/python/tests/run_test_server.py stop_nginx \
        && python3 sdk/python/tests/run_test_server.py stop_ws \
        && python3 sdk/python/tests/run_test_server.py stop_keep-web \
        && python3 sdk/python/tests/run_test_server.py stop_keep_proxy \
        && python3 sdk/python/tests/run_test_server.py stop_controller \
        && python3 sdk/python/tests/run_test_server.py stop \
        && all_services_stopped=1
    unset ARVADOS_CONFIG
}

interrupt() {
    if [[ -n "$ignore_sigint" ]]; then
        echo >&2 "ignored SIGINT"
        return
    fi
    failures+=("($(basename $0) interrupted)")
    exit_cleanly
}
trap interrupt INT

setup_ruby_environment() {
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
    export GEM_PATH="$tmpdir_gem_home:$(gem env gempath)"

    echo "Will install dependencies to $(gem env gemdir)"
    echo "Will install bundler and arvados gems to $tmpdir_gem_home"
    echo "Gem search path is GEM_PATH=$GEM_PATH"
    gem install --user --no-document --conservative --version '~> 2.4.0' bundler \
        || fatal 'install bundler'
    BUNDLE="$(gem contents --version '~> 2.4.0' bundler | grep -E '/(bin|exe)/bundle$' | tail -n1)"
    if [[ ! -x "$BUNDLE" ]]; then
        BUNDLE=false
        fatal "could not find 'bundle' executable after installation"
    fi
}

with_test_gemset() {
    GEM_HOME="$tmpdir_gem_home" GEM_PATH="$tmpdir_gem_home" "$@"
}

setup_virtualenv() {
    if [[ -z "${VENV3DIR:-}" ]]; then
        fatal "setup_virtualenv called before \$VENV3DIR was set"
    elif ! [[ -e "$VENV3DIR/bin/activate" ]]; then
        python3 -m venv "$VENV3DIR" || fatal "virtualenv creation failed"
        # Configure pip options we always want to use.
        "$VENV3DIR/bin/pip" config --quiet --site set global.disable-pip-version-check true
        "$VENV3DIR/bin/pip" config --quiet --site set global.no-input true
        "$VENV3DIR/bin/pip" config --quiet --site set global.no-python-version-warning true
        "$VENV3DIR/bin/pip" config --quiet --site set install.progress-bar off
        # If we didn't have a virtualenv before, we couldn't have started any
        # services. Set the flag used by stop_services to indicate that.
        all_services_stopped=1
    fi
    . "$VENV3DIR/bin/activate" || fatal "virtualenv activation failed"
    # We must have these in place *before* we install the PySDK below.
    pip install -r "$WORKSPACE/build/requirements.tests.txt" ||
        fatal "failed to install Python requirements in virtualenv"
    # run-tests.sh uses run_test_server.py from the Python SDK.
    do_install_once sdk/python pip || fatal "failed to install PySDK in virtualenv"
}

initialize() {
    # If dependencies like ruby, go, etc. are installed in
    # /var/lib/arvados -- presumably by "arvados-server install" --
    # then we want to use those versions, instead of whatever happens
    # to be installed in /usr.
    PATH="/var/lib/arvados/bin:${PATH}"
    sanity_checks

    echo "WORKSPACE=$WORKSPACE"
    cd "$WORKSPACE"

    if [[ -z "$temp" ]]; then
        temp="$(mktemp -d)"
    fi

    # Set up temporary install dirs (unless existing dirs were supplied)
    for tmpdir in VENV3DIR GOPATH GEMHOME R_LIBS
    do
        if [[ -z "${!tmpdir}" ]]; then
            eval "$tmpdir"="$temp/$tmpdir"
        fi
        if ! [[ -d "${!tmpdir}" ]]; then
            mkdir "${!tmpdir}" || fatal "can't create ${!tmpdir} (does $temp exist?)"
        fi
    done

    rm -vf "${WORKSPACE}/tmp/*.log"

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

    setup_ruby_environment
    setup_virtualenv

    echo "PATH is $PATH"
}

install_env() {
    go mod download || fatal "Go deps failed"
    which goimports >/dev/null || go install golang.org/x/tools/cmd/goimports@latest || fatal "Go setup failed"
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
        services/workbench2_units | services/workbench2_integration)
            suite=services/workbench2
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
        gofmt \
            | arvados_version.py \
            | cmd/arvados-package \
            | doc \
            | lib/boot \
            | lib/cli \
            | lib/cloud/azure \
            | lib/cloud/cloudtest \
            | lib/cloud/ec2 \
            | lib/cmd \
            | lib/dispatchcloud/sshexecutor \
            | lib/dispatchcloud/worker \
            | lib/install \
            | services/workbench2_integration \
            | services/workbench2_units \
            )
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
    echo "-X git.arvados.org/arvados.git/lib/cmd.version=${version} -X main.version=${version} -s -w"
}

do_test_once() {
    unset result

    if [[ "$2" == pip ]]; then
        # We need to install the module before testing to ensure all the
        # dependencies are satisfied. We need to do this before we start
        # the test header+timer.
        do_install_once "$1" "$2" || return
    fi

    local -a targs=()
    case "$1" in
        sdk/cwl )
            # The CWL conformance/integration tests each take ~30
            # minutes. Before July 2025 they were outside the standard test
            # suite, so we deselect them by default for consistency.
            targs+=(-m "not integration")
            ;;
    esac
    # Append the user's arguments to targs, respecting quoted strings.
    eval "targs+=(${testargs[$1]})"

    title "test $1"
    timer_reset

    result=
    if [[ "$2" == "go" ]]
    then
        covername="coverage-$(echo "$1" | sed -e 's/\//_/g')"
        coverflags=("-covermode=count" "-coverprofile=$WORKSPACE/tmp/.$covername.tmp")
        testflags=()
        if [[ "$1" == "cmd/arvados-package" ]]; then
            testflags+=("-timeout" "20m")
        fi
        # We do "go install" here to catch compilation errors
        # before trying "go test". Otherwise, coverage-reporting
        # mode makes Go show the wrong line numbers when reporting
        # compilation errors.
        go install -ldflags "$(go_ldflags)" "$WORKSPACE/$1" && \
            cd "$WORKSPACE/$1" && \
            if [[ "${#targs}" -gt 0 ]]
        then
            # "go test -check.vv giturl" doesn't work, but this
            # does:
            go test ${short:+-short} ${testflags[@]} "${targs[@]}"
        else
            # The above form gets verbose even when testargs is
            # empty, so use this form in such cases:
            go test ${short:+-short} ${testflags[@]} ${coverflags[@]} "git.arvados.org/arvados.git/$1"
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
        while :
        do
            tries=$((${tries}+1))
            env -C "$WORKSPACE/$1" python3 -m pytest "${targs[@]}"
            result=$?
            # pytest uses exit code 2 to mean "test collection failed."
            # See discussion in FUSE's IntegrationTest and MountTestBase.
            if [[ ${tries} < 3 && ${result} == 2 ]]
            then
                printf '\n*****\n%s tests exited with code 2 -- retrying\n*****\n\n' "$1"
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
        cd "$WORKSPACE"
        eval $(python3 sdk/python/tests/run_test_server.py setup_config)
    fi
    # Set all PostgreSQL connection variables, and write a .pgpass, to connect
    # to the test database, so test scripts can write `psql` commands with no
    # additional configuration.
    export PGPASSFILE="$WORKSPACE/tmp/.pgpass"
    export PGDATABASE="$(yq -r .Clusters.zzzzz.PostgreSQL.Connection.dbname "$ARVADOS_CONFIG")"
    export PGHOST="$(yq -r .Clusters.zzzzz.PostgreSQL.Connection.host "$ARVADOS_CONFIG")"
    export PGPORT="$(yq -r .Clusters.zzzzz.PostgreSQL.Connection.port "$ARVADOS_CONFIG")"
    export PGUSER="$(yq -r .Clusters.zzzzz.PostgreSQL.Connection.user "$ARVADOS_CONFIG")"
    local pgpassword="$(yq -r .Clusters.zzzzz.PostgreSQL.Connection.password "$ARVADOS_CONFIG")"
    echo "$PGHOST:$PGPORT:$PGDATABASE:$PGUSER:$pgpassword" >"$PGPASSFILE"
    chmod 0600 "$PGPASSFILE"
}

do_install() {
    if [[ -n ${skip["install_$1"]} || -n "${skip[install]}" || ( -n "${only_install}" && "${only_install}" != "${1}" && "${only_install}" != "${2}" ) ]]; then
        return 0
    fi
    check_arvados_config "$1"
    retry do_install_once ${@}
}

do_install_once() {
    title "install $1"
    timer_reset

    result=
    if [[ "$2" == "go" ]]
    then
        go install -ldflags "$(go_ldflags)" "$WORKSPACE/$1"
    elif [[ "$2" == "pip" ]]
    then
        # Generate _version.py before installing.
        python3 "$WORKSPACE/$1/arvados_version.py" >/dev/null &&
            pip install "$WORKSPACE/$1"
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
        if ! "$BUNDLE" install --local --no-deployment; then
            echo "(Running bundle install again, without --local.)"
            "$BUNDLE" install --no-deployment
        fi
        "$BUNDLE" package
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
    cd "$WORKSPACE/$srcpath" \
        && bundle_install_trylocal \
        && gem build "$gemname.gemspec" \
        && with_test_gemset gem install --no-document $(ls -t "$gemname"-*.gem|head -n1)
}

install_sdk/ruby() {
    install_gem arvados sdk/ruby
}

install_sdk/ruby-google-api-client() {
    install_gem arvados-google-api-client sdk/ruby-google-api-client
}

install_contrib/R-sdk() {
  if [[ "$NEED_SDK_R" = true ]]; then
    env -C "$WORKSPACE/contrib/R-sdk" Rscript --vanilla install_deps.R
  fi
}

install_sdk/cli() {
    install_gem arvados-cli sdk/cli
}

install_services/login-sync() {
    install_gem arvados-login-sync services/login-sync
}

install_services/api() {
    stop_services
    check_arvados_config "services/api"
    cd "$WORKSPACE/services/api" \
        && RAILS_ENV=test bundle_install_trylocal \
            || return 1

    rm -f config/environments/test.rb
    cp config/environments/test.rb.example config/environments/test.rb

    # Clear out any lingering postgresql connections to the test
    # database, so that we can drop it. This assumes the current user
    # is a postgresql superuser.
    psql -c "SELECT pg_terminate_backend (pg_stat_activity.pid::int) FROM pg_stat_activity WHERE pg_stat_activity.datname = '$PGDATABASE';" 2>/dev/null

    mkdir -p "$WORKSPACE/services/api/tmp/pids"

    cert="$WORKSPACE/services/api/tmp/self-signed"
    if [[ ! -e "$cert.pem" || "$(date -r "$cert.pem" +%s)" -lt 1512659226 ]]; then
        (
            dir="$WORKSPACE/services/api/tmp"
            set -e
            openssl req -newkey rsa:2048 -nodes -subj '/C=US/ST=State/L=City/CN=localhost' -out "$cert.csr" -keyout "$cert.key" </dev/null
            openssl x509 -req -in "$cert.csr" -signkey "$cert.key" -out "$cert.pem" -days 3650 -extfile <(printf 'subjectAltName=DNS:localhost,DNS:::1,DNS:0.0.0.0,DNS:127.0.0.1,IP:::1,IP:0.0.0.0,IP:127.0.0.1')
        ) || return 1
    fi

    (
        set -ex
        cd "$WORKSPACE/services/api"
        export RAILS_ENV=test
        if bin/rails db:environment:set ; then
            bin/rake db:drop
        fi
        bin/rake db:setup
        bin/rake db:fixtures:load
    ) || return 1
}

install_services/workbench2() {
    cd "$WORKSPACE/services/workbench2" \
        && make yarn-install ARVADOS_DIRECTORY="${WORKSPACE}"
}

do_migrate() {
    timer_reset
    local task="db:migrate"
    case "$1" in
        "")
            ;;
        rollback)
            task="db:rollback"
            shift
            ;;
        *)
            task="db:migrate:$1"
            shift
            ;;
    esac
    check_arvados_config services/api
    (
        set -x
        env -C "$WORKSPACE/services/api" RAILS_ENV=test \
            "$BUNDLE" exec rake $task ${@}
    )
    checkexit "$?" "services/api $task"
}

migrate_down_services/api() {
    echo "running db:migrate:down"
    env -C "$WORKSPACE/services/api" RAILS_ENV=test \
        "$BUNDLE" exec rake db:migrate:down ${testargs[services/api]}
    checkexit "$?" "services/api db:migrate:down"
}

test_doc() {
    local arvados_api_host=pirca.arvadosapi.com && \
        env -C "$WORKSPACE/doc" \
        "$BUNDLE" exec rake linkchecker \
        arvados_api_host="$arvados_api_host" \
        arvados_workbench_host="https://workbench.$arvados_api_host" \
        baseurl="file://$WORKSPACE/doc/.site/" \
        ${testargs[doc]}
}

test_gofmt() {
    cd "$WORKSPACE" || return 1
    dirs=$(ls -d */ | egrep -v 'vendor|tmp')
    [[ -z "$(gofmt -e -d $dirs | tee -a /dev/stderr)" ]]
    go vet -composites=false ./...
}

test_arvados_version.py() {
    local orig_fn=""
    local fail_count=0
    while read -d "" fn; do
        if [[ -z "$orig_fn" ]]; then
            orig_fn="$fn"
        elif ! cmp "$orig_fn" "$fn"; then
            fail_count=$(( $fail_count + 1 ))
            printf "FAIL: %s and %s are not identical\n" "$orig_fn" "$fn"
        fi
    done < <(git -C "$WORKSPACE" ls-files -z | grep -z '/arvados_version\.py$')
    case "$orig_fn" in
        "") return 66 ;;  # EX_NOINPUT
        *) return "$fail_count" ;;
    esac
}

test_services/api() {
    rm -f "$WORKSPACE/services/api/git-commit.version"
    cd "$WORKSPACE/services/api" \
        && eval env RAILS_ENV=test ${short:+RAILS_TEST_SHORT=1} "$BUNDLE" exec rake test TESTOPTS=\'-v -d\' ${testargs[services/api]}
}

test_sdk/ruby() {
    cd "$WORKSPACE/sdk/ruby" \
        && "$BUNDLE" exec rake test TESTOPTS=-v ${testargs[sdk/ruby]}
}

test_sdk/ruby-google-api-client() {
    echo "*** note \`test sdk/ruby-google-api-client\` does not actually run any tests, see https://dev.arvados.org/issues/20993 ***"
    true
}

test_contrib/R-sdk() {
  if [[ "$NEED_SDK_R" = true ]]; then
    env -C "$WORKSPACE/contrib/R-sdk" make test
  fi
}

test_sdk/cli() {
    cd "$WORKSPACE/sdk/cli" \
        && mkdir -p /tmp/keep \
        && KEEP_LOCAL_STORE=/tmp/keep "$BUNDLE" exec rake test TESTOPTS=-v ${testargs[sdk/cli]}
}

test_contrib/java-sdk-v2() {
    env -C "$WORKSPACE/contrib/java-sdk-v2" gradle test ${testargs[contrib/java-sdk-v2]}
}

test_services/login-sync() {
    cd "$WORKSPACE/services/login-sync" \
        && "$BUNDLE" exec rake test TESTOPTS=-v ${testargs[services/login-sync]}
}

test_services/workbench2_units() {
    cd "$WORKSPACE/services/workbench2" && make unit-tests ARVADOS_DIRECTORY="${WORKSPACE}" WORKSPACE="$(pwd)" ${testargs[services/workbench2]}
}

test_services/workbench2_integration() {
    INTERACTIVE=
    FAIL_FAST_ENABLED=false
    if [[ -n ${interactive} ]] && [[ -n ${DISPLAY} ]]; then
	INTERACTIVE=-i
	FAIL_FAST_ENABLED=true
    fi
    cd "$WORKSPACE/services/workbench2" && make integration-tests ARVADOS_DIRECTORY="${WORKSPACE}" \
						WORKSPACE="$(pwd)" \
						INTERACTIVE=$INTERACTIVE \
						CYPRESS_FAIL_FAST_ENABLED=$FAIL_FAST_ENABLED \
						${testargs[services/workbench2]}
}

install_deps() {
    # Install parts needed by test suites
    do_install env
    # Many other components rely on PySDK's run_test_server.py, which relies on
    # the SDK itself, so install that first.
    do_install sdk/python pip
    # lib/controller integration tests depend on arv-mount to run containers.
    do_install services/fuse pip
    # sdk/cwl depends on crunchstat-summary.
    do_install tools/crunchstat-summary pip
    do_install cmd/arvados-server go
    do_install sdk/ruby-google-api-client
    do_install sdk/ruby
    do_install sdk/cli
    do_install services/api
    do_install services/keepproxy go
    do_install services/keep-web go
}

install_all() {
    do_install env
    do_install doc
    do_install sdk/ruby-google-api-client
    do_install sdk/ruby
    do_install contrib/R-sdk
    do_install sdk/cli
    do_install services/login-sync
    local pkg_dir
    if [[ -z ${skip[python3]} ]]; then
        for pkg_dir in "${pythonstuff[@]}"
        do
            do_install "$pkg_dir" pip
        done
    fi
    for pkg_dir in "${gostuff[@]}"
    do
        do_install "$pkg_dir" go
    done
    do_install services/api
    do_install services/workbench2
}

test_all() {
    stop_services
    do_test services/api
    do_test gofmt
    do_test arvados_version.py
    do_test doc
    do_test sdk/ruby-google-api-client
    do_test sdk/ruby
    do_test contrib/R-sdk
    do_test sdk/cli
    do_test services/login-sync
    do_test contrib/java-sdk-v2
    local pkg_dir
    if [[ -z ${skip[python3]} ]]; then
        for pkg_dir in "${pythonstuff[@]}"
        do
            do_test "$pkg_dir" pip
        done
    fi
    for pkg_dir in "${gostuff[@]}"
    do
        do_test "$pkg_dir" go
    done
    do_test services/workbench2_units
    do_test services/workbench2_integration
}

test_go() {
    do_test gofmt
    for g in "${gostuff[@]}"
    do
        do_test "$g" go
    done
}

help_interactive() {
    echo "== Interactive commands:"
    echo "TARGET                   (short for 'test DIR')"
    echo "test TARGET"
    echo "10 test TARGET           (run test 10 times)"
    echo "test TARGET -check.vv    (pass arguments to test)"
    echo "install TARGET"
    echo "install env              (go/python libs)"
    echo "install deps             (go/python libs + arvados components needed for integration tests)"
    echo "migrate                  (run outstanding migrations)"
    echo "migrate rollback         (revert most recent migration)"
    echo "migrate <dir> VERSION=n  (revert and/or run a single migration; <dir> is up|down|redo)"
    echo "reset                    (...services used by integration tests)"
    echo "exit"
    echo "== Test targets:"
    printf "%s\n" "${!testfuncargs[@]}" | sort | column
}

declare -a failures
declare -A skip
declare -A only
declare -A testargs

declare -a pythonstuff
pythonstuff=(
    # The ordering of sdk/python, tools/crunchstat-summary, and
    # sdk/cwl here is significant. See
    # https://dev.arvados.org/issues/19744#note-26
    sdk/python
    tools/crunchstat-summary
    sdk/cwl
    services/dockercleaner
    services/fuse
    tools/cluster-activity
)

declare -a gostuff
if [[ -n "$WORKSPACE" ]]; then
    readarray -d "" -t gostuff < <(
        git -C "$WORKSPACE" ls-files -z |
            grep -z '\.go$' |
            xargs -0r dirname -z |
            sort -zu
    )
fi

declare -A testfuncargs=()
for testfuncname in $(declare -F | awk '
($3 ~ /^test_/ && $3 !~ /_package_presence$/) {
  print substr($3, 6);
}
'); do
    testfuncargs[$testfuncname]="$testfuncname"
done
for g in "${gostuff[@]}"; do
    testfuncargs[$g]="$g go"
done
for p in "${pythonstuff[@]}"; do
    testfuncargs[$p]="$p pip"
done

while [[ -n "$1" ]]
do
    arg="$1"; shift
    case "$arg" in
        --help)
            exec 1>&2
            echo "$helpmessage"
            if [[ ${#gostuff} -gt 0 ]]; then
                printf "\nAvailable targets:\n\n"
                printf "%s\n" "${!testfuncargs[@]}" | sort | column
            fi
            exit 1
            ;;
        --skip)
            skip["${1%:py3}"]=1; shift
            ;;
        --only)
            only["${1%:py3}"]=1; skip["${1%:py3}"]=""; shift
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
            testargs["${suite%:py3}"]="$args"
            ;;
        ARVADOS_*=*)
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
   [[ -z "${only['contrib/R-sdk']}" && -z "${only['doc']}" ]]; then
  NEED_SDK_R=false
fi

if [[ ${skip["contrib/R-sdk"]} == 1 && ${skip["doc"]} == 1 ]]; then
  NEED_SDK_R=false
fi

if [[ $NEED_SDK_R == false ]]; then
        echo "R SDK not needed, it will not be installed."
fi

initialize
if [[ -z ${interactive} ]]; then
    install_all
    test_all
else
    skip=()
    only=()
    only_install=""
    stop_services
    setnextcmd() {
        if [[ "$TERM" = dumb ]]; then
            # assume emacs, or something, is offering a history buffer
            # and pre-populating the command will only cause trouble
            nextcmd=
        elif [[ ! -e "$GOPATH/bin/arvados-server" ]]; then
            nextcmd="install deps"
        else
            nextcmd=""
        fi
    }
    echo
    help_interactive
    setnextcmd
    HISTFILE="$WORKSPACE/tmp/.history"
    history -r
    ignore_sigint=1
    while read -p 'What next? ' -e -i "$nextcmd" nextcmd; do
        history -s "$nextcmd"
        history -w
        count=1
        if [[ "${nextcmd}" =~ ^[0-9] ]]; then
          read count nextcmd <<<"${nextcmd}"
        fi
        read verb target opts <<<"${nextcmd}"
        target="${target%/}"
        target="${target/\/:/:}"
        # Remove old Python version suffix for backwards compatibility
        target="${target%:py3}"
        case "${verb}" in
            "exit" | "quit")
                exit_cleanly
                ;;
            "reset")
                stop_services
                ;;
            "migrate")
                do_migrate ${target} ${opts}
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
                        while [ $count -gt 0 ]; do
                          do_$verb ${testfuncargs[${target}]}
                          let "count=count-1"
                        done
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
