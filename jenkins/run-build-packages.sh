#!/bin/bash


read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados packages and (optionally) upload them.

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

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

# Test for fpm
fpm --version >/dev/null 2>&1

if [[ "$?" != 0 ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: fpm not found"
  echo >&2
  exit 1
fi

if [[ "$DEBUG" != 0 ]]; then
  echo "Workspace is $WORKSPACE"
fi

version_from_git() {
  # Generates a version number from the git log for the current working
  # directory, and writes it to stdout.
  local git_ts git_hash
  declare $(TZ=UTC git log -n1 --first-parent --max-count=1 \
      --format=format:"git_ts=%ct git_hash=%h" .)
  echo "0.1.$(date -ud "@$git_ts" +%Y%m%d%H%M%S).$git_hash"
}

handle_python_package () {
  # This function assumes the current working directory is the python package directory
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
}

# Build debs for everything
build_and_scp_deb () {
  # The package source.  Depending on the source type, this can be a
  # path, or the name of the package in an upstream repository (e.g.,
  # pip).
  PACKAGE=$1
  shift
  # The name of the package to build.  Defaults to $PACKAGE.
  PACKAGE_NAME=$1
  shift
  # Optional: the vendor of the package.  Should be "Curoverse, Inc." for
  # packages of our own software.  Passed to fpm --vendor.
  VENDOR=$1
  shift
  # The type of source package.  Passed to fpm -s.  Default "python".
  PACKAGE_TYPE=$1
  shift
  # Optional: the package version number.  Passed to fpm -v.
  VERSION=$1
  shift

  if [[ "$PACKAGE_NAME" == "" ]]; then
    PACKAGE_NAME=$PACKAGE
  fi

  if [[ "$PACKAGE_TYPE" == "" ]]; then
    PACKAGE_TYPE='python'
  fi

  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "-s" "$PACKAGE_TYPE" "-t" "deb" "-x" "usr/local/lib/python2.7/dist-packages/tests")

  if [[ "$PACKAGE_NAME" != "$PACKAGE" ]]; then
    COMMAND_ARR+=('-n' "$PACKAGE_NAME")
  fi

  if [[ "$VENDOR" != "" ]]; then
    COMMAND_ARR+=('--vendor' "$VENDOR")
  fi

  if [[ "$VERSION" != "" ]]; then
    COMMAND_ARR+=('-v' "$VERSION")
  fi

  # Append remaining function arguments directly to fpm's command line.
  for i; do
    COMMAND_ARR+=("$i")
  done

  COMMAND_ARR+=("$PACKAGE")

  if [[ "$DEBUG" != 0 ]]; then
    echo
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
    echo "Error: $PACKAGE: Unable to figure out package name from fpm results:"
    echo
    echo $FPM_RESULTS
    echo
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

source /etc/profile.d/rvm.sh

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# More cleanup - make sure all executables that we'll package are 755
find -type d -name 'bin' |xargs -I {} find {} -type f |xargs -I {} chmod 755 {}

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

if [[ "$DEBUG" != 0 ]]; then
  echo "umask is" `umask`
fi

# Perl packages
if [[ "$DEBUG" != 0 ]]; then
  echo -e "\nPerl packages\n"
fi

if [[ "$DEBUG" != 0 ]]; then
  PERL_OUT=/dev/stdout
else
  PERL_OUT=/dev/null
fi

cd "$WORKSPACE/sdk/perl"

if [[ -e Makefile ]]; then
  make realclean >"$PERL_OUT"
fi
find -maxdepth 1 \( -name 'MANIFEST*' -or -name 'libarvados-perl_*.deb' \) \
    -delete
rm -rf install

perl Makefile.PL >"$PERL_OUT" && \
    make install PREFIX=install INSTALLDIRS=perl >"$PERL_OUT" && \
    build_and_scp_deb install/=/usr libarvados-perl "Curoverse, Inc." dir \
      "$(version_from_git)"

# Ruby gems
if [[ "$DEBUG" != 0 ]]; then
  echo
  echo "Ruby gems"
  echo
fi

if type rvm-exec 2>/dev/null; then
  FPM_GEM_PREFIX=$(rvm-exec system gem environment gemdir)
else
  FPM_GEM_PREFIX=$(gem environment gemdir)
fi

cd "$WORKSPACE"
cd sdk/ruby
# clean up old packages
find -maxdepth 1 \( -name 'arvados-*.gem' -or -name 'rubygem-arvados_*.deb' \) \
    -delete

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

build_and_scp_deb arvados-*.gem "" "Curoverse, Inc." gem "" \
    --prefix "$FPM_GEM_PREFIX"

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

# Python packages
if [[ "$DEBUG" != 0 ]]; then
  echo
  echo "Python packages"
  echo
fi

cd "$WORKSPACE"
PKG_VERSION=$(version_from_git)

cd sdk/python
handle_python_package

cd ../../services/fuse
handle_python_package

cd ../../services/nodemanager
handle_python_package

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
build_and_scp_deb $WORKSPACE/src-build-dir/=/usr/local/arvados/src arvados-src 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--exclude=usr/local/arvados/src/.git" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

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
build_and_scp_deb $GOPATH/bin/keepstore=/usr/bin/keepstore keepstore 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Keepstore is the Keep storage daemon, accessible to clients on the LAN"

# keepproxy
go get "git.curoverse.com/arvados.git/services/keepproxy"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/keepproxy=/usr/bin/keepproxy keepproxy 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Keepproxy makes a Keep cluster accessible to clients that are not on the LAN"

# crunchstat
go get "git.curoverse.com/arvados.git/services/crunchstat"
cd $WORKSPACE/debs
build_and_scp_deb $GOPATH/bin/crunchstat=/usr/bin/crunchstat crunchstat 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Crunchstat gathers cpu/memory/network statistics of running Crunch jobs"

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

# The node manager
cd $WORKSPACE/debs
build_and_scp_deb $WORKSPACE/services/nodemanager arvados-node-manager 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/nodemanager/arvados_node_manager.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Arvados node manager"

# A few dependencies
for deppkg in python-gflags pyvcf google-api-python-client oauth2client \
      pyasn1 pyasn1-modules rsa uritemplate httplib2 ws4py virtualenv \
      pykka apache-libcloud requests six pyexecjs; do
    build_and_scp_deb "$deppkg"
done

# cwltool from common-workflow-language. We use this in arv-run-pipeline-instance.
# We use $WORKSPACE/common-workflow-language as the clean directory from which to build the cwltool package
if [[ ! -d "$WORKSPACE/common-workflow-language" ]]; then
  mkdir "$WORKSPACE/common-workflow-language"
  cd "$WORKSPACE"
  if [[ "$DEBUG" != 0 ]]; then
    git clone https://github.com/rabix/common-workflow-language.git common-workflow-language
  else
    git clone -q https://github.com/rabix/common-workflow-language.git common-workflow-language
  fi
fi

cd "$WORKSPACE/common-workflow-language"
if [[ "$DEBUG" != 0 ]]; then
  git checkout master
  git pull
else
  git checkout -q master
  git pull -q
fi

cd reference
handle_python_package
CWLTOOL_VERSION=`git log --first-parent --max-count=1 --format='format:0.1.%ct.%h'`

# Build cwltool package
cd $WORKSPACE/debs

build_and_scp_deb $WORKSPACE/common-workflow-language/reference cwltool 'Common Workflow Language Working Group' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/common-workflow-language/reference/cwltool.egg-info/PKG-INFO)"

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
