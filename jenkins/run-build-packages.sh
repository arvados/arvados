#!/bin/bash

EXITCODE=0
CALL_PRM=0

APTUSER=$1
APTSERVER=$2

if [[ "$APTUSER" == '' ]]; then
  echo "Syntax: $0 <aptuser> <aptserver>"
  exit 1
fi

if [[ "$APTSERVER" == '' ]]; then
  echo "Syntax: $0 <aptuser> <aptserver>"
  exit 1
fi

source /etc/profile.d/rvm.sh
echo $WORKSPACE

# Build arvados GEM
echo "Build and publish ruby gem"
cd "$WORKSPACE"
cd sdk/ruby
# clean up old gems
rm -f arvados-*gem
gem build arvados.gemspec
# publish new gem
gem push arvados-*gem

# Build arvados-cli GEM
echo "Build and publish ruby gem"
cd "$WORKSPACE"
cd sdk/cli
# clean up old gems
rm -f arvados-cli*gem
gem build arvados-cli.gemspec
# publish new gem
gem push arvados-cli*gem

# Build arvados-python-client Python package
echo "Build and publish arvados-python-client package"
cd "$WORKSPACE"

GIT_HASH=`git log --format=format:%ct.%h -n1 .`

cd sdk/python

# We mess with this file below, reset it here
git checkout setup.py

# Make sure only to use sdist - that's the only format pip can deal with (sigh)
python setup.py egg_info -b ".$GIT_HASH" sdist upload

cd ../../services/fuse

# We mess with this file below, reset it here
git checkout setup.py

# Make sure only to use sdist - that's the only format pip can deal with (sigh)
python setup.py egg_info -b ".$GIT_HASH" sdist upload

# Build debs for everything

# Build arvados src deb package

build_and_scp_deb () {
  PACKAGE=$1
  PACKAGE_NAME=$2
  # Put spaces in $3 and you will regret it. Despite the use of arrays below.
  # Because, bash sucks.
  VENDOR=${3// /_}
  PACKAGE_TYPE=$4
  EXTRA_ARGUMENTS=$5

  if [[ "$PACKAGE_NAME" == "" ]]; then
    PACKAGE_NAME=$PACKAGE
  fi

  if [[ "$PACKAGE_TYPE" == "" ]]; then
    PACKAGE_TYPE='python'
  fi

  COMMAND_ARR=("fpm" "-s" "$PACKAGE_TYPE" "-t" "deb")

  if [[ "$PACKAGE_NAME" != "$PACKAGE" ]]; then
    COMMAND_ARR+=('-n' "$PACKAGE_NAME")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi
  for a in $EXTRA_ARGUMENTS; do
    COMMAND_ARR+=("$a")
  done

  COMMAND_ARR+=("$PACKAGE")

  FPM_RESULTS=$(${COMMAND_ARR[@]})
  FPM_EXIT_CODE=$?
  echo ${COMMAND_ARR[@]}
  if [[ ! $FPM_RESULTS =~ "File already exists" ]]; then
    if [[ "$FPM_EXIT_CODE" != "0" ]]; then
      echo "Error building debian package for $1:\n $FPM_RESULTS"
    else
      scp -P2222 $PACKAGE_NAME*.deb $APTUSER@$APTSERVER:tmp/
      CALL_PRM=1
    fi
  else
    echo "Debian package for $1 exists, not rebuilding"
  fi
}

if [[ ! -d "$WORKSPACE/debs" ]]; then
  mkdir -p $WORKSPACE/debs
fi

# Make sure our destination directory on $APTSERVER exists - prm can delete it when invoked improperly
ssh -p2222 $APTUSER@$APTSERVER mkdir tmp

# Arvados-src
# We use $WORKSPACE/src-build-dir as the clean directory from which to build the src package
if [[ ! -d "$WORKSPACE/src-build-dir" ]]; then
  mkdir "$WORKSPACE/src-build-dir"
  cd "$WORKSPACE"
  git clone https://github.com/curoverse/arvados.git src-build-dir
fi  

cd "$WORKSPACE/src-build-dir"
# just in case, check out master
git checkout master
git pull
# go into detached-head state
git checkout `git log --format=format:%h -n1 .`
cd $WORKSPACE

cd $WORKSPACE/debs
build_and_scp_deb $WORKSPACE/src-build-dir/=/usr/local/arvados/src arvados-src 'Curoverse, Inc.' 'dir' "-v 0.1.$GIT_HASH -x 'usr/local/arvados/src/.git*'"

# clean up, check out master and step away from detached-head state
cd "$WORKSPACE/src-build-dir"
git checkout master

# Keep
export GOPATH=$(mktemp -d)
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"

# Keep -> keepstore
go get "git.curoverse.com/arvados.git/services/keepstore"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/keepstore=/usr/bin/keep keep 'Curoverse, Inc.' 'dir' "-v 0.1.$GIT_HASH"

# Keep proxy

# Keep -> keepproxy
go get "git.curoverse.com/arvados.git/services/keepproxy"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/keepproxy=/usr/bin/keepproxy keepproxy 'Curoverse, Inc.' 'dir' "-v 0.1.$GIT_HASH"

# crunchstat
go get "git.curoverse.com/arvados.git/services/crunchstat"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/crunchstat=/usr/bin/crunchstat crunchstat 'Curoverse, Inc.' 'dir' "-v 0.1.$GIT_HASH"

# The Python SDK
cd $WORKSPACE/sdk/python
sed -i'' -e "s:version='0.1':version='0.1.$GIT_HASH':" setup.py

cd $WORKSPACE/debs

# Please resist the temptation to add --no-python-fix-name to the fpm call here
# (which would remove the python- prefix from the package name), because this
# package is a dependency of arvados-fuse, and fpm can not omit the python-
# prefix from only one of the dependencies of a package...  Maybe I could
# whip up a patch and send it upstream, but that will be for another day. Ward,
# 2014-05-15
build_and_scp_deb $WORKSPACE/sdk/python python-arvados-python-client 'Curoverse, Inc.' 'python' "-v 0.1.${GIT_HASH}"

# The FUSE driver
cd $WORKSPACE/services/fuse
sed -i'' -e "s:version='0.1':version='0.1.$GIT_HASH':" setup.py


cd $WORKSPACE/debs

# Please seem comment about --no-python-fix-name above; we stay consistent and do
# not omit the python- prefix first.
build_and_scp_deb $WORKSPACE/services/fuse python-arvados-fuse 'Curoverse, Inc.' 'python' "-v 0.1.${GIT_HASH}"

# A few dependencies
build_and_scp_deb python-gflags
build_and_scp_deb pyvcf
build_and_scp_deb google-api-python-client
build_and_scp_deb httplib2
build_and_scp_deb ws4py
build_and_scp_deb virtualenv

# Finally, publish the packages, if necessary
if [[ "$CALL_PRM" != "0" ]]; then
  ssh -p2222 $APTUSER@$APTSERVER -t "cd /var/www/$APTSERVER; /usr/local/rvm/bin/rvm default do prm --type deb -p . --component main --release wheezy --arch amd64  -d /home/$APTUSER/tmp/ --gpg 1078ECD7"
else
  echo "No new packages generated. No PRM run necessary."
fi

