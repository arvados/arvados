#!/bin/bash

EXITCODE=0
CALL_FREIGHT=0

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

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
  echo "WORKSPACE environment variable not set"
  exit 1
fi

source /etc/profile.d/rvm.sh
echo $WORKSPACE

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

echo "umask is"
umask

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

# Make sure only to use sdist - that's the only format pip can deal with (sigh)
python setup.py sdist upload

cd ../../services/fuse

# Make sure only to use sdist - that's the only format pip can deal with (sigh)
python setup.py sdist upload

# Build debs for everything
build_and_scp_deb () {
  PACKAGE=$1
  shift
  PACKAGE_NAME=$1
  shift
  VENDOR=$1
  shift
  PACKAGE_TYPE=$1
  shift
  VERSION=$1
  shift

  if [[ "$PACKAGE_NAME" == "" ]]; then
    PACKAGE_NAME=$PACKAGE
  fi

  if [[ "$PACKAGE_TYPE" == "" ]]; then
    PACKAGE_TYPE='python'
  fi

  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "-s" "$PACKAGE_TYPE" "-t" "deb")

  if [[ "$PACKAGE_NAME" != "$PACKAGE" ]]; then
    COMMAND_ARR+=('-n' "$PACKAGE_NAME")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi

  if [[ "$VERSION" != "" ]]; then
    COMMAND_ARR+=('-v' "$VERSION")
  fi

  for i; do
    COMMAND_ARR+=("$i")
  done

  COMMAND_ARR+=("$PACKAGE")

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?

  FPM_PACKAGE_NAME=''
  if [[ $FPM_RESULTS =~ ([A-Za-z0-9_\-.]*\.deb) ]]; then
    FPM_PACKAGE_NAME=${BASH_REMATCH[1]}
  fi

  if [[ "$FPM_PACKAGE_NAME" == "" ]]; then
    EXITCODE=1
    echo "Error: Unabled figure out package name from fpm results:\n $FPM_RESULTS"
  else
    if [[ ! $FPM_RESULTS =~ "File already exists" ]]; then
      if [[ "$FPM_EXIT_CODE" != "0" ]]; then
        echo "Error building debian package for $1:\n $FPM_RESULTS"
      else
        scp -P2222 $FPM_PACKAGE_NAME $APTUSER@$APTSERVER:tmp/
        CALL_FREIGHT=1
      fi
    else
      echo "Debian package $FPM_PACKAGE_NAME exists, not rebuilding"
    fi
  fi
}

if [[ ! -d "$WORKSPACE/debs" ]]; then
  mkdir -p $WORKSPACE/debs
fi

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

# Build arvados src deb package
cd $WORKSPACE/debs
build_and_scp_deb $WORKSPACE/src-build-dir/=/usr/local/arvados/src arvados-src 'Curoverse, Inc.' 'dir' "0.1.$GIT_HASH" "-x 'usr/local/arvados/src/.git*'" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

# clean up, check out master and step away from detached-head state
cd "$WORKSPACE/src-build-dir"
git checkout master

# Keep
export GOPATH=$(mktemp -d)
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"

# keepstore
go get "git.curoverse.com/arvados.git/services/keepstore"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/keepstore=/usr/bin/keepstore keepstore 'Curoverse, Inc.' 'dir' "0.1.$GIT_HASH" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Keepstore is the Keep storage daemon, accessible to clients on the LAN"

# keepproxy
go get "git.curoverse.com/arvados.git/services/keepproxy"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/keepproxy=/usr/bin/keepproxy keepproxy 'Curoverse, Inc.' 'dir' "0.1.$GIT_HASH" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Keepproxy makes a Keep cluster accessible to clients that are not on the LAN"

# crunchstat
go get "git.curoverse.com/arvados.git/services/crunchstat"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/crunchstat=/usr/bin/crunchstat crunchstat 'Curoverse, Inc.' 'dir' "0.1.$GIT_HASH" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Crunchstat gathers cpu/memory/network statistics of running Crunch jobs"

# The Python SDK
# Please resist the temptation to add --no-python-fix-name to the fpm call here
# (which would remove the python- prefix from the package name), because this
# package is a dependency of arvados-fuse, and fpm can not omit the python-
# prefix from only one of the dependencies of a package...  Maybe I could
# whip up a patch and send it upstream, but that will be for another day. Ward,
# 2014-05-15
cd $WORKSPACE/debs
build_and_scp_deb $WORKSPACE/sdk/python python-arvados-python-client 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/python/arvados_python_client.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Arvados Python SDK"

# The FUSE driver
# Please seem comment about --no-python-fix-name above; we stay consistent and do
# not omit the python- prefix first.
cd $WORKSPACE/debs
build_and_scp_deb $WORKSPACE/services/fuse python-arvados-fuse 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/fuse/arvados_fuse.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Keep FUSE driver"

# A few dependencies
build_and_scp_deb python-gflags
build_and_scp_deb pyvcf
build_and_scp_deb google-api-python-client
build_and_scp_deb httplib2
build_and_scp_deb ws4py
build_and_scp_deb virtualenv

# Finally, publish the packages, if necessary
if [[ "$CALL_FREIGHT" != "0" ]]; then
  ssh -p2222 $APTUSER@$APTSERVER -t "cd tmp && ls -laF *deb && freight add *deb apt/wheezy && freight cache && rm -f *deb"
else
  echo "No new packages generated. No freight run necessary."
fi

# clean up temporary GOPATH
rm -rf "$GOPATH"

exit $EXITCODE
