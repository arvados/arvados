#!/bin/bash

EXITCODE=0

COLUMNS=80

ARVADOS_API_HOST=qr1hi.arvadosapi.com

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

source /etc/profile.d/rvm.sh
echo $WORKSPACE

export GOPATH="$HOME/gocode"
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"

# DOCS
title "Starting DOC build"
cd "$WORKSPACE"
cd doc
bundle install --deployment
rm -rf .site
# Make sure python-epydoc is installed or the next line won't do much good!
PYTHONPATH=$WORKSPACE/sdk/python/ bundle exec rake generate baseurl=file://$WORKSPACE/doc/.site/ arvados_workbench_host=workbench.$ARVADOS_API_HOST arvados_api_host=$ARVADOS_API_HOST

checkexit() {
    ECODE=$?

    if [[ "$ECODE" != "0" ]]; then
        title "!!!!!! $1 FAILED !!!!!!"
        EXITCODE=$(($EXITCODE + $ECODE))
    fi
}

checkexit "Doc build"
title "DOC build complete"

# DOC linkchecker
title "Starting DOC linkchecker"
cd "$WORKSPACE"
cd doc
bundle exec rake linkchecker baseurl=file://$WORKSPACE/doc/.site/

checkexit "Doc linkchecker"
title "DOC linkchecker complete"

# API SERVER
title "Starting API server tests"
cd "$WORKSPACE"
cd services/api
bundle install --deployment

rm -f config/database.yml
rm -f config/environments/test.rb
cp config/environments/test.rb.example config/environments/test.rb

# Get test database config
cp $HOME/arvados-api-server/database.yml config/
# Get test application.yml file
cp $HOME/arvados-api-server/application.yml config/

# Fill in a random secret_token and blob_signing_key for testing
SECRET_TOKEN=`echo 'puts rand(2**512).to_s(36)' |ruby`
BLOB_SIGNING_KEY=`echo 'puts rand(2**512).to_s(36)' |ruby`

sed -i'' -e "s:SECRET_TOKEN:$SECRET_TOKEN:" config/application.yml
sed -i'' -e "s:BLOB_SIGNING_KEY:$BLOB_SIGNING_KEY:" config/application.yml

export RAILS_ENV=test

# Set up empty git repo (for git tests)
GITDIR=$WORKSPACE/tmpgit
rm -rf $GITDIR
mkdir $GITDIR
sed -i'' -e "s:/var/cache/git:$GITDIR:" config/application.default.yml

rm -rf $GITDIR
mkdir -p $GITDIR/test
cd $GITDIR/test
/usr/bin/git init
/usr/bin/git config user.email "jenkins@ci.curoverse.com"
/usr/bin/git config user.name "Jenkins, CI"
touch tmp
/usr/bin/git add tmp
/usr/bin/git commit -m 'initial commit'

cd "$WORKSPACE"
cd services/api

bundle exec rake db:drop
bundle exec rake db:create
bundle exec rake db:setup
bundle exec rake test

checkexit "API server tests"
title "API server tests complete"

# Install and test Go bits. keepstore must come before keepproxy and keepclient.
for dir in services/keepstore services/keepproxy sdk/go/arvadosclient sdk/go/keepclient sdk/go/streamer
do
  title "Starting $dir tests"
  cd "$WORKSPACE"

  go get -t "git.curoverse.com/arvados.git/$dir" \
  && go test "git.curoverse.com/arvados.git/$dir"

  checkexit "$dir tests"
  title "$dir tests complete"
done


# WORKBENCH
title "Starting workbench tests"
cd "$WORKSPACE"
cd apps/workbench
bundle install --deployment

echo $PATH


bundle exec rake test

checkexit "Workbench tests"
title "Workbench tests complete"

# Python SDK
title "Starting Python SDK tests"
cd "$WORKSPACE"
cd sdk/cli
bundle install --deployment

# Set up Python SDK and dependencies

cd "$WORKSPACE"
cd sdk/python

VENVDIR=$(mktemp -d)
virtualenv --setuptools "$VENVDIR"
"$VENVDIR/bin/python" setup.py test

checkexit "Python SDK tests"

"$VENVDIR/bin/python" setup.py egg_info -b ".$(git log --format=format:%ct.%h -n1 .)" sdist rotate --keep=1 --match .tar.gz
"$VENVDIR/bin/pip" install dist/arvados-python-client-0.1.*.tar.gz

checkexit "Python SDK install"

cd "$WORKSPACE"
cd services/fuse

# We reuse $VENVDIR from the Python SDK tests above
"$VENVDIR/bin/python" setup.py test

checkexit "FUSE tests"

"$VENVDIR/bin/python" setup.py egg_info -b ".$(git log --format=format:%ct.%h -n1 .)" sdist rotate --keep=1 --match .tar.gz
"$VENVDIR/bin/pip" install dist/arvados_fuse-0.1.*.tar.gz

checkexit "FUSE install"

title "Python SDK tests complete"

# Clean up $VENVDIR
rm -rf "$VENVDIR"

# The CLI SDK tests require a working API server, so let's skip those for now.
exit $EXITCODE

# CLI SDK
title "Starting SDK CLI tests"
cd "$WORKSPACE"
cd sdk/cli
bundle install --deployment

# Set up Python SDK and dependencies
cd ../python
rm -rf $HOME/lib/python
mkdir $HOME/lib/python
PYTHONPATH="$HOME/lib/python" easy_install --install-dir=$HOME/lib/python --upgrade google-api-python-client
PYTHONPATH="$HOME/lib/python" python setup.py install --home=$HOME

cd ../cli
mkdir -p /tmp/keep
export KEEP_LOCAL_STORE=/tmp/keep
PYTHONPATH="$HOME/lib/python" bundle exec rake test

checkexit "SDK CLI tests"
title "SDK CLI tests complete"

exit $EXITCODE
