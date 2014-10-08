#!/bin/bash


read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados packages and (optionally) upload them.

Syntax:
        $(basename $0) WORKSPACE=/path/to/arvados [options]

Options:

--upload               Upload packages (default: false)
--scp-user USERNAME    Scp user for apt server (only required when --upload is specified)
--apt-server HOSTNAME  Apt server hostname (only required when --upload is specified)
--debug                Output debug information (default: false)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

EXITCODE=0
CALL_FREIGHT=0

DEBUG=0
UPLOAD=0

while [[ -n "$1" ]]
do
    arg="$1"; shift
    case "$arg" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --scp-user)
            APTUSER="$1"; shift
            ;;
        --apt-server)
            APTSERVER="$1"; shift
            ;;
        --debug)
            DEBUG=1
            ;;
        --upload)
            UPLOAD=1
            ;;
        *)
            echo >&2 "$0: Unrecognized option: '$arg'. Try: $0 --help"
            exit 1
            ;;
    esac
done

# Sanity checks
if [[ "$UPLOAD" != '0' && ("$APTUSER" == '' || "$APTSERVER" == '') ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: please specify --scp-user and --apt-server if --upload is set"
  echo >&2
  exit 1
fi

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

source /etc/profile.d/rvm.sh

if [[ "$DEBUG" != 0 ]]; then
  echo "Workspace is $WORKSPACE"
fi

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

if [[ "$DEBUG" != 0 ]]; then
  echo "umask is" `umask`
fi

# Build arvados GEM
if [[ "$DEBUG" != 0 ]]; then
  echo "Build and publish ruby gems"
fi

cd "$WORKSPACE"
cd sdk/ruby
# clean up old gems
rm -f arvados-*gem

if [[ "$DEBUG" != 0 ]]; then
  gem build arvados.gemspec
else
  # -q appears to be broken in gem version 2.2.2
  gem build arvados.gemspec -q >/dev/null
fi

if [[ "$UPLOAD" != 0 ]]; then
  # publish new gem
  gem push arvados-*gem
fi

# Build arvados-cli GEM
cd "$WORKSPACE"
cd sdk/cli
# clean up old gems
rm -f arvados-cli*gem

if [[ "$DEBUG" != 0 ]]; then
  gem build arvados-cli.gemspec
else
  # -q appears to be broken in gem version 2.2.2
  gem build arvados-cli.gemspec -q >/dev/null
fi

if [[ "$UPLOAD" != 0 ]]; then
  # publish new gem
  gem push arvados-cli*gem
fi

# Build arvados-python-client Python package
if [[ "$DEBUG" != 0 ]]; then
  echo "Build and publish arvados-python-client package"
fi

cd "$WORKSPACE"

GIT_HASH=`git log --format=format:%ct.%h -n1 .`

cd sdk/python

# Make sure only to use sdist - that's the only format pip can deal with (sigh)

if [[ "$UPLOAD" != 0 ]]; then
  # Make sure only to use sdist - that's the only format pip can deal with (sigh)
  if [[ "$DEBUG" != 0 ]]; then
    python setup.py sdist upload
  else
    python setup.py -q sdist upload
  fi
else
  # Make sure only to use sdist - that's the only format pip can deal with (sigh)
  if [[ "$DEBUG" != 0 ]]; then
    python setup.py sdist
  else
    python setup.py -q sdist
  fi
fi

cd ../../services/fuse

if [[ "$UPLOAD" != 0 ]]; then
  # Make sure only to use sdist - that's the only format pip can deal with (sigh)
  if [[ "$DEBUG" != 0 ]]; then
    python setup.py sdist upload
  else
    python setup.py -q sdist upload
  fi
else
  # Make sure only to use sdist - that's the only format pip can deal with (sigh)
  if [[ "$DEBUG" != 0 ]]; then
    python setup.py sdist
  else
    python setup.py -q sdist
  fi
fi

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

  if [[ "$DEBUG" != 0 ]]; then
    echo
    echo "Fpm command:"
    echo "${COMMAND_ARR[@]}"
    echo
  fi

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?

  FPM_PACKAGE_NAME=''
  if [[ $FPM_RESULTS =~ ([A-Za-z0-9_\-.]*\.deb) ]]; then
    FPM_PACKAGE_NAME=${BASH_REMATCH[1]}
  fi

  if [[ "$FPM_PACKAGE_NAME" == "" ]]; then
    EXITCODE=1
    echo "Error: Unable to figure out package name from fpm results:\n $FPM_RESULTS"
  else
    if [[ ! $FPM_RESULTS =~ "File already exists" ]]; then
      if [[ "$FPM_EXIT_CODE" != "0" ]]; then
        echo "Error building debian package for $1:\n $FPM_RESULTS"
      else
        if [[ "$UPLOAD" != 0 ]]; then
          scp -P2222 $FPM_PACKAGE_NAME $APTUSER@$APTSERVER:tmp/
          CALL_FREIGHT=1
        fi
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
  if [[ "$DEBUG" != 0 ]]; then
    git clone https://github.com/curoverse/arvados.git src-build-dir
  else
    git clone -q https://github.com/curoverse/arvados.git src-build-dir
  fi
fi

cd "$WORKSPACE/src-build-dir"
# just in case, check out master
if [[ "$DEBUG" != 0 ]]; then
  git checkout master
  git pull
  # go into detached-head state
  git checkout `git log --format=format:%h -n1 .`
else
  git checkout -q master
  git pull -q
  # go into detached-head state
  git checkout -q `git log --format=format:%h -n1 .`
fi

# Build arvados src deb package
cd $WORKSPACE/debs
build_and_scp_deb $WORKSPACE/src-build-dir/=/usr/local/arvados/src arvados-src 'Curoverse, Inc.' 'dir' "0.1.$GIT_HASH" "-x 'usr/local/arvados/src/.git*'" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

# clean up, check out master and step away from detached-head state
cd "$WORKSPACE/src-build-dir"
if [[ "$DEBUG" != 0 ]]; then
  git checkout master
else
  git checkout -q master
fi

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
if [[ "$UPLOAD" != 0 && "$CALL_FREIGHT" != 0 ]]; then
  ssh -p2222 $APTUSER@$APTSERVER -t "cd tmp && ls -laF *deb && freight add *deb apt/wheezy && freight cache && rm -f *deb"
else
  if [[ "$UPLOAD" != 0 ]]; then
    echo "No new packages generated. No freight run necessary."
  fi
fi

# clean up temporary GOPATH
rm -rf "$GOPATH"

exit $EXITCODE
