#!/bin/bash

# Install and test Arvados components.
#
# Exit non-zero if any tests fail.
#
# Arguments:
# --skip FOO     Do not test the FOO component.
# --only FOO     Do not test anything except the FOO component.
# WORKSPACE=path Arvados source tree to test.
# CONFIGSRC=path Dir with api server config files to copy into source tree.
# envvar=value   Set $envvar to value
#
# Regardless of which components are tested, install all components in
# the usual sequence. (Many test suites depend on other components
# being installed.)
#
# To run a specific Ruby test, set $workbench_test, $apiserver_test or
# $cli_test on the command line:
#
# $ run-tests.sh --only workbench workbench_test=TEST=test/integration/pipeline_instances_test.rb
#
#
# To run a specific Python test set $python_sdk_test or $fuse_test.
#
# $ run-tests.sh --only python_sdk python_sdk_test="--test-suite tests.test_keep_locator"
#
#
# You can also pass "export ARVADOS_DEBUG=1" to enable additional debugging output:
#
# $ run-tests.sh "export ARVADOS_DEBUG=1"
#
#
# Finally, you can skip the installation steps on subsequent runs this way:
#
## First run
# $ run-tests.sh --leave-temp
#
## Subsequent runs: record the values of VENVDIR and GOPATH from the first run, and
# provide them on the command line in subsequent runs:
#
# $ run-tests.sh --skip-install VENVDIR="/tmp/tmp.y3tsTmigio" GOPATH="/tmp/tmp.3r4sSA9F3l"


# First make sure to remove any ARVADOS_ variables from the calling environment
# that could interfer with the tests.
unset $(env | cut -d= -f1 | grep \^ARVADOS_)

COLUMNS=80

GOPATH=
VENVDIR=
cli_test=
workbench_test=
apiserver_test=
python_sdk_test=
ruby_sdk_test=
fuse_test=
leave_temp=
skip_install=

if [[ -f /etc/profile.d/rvm.sh ]]
then
    source /etc/profile.d/rvm.sh
fi

fatal() {
    clear_temp
    echo >&2 "Fatal: $* in ${FUNCNAME[1]} at ${BASH_SOURCE[1]} line ${BASH_LINENO[0]}"
    exit 1
}

declare -a failures
declare -A skip
declare -A leave_temp

# Always skip CLI tests. They don't know how to use run_test_server.py.
skip[cli]=1

while [[ -n "$1" ]]
do
    arg="$1"; shift
    case "$arg" in
        --skip)
            skipwhat="$1"; shift
            skip[$skipwhat]=1
            ;;
        --only)
            only="$1"; shift
            ;;
        --skip-install)
            skip_install=1
            ;;
        --leave-temp)
            leave_temp[VENVDIR]=1
            leave_temp[GOPATH]=1
            ;;
        *=*)
            eval $(echo $arg | cut -d= -f1)=\"$(echo $arg | cut -d= -f2-)\"
            ;;
        *)
            echo >&2 "$0: Unrecognized option: '$arg'"
            exit 1
            ;;
    esac
done

# Sanity check
echo "WORKSPACE=$WORKSPACE"
[[ -n "$WORKSPACE" ]] || fatal "WORKSPACE not set"

if [[ -n "$CONFIGSRC" ]]; then
    if [[ -d "$HOME/arvados-api-server" ]]; then
        # Jenkins expects us to use this by default.
        CONFIGSRC="$HOME/arvados-api-server"
    fi
fi

# Set up temporary install dirs (unless existing dirs were supplied)
if [[ -n "$VENVDIR" ]]; then
    VENVDIR=$(mktemp -d)
else
    leave_temp[VENVDIR]=1
fi
if [[ -n "$GOPATH" ]]; then
    GOPATH=$(mktemp -d)
else
    leave_temp[GOPATH]=1
fi
export GOPATH
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git" \
    || fatal "symlink failed"

virtualenv --setuptools "$VENVDIR" || fatal "virtualenv $VENVDIR failed"
PATH="$VENVDIR/bin:$PATH"

checkexit() {
    if [[ "$?" != "0" ]]; then
        title "!!!!!! $1 FAILED !!!!!!"
        failures+=("$1 (`timer`)")
    else
        successes+=("$1 (`timer`)")
    fi
}

timer_reset() {
    t0=$SECONDS
}

timer() {
    echo -n "$(($SECONDS - $t0))s"
}

do_test() {
    if [[ -z "${skip[$1]}" ]] && ( [[ -z "$only" ]] || [[ "$only" == "$1" ]] )
    then
        title "Running $1 tests"
        timer_reset
        if [[ "$2" == "go" ]]
        then
            go test "git.curoverse.com/arvados.git/$1"
        else
            "test_$1"
        fi
        checkexit "$1 tests"
        title "End of $1 tests (`timer`)"
    else
        title "Skipping $1 tests"
    fi
}

do_install() {
    if [[ -z "$skip_install" ]]
    then
        title "Running $1 install"
        timer_reset
        if [[ "$2" == "go" ]]
        then
            go get -t "git.curoverse.com/arvados.git/$1"
        else
            "install_$1"
        fi
        checkexit "$1 install"
        title "End of $1 install (`timer`)"
    else
        title "Skipping $1 install"
    fi
}

title () {
    txt="********** $1 **********"
    printf "\n%*s%s\n\n" $((($COLUMNS-${#txt})/2)) "" "$txt"
}

clear_temp() {
    for var in VENVDIR GOPATH
    do
        if [[ -z "${leave_temp[$var]}" ]]
        then
            if [[ -n "${!var}" ]]
            then
                rm -rf "${!var}"
            fi
        else
            echo "Leaving $var=\"${!var}\""
        fi
    done
}

test_docs() {
    cd "$WORKSPACE/doc"
    bundle install --no-deployment
    rm -rf .site
    # Make sure python-epydoc is installed or the next line won't do much good!
    ARVADOS_API_HOST=qr1hi.arvadosapi.com
    PYTHONPATH=$WORKSPACE/sdk/python/ bundle exec rake generate baseurl=file://$WORKSPACE/doc/.site/ arvados_workbench_host=workbench.$ARVADOS_API_HOST arvados_api_host=$ARVADOS_API_HOST
    unset ARVADOS_API_HOST
}
do_test docs

test_doclinkchecker() {
    cd "$WORKSPACE/doc"
    bundle exec rake linkchecker baseurl=file://$WORKSPACE/doc/.site/
}
do_test doclinkchecker

test_ruby_sdk() {
    cd "$WORKSPACE/sdk/ruby" \
        && bundle install --no-deployment \
        && bundle exec rake test
}
do_test ruby_sdk

install_ruby_sdk() {
    cd "$WORKSPACE/sdk/ruby" \
        && gem build arvados.gemspec \
        && gem install --no-ri --no-rdoc `ls -t arvados-*.gem|head -n1`
}
do_install ruby_sdk

install_cli() {
    cd "$WORKSPACE/sdk/cli" \
        && gem build arvados-cli.gemspec \
        && gem install --no-ri --no-rdoc `ls -t arvados-cli-*.gem|head -n1`
}
do_install cli

test_cli() {
    title "Starting SDK CLI tests"
    cd "$WORKSPACE/sdk/cli" \
        && bundle install --no-deployment \
        && mkdir -p /tmp/keep \
        && KEEP_LOCAL_STORE=/tmp/keep bundle exec rake test $cli_test
}
do_test cli

install_apiserver() {
    cd "$WORKSPACE/services/api"
    bundle install --no-deployment

    rm -f config/environments/test.rb
    cp config/environments/test.rb.example config/environments/test.rb

    if [ -n "$CONFIGSRC" ]
    then
        for f in database.yml application.yml
        do
            cp "$CONFIGSRC/$f" config/ || fatal "$f"
        done
    fi

    # Fill in a random secret_token and blob_signing_key for testing
    SECRET_TOKEN=`echo 'puts rand(2**512).to_s(36)' |ruby`
    BLOB_SIGNING_KEY=`echo 'puts rand(2**512).to_s(36)' |ruby`

    sed -i'' -e "s:SECRET_TOKEN:$SECRET_TOKEN:" config/application.yml
    sed -i'' -e "s:BLOB_SIGNING_KEY:$BLOB_SIGNING_KEY:" config/application.yml

    export RAILS_ENV=test

    # Set up empty git repo (for git tests)
    GITDIR="$WORKSPACE/tmpgit"
    sed -i'' -e "s:/var/cache/git:$GITDIR:" config/application.default.yml

    rm -rf $GITDIR
    mkdir -p $GITDIR/test
    cd $GITDIR/test \
        && git init \
        && git config user.email "jenkins@ci.curoverse.com" \
        && git config user.name "Jenkins, CI" \
        && touch tmp \
        && git add tmp \
        && git commit -m 'initial commit'

    # Clear out any lingering postgresql connections to arvados_test, so that we can drop it
    # This assumes the current user is a postgresql superuser
    psql arvados_test -c "SELECT pg_terminate_backend (pg_stat_activity.procpid::int) FROM pg_stat_activity WHERE pg_stat_activity.datname = 'arvados_test';" 2>/dev/null

    cd "$WORKSPACE/services/api" \
        && bundle exec rake db:drop \
        && bundle exec rake db:create \
        && bundle exec rake db:setup
}
do_install apiserver

test_apiserver() {
    cd "$WORKSPACE/services/api"
    bundle exec rake test $apiserver_test
}
do_test apiserver

declare -a gostuff
gostuff=(
    services/keepstore
    services/keepproxy
    sdk/go/arvadosclient
    sdk/go/keepclient
    sdk/go/streamer
    )
for g in "${gostuff[@]}"
do
    do_install "$g" go
done

install_python_sdk() {
    # Install the Python SDK early. Various other test suites (like
    # keepproxy) bring up run_test_server.py, which imports the arvados
    # module. We can't actually *test* the Python SDK yet though, because
    # its own test suite brings up some of those other programs (like
    # keepproxy).

    cd "$WORKSPACE/sdk/python" \
        && python setup.py egg_info -b ".$(git log --format=format:%ct.%h -n1 .)" sdist rotate --keep=1 --match .tar.gz \
        && pip install dist/arvados-python-client-0.1.*.tar.gz
}
do_install python_sdk

install_fuse() {
    cd "$WORKSPACE/services/fuse" \
        && python setup.py egg_info -b ".$(git log --format=format:%ct.%h -n1 .)" sdist rotate --keep=1 --match .tar.gz \
        && pip install dist/arvados_fuse-0.1.*.tar.gz
}
do_install fuse

test_python_sdk() {
    # Python SDK. We test this before testing keepproxy: keepproxy runs
    # run_test_server.py, which depends on the yaml package, which is in
    # tests_require but not install_requires, and therefore does not get
    # installed by setuptools until we run "setup.py test" *and* install
    # the .egg files that setup.py downloads.

    cd "$WORKSPACE/sdk/python" \
        && python setup.py test $python_sdk_test
    r=$?
    easy_install *.egg
    return $r
}
do_test python_sdk

test_fuse() {
    # Install test dependencies here too, in case run_test_server needs them.
    cd "$WORKSPACE/services/fuse" \
        && python setup.py test $fuse_test
    r=$?
    easy_install *.egg
    return $r
}
do_test fuse

for g in "${gostuff[@]}"
do
    do_test "$g" go
done

test_workbench() {
    cd "$WORKSPACE/apps/workbench" \
        && bundle install --no-deployment \
        && bundle exec rake test $workbench_test
}
do_test workbench

clear_temp

for x in "${successes[@]}"
do
    echo "Pass: $x"
done

if [[ ${#failures[@]} == 0 ]]
then
    echo "All test suites passed."
else
    echo "Failures (${#failures[@]}):"
    for x in "${failures[@]}"
    do
        echo "Fail: $x"
    done
fi
exit ${#failures}
