#!/bin/bash

# Install and test Arvados components.
#
# Exit non-zero if any tests fail.
#
# Arguments:
# --skip FOO   Do not test the FOO component.
# --only FOO   Do not test anything except the FOO component.
#
# Regardless of which components are tested, install all components in
# the usual sequence. (Many test suites depend on other components
# being installed.)

COLUMNS=80

ARVADOS_API_HOST=qr1hi.arvadosapi.com

export GOPATH=$(mktemp -d)
VENVDIR=$(mktemp -d)

source /etc/profile.d/rvm.sh

fatal() {
    clear_temp
    echo >&2 "Fatal: $* in ${FUNCNAME[1]} at ${BASH_SOURCE[1]} line ${BASH_LINENO[0]}"
    exit 1
}

# Sanity check
echo "WORKSPACE=$WORKSPACE"
[[ -n "$WORKSPACE" ]] || fatal "WORKSPACE not set"

# Set up temporary install dirs
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git" \
    || fatal "symlink failed"

virtualenv --setuptools "$VENVDIR" || fatal "virtualenv $VENVDIR failed"
PATH="$VENVDIR/bin:$PATH"

declare -a failures
declare -A skip

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
        *)
            echo >&2 "$0: Unrecognized option: '$arg'"
            exit 1
            ;;
    esac
done

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
}

title () {
    txt="********** $1 **********"
    printf "\n%*s%s\n\n" $((($COLUMNS-${#txt})/2)) "" "$txt"
}

clear_temp() {
    for t in "$VENVDIR" "$GOPATH"
    do
        if [[ -n "$t" ]]
        then
            rm -rf "$t"
        fi
    done
}

install_docs() {
    cd "$WORKSPACE/doc"
    bundle install --deployment
    rm -rf .site
    # Make sure python-epydoc is installed or the next line won't do much good!
    PYTHONPATH=$WORKSPACE/sdk/python/ bundle exec rake generate baseurl=file://$WORKSPACE/doc/.site/ arvados_workbench_host=workbench.$ARVADOS_API_HOST arvados_api_host=$ARVADOS_API_HOST
}
do_install docs

test_doclinkchecker() {
    cd "$WORKSPACE/doc"
    bundle exec rake linkchecker baseurl=file://$WORKSPACE/doc/.site/
}
do_test doclinkchecker

install_apiserver() {
    cd "$WORKSPACE/services/api"
    bundle install --deployment

    rm -f config/environments/test.rb
    cp config/environments/test.rb.example config/environments/test.rb

    cp $HOME/arvados-api-server/database.yml config/ || fatal "database.yml"
    cp $HOME/arvados-api-server/application.yml config/ || fatal "application.yml"

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

    cd "$WORKSPACE/services/api" \
        && bundle exec rake db:drop \
        && bundle exec rake db:create \
        && bundle exec rake db:setup
}
do_install apiserver

test_apiserver() {
    cd "$WORKSPACE/services/api"
    bundle exec rake test
}
do_test apiserver

install_cli() {
    cd "$WORKSPACE/sdk/cli"
    bundle install --deployment
}
do_install cli

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

test_fuse() {
    cd "$WORKSPACE/services/fuse" \
        && python setup.py egg_info -b ".$(git log --format=format:%ct.%h -n1 .)" sdist rotate --keep=1 --match .tar.gz \
        && pip install dist/arvados_fuse-0.1.*.tar.gz
}
do_test fuse

test_python_sdk() {
    # Python SDK. We test this before testing keepproxy: keepproxy runs
    # run_test_server.py, which depends on the yaml package, which is in
    # tests_require but not install_requires, and therefore does not get
    # installed by setuptools until we run "setup.py test" *and* install
    # the .egg files that setup.py downloads.

    cd "$WORKSPACE/sdk/python" \
        && python setup.py test \
        && easy_install *.egg
}
do_test python_sdk

install_fuse() {
    # Install test dependencies here too, in case run_test_server needs them.
    cd "$WORKSPACE/services/fuse" \
        && python setup.py test \
        && easy_install *.egg
}
do_install fuse

for g in "${gostuff[@]}"
do
    do_test "$g" go
done

test_workbench() {
    cd "$WORKSPACE/apps/workbench" \
        && bundle install --deployment \
        && bundle exec rake test
}
do_test workbench

test_cli() {
    title "Starting SDK CLI tests"
    cd "$WORKSPACE/sdk/cli" \
        && bundle install --deployment \
        && mkdir -p /tmp/keep \
        && KEEP_LOCAL_STORE=/tmp/keep bundle exec rake test
}
do_test cli

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
