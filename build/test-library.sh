# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

COLUMNS=80
. `dirname "$(readlink -f "$0")"`/run-library.sh

declare -a failures

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

rotate_logfile() {
  # i.e.  rotate_logfile "$WORKSPACE/apps/workbench/log/" "test.log"
  # $BUILD_NUMBER is set by Jenkins if this script is being called as part of a Jenkins run
  if [[ -f "$1/$2" ]]; then
    THEDATE=`date +%Y%m%d%H%M%S`
    mv "$1/$2" "$1/$THEDATE-$BUILD_NUMBER-$2"
    gzip "$1/$THEDATE-$BUILD_NUMBER-$2"
  fi
}

exit_cleanly() {
    trap - INT
    #create-plot-data-from-log.sh $BUILD_NUMBER "$WORKSPACE/apps/workbench/log/test.log" "$WORKSPACE/apps/workbench/log/"
    rotate_logfile "$WORKSPACE/apps/workbench/log/" "test.log"
    stop_services
    rotate_logfile "$WORKSPACE/services/api/log/" "test.log"
    report_outcomes
    clear_temp
    exit ${#failures}
}

retry() {
    local remain="${repeat}"
    while :
    do
        if ${@}; then
            if [[ "$remain" -gt 1 ]]; then
                remain=$((${remain}-1))
                title "Repeating ${remain} more times"
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

declare -A did_install

should_install() {
    if [[ -n "${did_install[$1]}" ]] ; then
	return 1
    fi

    if [[ -n "${include_install[@]}" && -z "${include_install[$1]}" ]] ; then
	title "Skipping $1 install because not in include_install"
	did_install[$1]=1
	return 1
    fi

    if [[ -n "${exclude_install[$1]}" ]] ; then
	title "Skipping $1 install because of exclude_install"
	did_install[$1]=1
	return 1
    fi

    return 0
}

do_install() {
    if [[ -z "${did_install[$1]}" ]] ; then
	retry do_install_once ${@}
    fi
}

do_install_once() {
    local name="$1"
    title "Running $name install"
    timer_reset
    if [[ "$2" == "go" ]]
    then
	do_install gopath
        go get -t -v "git.curoverse.com/arvados.git/$name"
    elif [[ "$2" == "pip" ]]
    then
	if ! should_install "$1" ; then
	    return
	fi

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
        cd "$WORKSPACE/$name" \
            && "${3}python" setup.py sdist rotate --keep=1 --match .tar.gz \
            && cd "$WORKSPACE" \
            && "${3}pip" install --no-cache-dir --quiet "$WORKSPACE/$name/dist"/*.tar.gz \
            && "${3}pip" install --no-cache-dir --quiet --no-deps --ignore-installed "$WORKSPACE/$name/dist"/*.tar.gz
    else
	shift
        "install_$name" "$@"
    fi
    result=$?
    checkexit $result "$name install"
    did_install[$name]=1
    title "End of $name install (`timer`)"
    return $result
}

declare -A testargs

do_test_once() {
    local result

    if [[ "$2" == "go" ]]
    then
	do_install gopath
    fi

    title "Running $1 tests"
    timer_reset
    if [[ "$2" == "go" ]]
    then
        covername="coverage-$(echo "$1" | sed -e 's/\//_/g')"
        coverflags=("-covermode=count" "-coverprofile=$WORKSPACE/tmp/.$covername.tmp")
        # We do "go get -t" here to catch compilation errors
        # before trying "go test". Otherwise, coverage-reporting
        # mode makes Go show the wrong line numbers when reporting
        # compilation errors.
        go get -t "git.curoverse.com/arvados.git/$1" && \
            cd "$GOPATH/src/git.curoverse.com/arvados.git/$1" && \
            [[ -z "$(gofmt -e -d . | tee /dev/stderr)" ]] && \
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
    elif [[ "$2" == "pip" ]]
    then
        tries=0
        cd "$WORKSPACE/$1" && while :
        do
            tries=$((${tries}+1))
            # $3 can name a path directory for us to use, including trailing
            # slash; e.g., the bin/ subdirectory of a virtualenv.
            "${3}python" setup.py ${short:+--short-tests-only} test ${testargs[$1]}
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
    title "End of $1 tests (`timer`)"
    return $result
}

do_test() {
    retry do_test_once ${@}
}

mktmpdir() {
    if [[ -z "$temp" ]]; then
	temp="$(mktemp -d)"
    fi

    if ! [[ -d "$temp" ]] ; then
	mkdir -p "$temp"
    fi

    tmpdir="${1}"
    if [[ -z "${!tmpdir}" ]]; then
        eval "$tmpdir"="$temp/$tmpdir"
    fi
    if ! [[ -d "${!tmpdir}" ]]; then
        mkdir "${!tmpdir}" || fatal "can't create ${!tmpdir} (does $temp exist?)"
    fi
}

install_gopath() {
    mktmpdir GOPATH
    export GOPATH

    if ! should_install gopath ; then
	return
    fi

    mkdir -p "$GOPATH/src/git.curoverse.com"
    rmdir -v --parents --ignore-fail-on-non-empty "$GOPATH/src/git.curoverse.com/arvados.git/tmp/GOPATH"
    for d in \
	"$GOPATH/src/git.curoverse.com/arvados.git/arvados.git" \
	    "$GOPATH/src/git.curoverse.com/arvados.git"; do
	[[ -d "$d" ]] && rmdir "$d"
	[[ -h "$d" ]] && rm "$d"
    done
    ln -vsnfT "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git" \
	|| fatal "symlink failed"
    go get -v github.com/kardianos/govendor \
	|| fatal "govendor install failed"
    cd "$GOPATH/src/git.curoverse.com/arvados.git" \
	|| fatal
    # Remove cached source dirs in workdir. Otherwise, they won't qualify
    # as +missing or +external below, and we won't be able to detect that
    # they're missing from vendor/vendor.json.
    rm -rf vendor/*/
    go get -v -d ...
    "$GOPATH/bin/govendor" sync \
	|| fatal "govendor sync failed"
    [[ -z $("$GOPATH/bin/govendor" list +unused +missing +external | tee /dev/stderr) ]] \
	|| fatal "vendor/vendor.json has unused or missing dependencies -- try:
* govendor remove +unused
* govendor add +missing +external
"
}

install_virtualenv() {
    local venvdest="$1"; shift
    if ! [[ -e "$venvdest/bin/activate" ]] || ! [[ -e "$venvdest/bin/pip" ]]; then
        virtualenv --setuptools "$@" "$venvdest" || fatal "virtualenv $venvdest failed"
    fi
    if [[ $("$venvdest/bin/python" --version 2>&1) =~ \ 3\.[012]\. ]]; then
        # pip 8.0.0 dropped support for python 3.2, e.g., debian wheezy
        "$venvdest/bin/pip" install --no-cache-dir 'setuptools>=18.5' 'pip>=7,<8'
    else
        "$venvdest/bin/pip" install --no-cache-dir 'setuptools>=18.5' 'pip>=7'
    fi
    # ubuntu1404 can't seem to install mock via tests_require, but it can do this.
    "$venvdest/bin/pip" install --no-cache-dir 'mock>=1.0' 'pbr<1.7.0'
}


install_py2_virtualenv() {
    mktmpdir VENVDIR

    if ! should_install py2_virtualenv ; then
	return
    fi

    do_install virtualenv "$VENVDIR" --python python2.7
    . "$VENVDIR/bin/activate"

    # Needed for run_test_server.py which is used by certain (non-Python) tests.
    pip freeze 2>/dev/null | egrep ^PyYAML= \
	|| pip install --no-cache-dir PyYAML >/dev/null \
	|| fatal "pip install PyYAML failed"

    # Preinstall libcloud, because nodemanager "pip install"
    # won't pick it up by default.
    pip freeze 2>/dev/null | egrep ^apache-libcloud==$LIBCLOUD_PIN \
	|| pip install --pre --ignore-installed --no-cache-dir "apache-libcloud>=$LIBCLOUD_PIN" >/dev/null \
	|| fatal "pip install apache-libcloud failed"

    # We need an unreleased (as of 2017-08-17) llfuse bugfix, otherwise our fuse test suite deadlocks.
    pip freeze | grep -x llfuse==1.2.0 || (
	set -e
	yes | pip uninstall llfuse || true
	cython --version || fatal "no cython; try sudo apt-get install cython"
	cd "$temp"
	(cd python-llfuse 2>/dev/null || git clone https://github.com/curoverse/python-llfuse)
	cd python-llfuse
	git checkout 620722fd990ea642ddb8e7412676af482c090c0c
	git checkout setup.py
	sed -i -e "s:'1\\.2':'1.2.0':" setup.py
	python setup.py build_cython
	python setup.py install --force
    ) || fatal "llfuse fork failed"
    pip freeze | grep -x llfuse==1.2.0 || fatal "error: installed llfuse 1.2.0 but '$(pip freeze | grep llfuse)' ???"

    deactivate
}

install_py3_virtualenv() {
    mktmpdir VENVDIR3
    do_install virtualenv "$VENVDIR3" --python python3
}

install_apiserver() {
    if ! should_install apiserver ; then
	return
    fi

    cd "$WORKSPACE/services/api" \
        && RAILS_ENV=test bundle_install_trylocal

    rm -f config/environments/test.rb
    cp config/environments/test.rb.example config/environments/test.rb

    if [ -n "$CONFIGSRC" ]
    then
        for f in database.yml
        do
            cp "$CONFIGSRC/$f" config/ || fatal "$f"
        done
    fi

    # Clear out any lingering postgresql connections to the test
    # database, so that we can drop it. This assumes the current user
    # is a postgresql superuser.
    cd "$WORKSPACE/services/api" \
        && test_database=$(python -c "import yaml; print yaml.load(file('config/database.yml'))['test']['database']") \
        && psql "$test_database" -c "SELECT pg_terminate_backend (pg_stat_activity.procpid::int) FROM pg_stat_activity WHERE pg_stat_activity.datname = '$test_database';" 2>/dev/null

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

    cd "$WORKSPACE/services/api" \
        && RAILS_ENV=test bundle exec rake db:drop \
        && RAILS_ENV=test bundle exec rake db:setup \
        && RAILS_ENV=test bundle exec rake db:fixtures:load
}


start_services() {
    if [[ -n "${did_install[start_services]}" ]] ; then
       return
    fi

    do_install py2_virtualenv

    # Install the Python SDK early. Various other test suites (like
    # keepproxy) bring up run_test_server.py, which imports the arvados
    # module. We can't actually *test* the Python SDK yet though, because
    # its own test suite brings up some of those other programs (like
    # keepproxy).
    . "$VENVDIR/bin/activate"
    do_install sdk/python pip

    do_install apiserver
    do_install services/keepstore go
    do_install services/keepproxy go
    do_install services/keep-web go
    do_install services/ws go
    do_install services/arv-git-httpd go

    echo 'Starting API, keepproxy, keep-web, ws, arv-git-httpd, and nginx ssl proxy...'
    if [[ ! -d "$WORKSPACE/services/api/log" ]]; then
	mkdir -p "$WORKSPACE/services/api/log"
    fi
    # Remove empty api.pid file if it exists
    if [[ -f "$WORKSPACE/tmp/api.pid" && ! -s "$WORKSPACE/tmp/api.pid" ]]; then
	rm -f "$WORKSPACE/tmp/api.pid"
    fi
    cd "$WORKSPACE" \
        && eval $(python sdk/python/tests/run_test_server.py start --auth admin) \
        && export ARVADOS_TEST_API_HOST="$ARVADOS_API_HOST" \
        && export ARVADOS_TEST_API_INSTALLED="$$" \
        && python sdk/python/tests/run_test_server.py start_keep_proxy \
        && python sdk/python/tests/run_test_server.py start_keep-web \
        && python sdk/python/tests/run_test_server.py start_arv-git-httpd \
        && python sdk/python/tests/run_test_server.py start_ws \
        && python sdk/python/tests/run_test_server.py start_nginx \
        && (env | egrep ^ARVADOS)
    did_install[start_services]=1
}

stop_services() {
    if [[ -z "$ARVADOS_TEST_API_HOST" ]]; then
        return
    fi
    unset ARVADOS_TEST_API_HOST
    cd "$WORKSPACE" \
        && python sdk/python/tests/run_test_server.py stop_nginx \
        && python sdk/python/tests/run_test_server.py stop_arv-git-httpd \
        && python sdk/python/tests/run_test_server.py stop_ws \
        && python sdk/python/tests/run_test_server.py stop_keep-web \
        && python sdk/python/tests/run_test_server.py stop_keep_proxy \
        && python sdk/python/tests/run_test_server.py stop
    did_install[start_services]=""
}

install_run_test_server() {
    start_services
}
