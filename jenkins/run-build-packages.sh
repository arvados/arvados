#!/bin/bash


read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados packages

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

Options:

--build-bundle-packages  (default: false)
    Build api server and workbench packages with vendor/bundle included
--debug
    Output debug information (default: false)
--target
    Distribution to build packages for (default: debian7)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

EXITCODE=0
DEBUG=${ARVADOS_DEBUG:-0}
BUILD_BUNDLE_PACKAGES=0
TARGET=debian7

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,build-bundle-packages,debug,target: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

eval set -- "$PARSEDOPTS"
while [ $# -gt 0 ]; do
    case "$1" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --target)
            TARGET="$2"; shift
            ;;
        --debug)
            DEBUG=1
            ;;
        --build-bundle-packages)
            BUILD_BUNDLE_PACKAGES=1
            ;;
        --)
            if [ $# -gt 1 ]; then
                echo >&2 "$0: unrecognized argument '$2'. Try: $0 --help"
                exit 1
            fi
            ;;
    esac
    shift
done

STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
if [[ "$DEBUG" != 0 ]]; then
    STDOUT_IF_DEBUG=/dev/stdout
    STDERR_IF_DEBUG=/dev/stderr
    DASHQ_UNLESS_DEBUG=
fi

debug_echo () {
    echo "$@" >"$STDOUT_IF_DEBUG"
}

declare -a PYTHON_BACKPORTS PYTHON3_BACKPORTS

PYTHON2_VERSION=2.7
PYTHON3_VERSION=$(python3 -c 'import sys; print("{v.major}.{v.minor}".format(v=sys.version_info))')

case "$TARGET" in
    debian7)
        FORMAT=deb
        PYTHON2_PACKAGE=python$PYTHON2_VERSION
        PYTHON2_PKG_PREFIX=python
        PYTHON3_PACKAGE=python$PYTHON3_VERSION
        PYTHON3_PKG_PREFIX=python3
        PYTHON_BACKPORTS=(python-gflags pyvcf google-api-python-client \
            oauth2client pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py \
            virtualenv pykka apache-libcloud requests six pyexecjs jsonschema \
            ciso8601 pycrypto backports.ssl_match_hostname pycurl llfuse)
        PYTHON3_BACKPORTS=(docker-py six requests websocket-client)
        ;;
    debian8)
        FORMAT=deb
        PYTHON2_PACKAGE=python$PYTHON2_VERSION
        PYTHON2_PKG_PREFIX=python
        PYTHON3_PACKAGE=python$PYTHON3_VERSION
        PYTHON3_PKG_PREFIX=python3
        PYTHON_BACKPORTS=(python-gflags pyvcf google-api-python-client \
            oauth2client pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py \
            virtualenv pykka apache-libcloud requests six pyexecjs jsonschema \
            ciso8601 pycrypto backports.ssl_match_hostname pycurl llfuse)
        PYTHON3_BACKPORTS=(docker-py six requests websocket-client)
        ;;
    ubuntu1204)
        FORMAT=deb
        PYTHON2_PACKAGE=python$PYTHON2_VERSION
        PYTHON2_PKG_PREFIX=python
        PYTHON3_PACKAGE=python$PYTHON3_VERSION
        PYTHON3_PKG_PREFIX=python3
        PYTHON_BACKPORTS=(python-gflags pyvcf google-api-python-client \
            oauth2client pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py \
            virtualenv pykka apache-libcloud requests six pyexecjs jsonschema \
            ciso8601 pycrypto backports.ssl_match_hostname pycurl llfuse)
        PYTHON3_BACKPORTS=(docker-py six requests websocket-client)
        ;;
    ubuntu1404)
        FORMAT=deb
        PYTHON2_PACKAGE=python$PYTHON2_VERSION
        PYTHON2_PKG_PREFIX=python
        PYTHON3_PACKAGE=python$PYTHON3_VERSION
        PYTHON3_PKG_PREFIX=python3
        PYTHON_BACKPORTS=(pyasn1==0.1.7 pyasn1-modules==0.0.5 llfuse ciso8601 \
            google-api-python-client six uritemplate oauth2client httplib2 \
            rsa apache-libcloud pycurl backports.ssl_match_hostname)
        PYTHON3_BACKPORTS=(docker-py requests websocket-client)
        ;;
    centos6)
        FORMAT=rpm
        PYTHON2_PACKAGE=$(rpm -qf "$(which python$PYTHON2_VERSION)" --queryformat '%{NAME}\n')
        PYTHON2_PKG_PREFIX=$PYTHON2_PACKAGE
        PYTHON3_PACKAGE=$(rpm -qf "$(which python$PYTHON3_VERSION)" --queryformat '%{NAME}\n')
        PYTHON3_PKG_PREFIX=$PYTHON3_PACKAGE
        PYTHON_BACKPORTS=(python-gflags pyvcf google-api-python-client \
            oauth2client pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py \
            pykka apache-libcloud requests six pyexecjs jsonschema \
            ciso8601 pycrypto backports.ssl_match_hostname pycurl
            python-daemon lockfile llfuse)
        PYTHON3_BACKPORTS=(docker-py six requests)
        export PYCURL_SSL_LIBRARY=nss
        ;;
    *)
        echo -e "$0: Unknown target '$TARGET'.\n" >&2
        exit 1
        ;;
esac


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

find_easy_install() {
    for version_suffix in "$@"; do
        if "easy_install$version_suffix" --version >/dev/null 2>&1; then
            echo "easy_install$version_suffix"
            return 0
        fi
    done
    cat >&2 <<EOF
$helpmessage

Error: easy_install$1 (from Python setuptools module) not found

EOF
    exit 1
}

EASY_INSTALL2=$(find_easy_install -$PYTHON2_VERSION "")
EASY_INSTALL3=$(find_easy_install -$PYTHON3_VERSION 3)

RUN_BUILD_PACKAGES_PATH="`dirname \"$0\"`"
RUN_BUILD_PACKAGES_PATH="`( cd \"$RUN_BUILD_PACKAGES_PATH\" && pwd )`"  # absolutized and normalized
if [ -z "$RUN_BUILD_PACKAGES_PATH" ] ; then
  # error; for some reason, the path is not accessible
  # to the script (e.g. permissions re-evaled after suid)
  exit 1  # fail
fi

debug_echo "$0 is running from $RUN_BUILD_PACKAGES_PATH"
debug_echo "Workspace is $WORKSPACE"

format_last_commit_here() {
    local format=$1; shift
    TZ=UTC git log -n1 --first-parent "--format=format:$format" .
}

version_from_git() {
  # Generates a version number from the git log for the current working
  # directory, and writes it to stdout.
  local git_ts git_hash
  declare $(format_last_commit_here "git_ts=%ct git_hash=%h")
  echo "0.1.$(date -ud "@$git_ts" +%Y%m%d%H%M%S).$git_hash"
}

nohash_version_from_git() {
    version_from_git | cut -d. -f1-3
}

timestamp_from_git() {
    format_last_commit_here "%ct"
}

handle_python_package () {
  # This function assumes the current working directory is the python package directory
  if [ -n "$(find dist -name "*-$(nohash_version_from_git).tar.gz" -print -quit)" ]; then
    # This package doesn't need rebuilding.
    return
  fi
  # Make sure only to use sdist - that's the only format pip can deal with (sigh)
  python setup.py $DASHQ_UNLESS_DEBUG sdist
}

handle_ruby_gem() {
    local gem_name=$1; shift
    local gem_version=$(nohash_version_from_git)
    local gem_src_dir="$(pwd)"

    if ! [[ -e "${gem_name}-${gem_version}.gem" ]]; then
        find -maxdepth 1 -name "${gem_name}-*.gem" -delete

        # -q appears to be broken in gem version 2.2.2
        $GEM build "$gem_name.gemspec" $DASHQ_UNLESS_DEBUG >"$STDOUT_IF_DEBUG" 2>"$STDERR_IF_DEBUG"
    fi

    cd "$WORKSPACE/packages/$TARGET"
    fpm_build "$gem_src_dir/$gem_name"-*.gem "" "Curoverse, Inc." gem "" \
        --prefix "$FPM_GEM_PREFIX"
}

# Build packages for everything
fpm_build () {
  # The package source.  Depending on the source type, this can be a
  # path, or the name of the package in an upstream repository (e.g.,
  # pip).
  PACKAGE=$1
  shift
  # The name of the package to build.  Defaults to $PACKAGE.
  PACKAGE_NAME=${1:-$PACKAGE}
  shift
  # Optional: the vendor of the package.  Should be "Curoverse, Inc." for
  # packages of our own software.  Passed to fpm --vendor.
  VENDOR=$1
  shift
  # The type of source package.  Passed to fpm -s.  Default "python".
  PACKAGE_TYPE=${1:-python}
  shift
  # Optional: the package version number.  Passed to fpm -v.
  VERSION=$1
  shift

  case "$PACKAGE_TYPE" in
      python)
          # All Arvados Python2 packages depend on Python 2.7.
          # Make sure we build with that for consistency.
          set -- "$@" --python-bin python2.7 \
              --python-easyinstall "$EASY_INSTALL2" \
              --python-package-name-prefix "$PYTHON2_PKG_PREFIX" \
              --depends "$PYTHON2_PACKAGE"
          ;;
      python3)
          # fpm does not actually support a python3 package type.  Instead
          # we recognize it as a convenience shortcut to add several
          # necessary arguments to fpm's command line later, after we're
          # done handling positional arguments.
          PACKAGE_TYPE=python
          set -- "$@" --python-bin python3 \
              --python-easyinstall "$EASY_INSTALL3" \
              --python-package-name-prefix "$PYTHON3_PKG_PREFIX" \
              --depends "$PYTHON3_PACKAGE"
          ;;
  esac

  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "-s" "$PACKAGE_TYPE" "-t" "$FORMAT")
  if [ python = "$PACKAGE_TYPE" ]; then
    COMMAND_ARR+=(--exclude=\*/{dist,site}-packages/tests/\*)
  fi

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

  # Append --depends X and other arguments specified by fpm-info.sh in
  # the package source dir. These are added last so they can override
  # the arguments added by this script.
  declare -a fpm_args=()
  declare -a fpm_depends=()
  if [[ -d "$PACKAGE" ]]; then
      FPM_INFO="$PACKAGE/fpm-info.sh"
  else
      FPM_INFO="${WORKSPACE}/backports/${PACKAGE_TYPE}-${PACKAGE}/fpm-info.sh"
  fi
  if [[ -e "$FPM_INFO" ]]; then
      debug_echo "Loading fpm overrides from $FPM_INFO"
      source "$FPM_INFO"
  fi
  for i in "${fpm_depends[@]}"; do
    COMMAND_ARR+=('--depends' "$i")
  done
  COMMAND_ARR+=("${fpm_args[@]}")

  COMMAND_ARR+=("$PACKAGE")

  debug_echo -e "\n${COMMAND_ARR[@]}\n"

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?

  fpm_verify $FPM_EXIT_CODE $FPM_RESULTS
}

# verify build results
fpm_verify () {
  FPM_EXIT_CODE=$1
  shift
  FPM_RESULTS=$@

  FPM_PACKAGE_NAME=''
  if [[ $FPM_RESULTS =~ ([A-Za-z0-9_\.-]*\.)(deb|rpm) ]]; then
    FPM_PACKAGE_NAME=${BASH_REMATCH[1]}${BASH_REMATCH[2]}
  fi

  if [[ "$FPM_PACKAGE_NAME" == "" ]]; then
    EXITCODE=1
    echo "Error: $PACKAGE: Unable to figure out package name from fpm results:"
    echo
    echo $FPM_RESULTS
    echo
  elif [[ "$FPM_RESULTS" =~ "File already exists" ]]; then
    echo "Package $FPM_PACKAGE_NAME exists, not rebuilding"
  elif [[ 0 -ne "$FPM_EXIT_CODE" ]]; then
    echo "Error building package for $1:\n $FPM_RESULTS"
  fi
}

if [[ -f /etc/profile.d/rvm.sh ]]; then
    source /etc/profile.d/rvm.sh
    GEM="rvm-exec default gem"
else
    GEM=gem
fi

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# More cleanup - make sure all executables that we'll package are 755
find -type d -name 'bin' |xargs -I {} find {} -type f |xargs -I {} chmod 755 {}

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

debug_echo "umask is" `umask`

if [[ ! -d "$WORKSPACE/packages/$TARGET" ]]; then
  mkdir -p $WORKSPACE/packages/$TARGET
fi

# Perl packages
debug_echo -e "\nPerl packages\n"

cd "$WORKSPACE/sdk/perl"

if [[ -e Makefile ]]; then
  make realclean >"$STDOUT_IF_DEBUG"
fi
find -maxdepth 1 \( -name 'MANIFEST*' -or -name "libarvados-perl*.$FORMAT" \) \
    -delete
rm -rf install

perl Makefile.PL INSTALL_BASE=install >"$STDOUT_IF_DEBUG" && \
    make install INSTALLDIRS=perl >"$STDOUT_IF_DEBUG" && \
    fpm_build install/lib/=/usr/share libarvados-perl \
    "Curoverse, Inc." dir "$(version_from_git)" install/man/=/usr/share/man && \
    mv libarvados-perl*.$FORMAT "$WORKSPACE/packages/$TARGET/"

# Ruby gems
debug_echo -e "\nRuby gems\n"

FPM_GEM_PREFIX=$($GEM environment gemdir)

cd "$WORKSPACE/sdk/ruby"
handle_ruby_gem arvados

cd "$WORKSPACE/sdk/cli"
handle_ruby_gem arvados-cli

# Python packages
debug_echo -e "\nPython packages\n"

cd "$WORKSPACE/sdk/pam"
handle_python_package

cd "$WORKSPACE/sdk/python"
handle_python_package

cd "$WORKSPACE/services/fuse"
handle_python_package

cd "$WORKSPACE/services/nodemanager"
handle_python_package

# arvados-src
(
    set -e

    cd "$WORKSPACE"
    COMMIT_HASH=$(format_last_commit_here "%H")

    SRC_BUILD_DIR=$(mktemp -d)
    git clone $DASHQ_UNLESS_DEBUG --branch "$COMMIT_HASH" "$WORKSPACE/.git" "$SRC_BUILD_DIR"
    cd "$SRC_BUILD_DIR"
    echo "$COMMIT_HASH" >git-commit.version

    cd "$SRC_BUILD_DIR"
    PKG_VERSION=$(version_from_git)
    cd $WORKSPACE/packages/$TARGET
    fpm_build $SRC_BUILD_DIR/=/usr/local/arvados/src arvados-src 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--exclude=usr/local/arvados/src/.git" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

    rm -r "$SRC_BUILD_DIR"
)

# Keep
export GOPATH=$(mktemp -d)
mkdir -p "$GOPATH/src/git.curoverse.com"
ln -sfn "$WORKSPACE" "$GOPATH/src/git.curoverse.com/arvados.git"

# keepstore
cd "$GOPATH/src/git.curoverse.com/arvados.git/services/keepstore"
PKG_VERSION=$(version_from_git)
go get "git.curoverse.com/arvados.git/services/keepstore"
cd $WORKSPACE/packages/$TARGET
fpm_build $GOPATH/bin/keepstore=/usr/bin/keepstore keepstore 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Keepstore is the Keep storage daemon, accessible to clients on the LAN"

# Get GO SDK version
cd "$GOPATH/src/git.curoverse.com/arvados.git/sdk/go"
GO_SDK_VERSION=$(version_from_git)
GO_SDK_TIMESTAMP=$(timestamp_from_git)

# keepproxy
cd "$GOPATH/src/git.curoverse.com/arvados.git/services/keepproxy"
KEEPPROXY_VERSION=$(version_from_git)
KEEPPROXY_TIMESTAMP=$(timestamp_from_git)

if [[ "$GO_SDK_TIMESTAMP" -gt "$KEEPPROXY_TIMESTAMP" ]]; then
  PKG_VERSION=$GO_SDK_VERSION
else
  PKG_VERSION=$KEEPPROXY_VERSION
fi

go get "git.curoverse.com/arvados.git/services/keepproxy"
cd $WORKSPACE/packages/$TARGET
fpm_build $GOPATH/bin/keepproxy=/usr/bin/keepproxy keepproxy 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Keepproxy makes a Keep cluster accessible to clients that are not on the LAN"

# datamanager
cd "$GOPATH/src/git.curoverse.com/arvados.git/services/datamanager"
DATAMANAGER_VERSION=$(version_from_git)
DATAMANAGER_TIMESTAMP=$(timestamp_from_git)

if [[ "$GO_SDK_TIMESTAMP" -gt "$DATAMANAGER_TIMESTAMP" ]]; then
  PKG_VERSION=$GO_SDK_VERSION
else
  PKG_VERSION=$DATAMANAGER_VERSION
fi

go get "git.curoverse.com/arvados.git/services/datamanager"
cd $WORKSPACE/packages/$TARGET
fpm_build $GOPATH/bin/datamanager=/usr/bin/arvados-data-manager arvados-data-manager 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Datamanager ensures block replication levels, reports on disk usage and determines which blocks should be deleted when space is needed."

# arv-git-httpd
cd "$GOPATH/src/git.curoverse.com/arvados.git/services/arv-git-httpd"
ARVGITHTTPD_VERSION=$(version_from_git)
ARVGITHTTPD_TIMESTAMP=$(timestamp_from_git)

if [[ "$GO_SDK_TIMESTAMP" -gt "$ARVGITHTTPD_TIMESTAMP" ]]; then
  PKG_VERSION=$GO_SDK_VERSION
else
  PKG_VERSION=$ARVGITHTTPD_VERSION
fi

go get "git.curoverse.com/arvados.git/services/arv-git-httpd"
cd $WORKSPACE/packages/$TARGET
fpm_build $GOPATH/bin/arv-git-httpd=/usr/bin/arvados-git-httpd arvados-git-httpd 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Provides authenticated http access to Arvados-hosted git repositories."

# crunchstat
cd "$GOPATH/src/git.curoverse.com/arvados.git/services/crunchstat"
PKG_VERSION=$(version_from_git)
go get "git.curoverse.com/arvados.git/services/crunchstat"
cd $WORKSPACE/packages/$TARGET
fpm_build $GOPATH/bin/crunchstat=/usr/bin/crunchstat crunchstat 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=Crunchstat gathers cpu/memory/network statistics of running Crunch jobs"

# The Python SDK
# Please resist the temptation to add --no-python-fix-name to the fpm call here
# (which would remove the python- prefix from the package name), because this
# package is a dependency of arvados-fuse, and fpm can not omit the python-
# prefix from only one of the dependencies of a package...  Maybe I could
# whip up a patch and send it upstream, but that will be for another day. Ward,
# 2014-05-15
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/sdk/python/build"
fpm_build $WORKSPACE/sdk/python "${PYTHON2_PKG_PREFIX}-arvados-python-client" 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/python/arvados_python_client.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Arvados Python SDK" --deb-recommends=git

# The PAM module
if [[ $TARGET =~ debian|ubuntu ]]; then
    cd $WORKSPACE/packages/$TARGET
    rm -rf "$WORKSPACE/sdk/pam/build"
    fpm_build $WORKSPACE/sdk/pam libpam-arvados 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/pam/arvados_pam.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=PAM module for authenticating shell logins using Arvados API tokens" --depends libpam-python
fi

# The FUSE driver
# Please see comment about --no-python-fix-name above; we stay consistent and do
# not omit the python- prefix first.
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/services/fuse/build"
fpm_build $WORKSPACE/services/fuse "${PYTHON2_PKG_PREFIX}-arvados-fuse" 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/fuse/arvados_fuse.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Keep FUSE driver"

# The node manager
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/services/nodemanager/build"
fpm_build $WORKSPACE/services/nodemanager arvados-node-manager 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/nodemanager/arvados_node_manager.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Arvados node manager"

# The Docker image cleaner
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/services/dockercleaner/build"
fpm_build $WORKSPACE/services/dockercleaner arvados-docker-cleaner 'Curoverse, Inc.' 'python3' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/dockercleaner/arvados_docker_cleaner.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Arvados Docker image cleaner"

# A few dependencies
for deppkg in "${PYTHON_BACKPORTS[@]}"; do
    outname=$(echo "$deppkg" | sed -e 's/^python-//' -e 's/[<=>].*//' -e 's/_/-/g' -e "s/^/${PYTHON2_PKG_PREFIX}-/")
    fpm_build "$deppkg" "$outname"
done

# Python 3 dependencies
for deppkg in "${PYTHON3_BACKPORTS[@]}"; do
    outname=$(echo "$deppkg" | sed -e 's/^python-//' -e 's/[<=>].*//' -e 's/_/-/g' -e "s/^/${PYTHON3_PKG_PREFIX}-/")
    # The empty string is the vendor argument: these aren't Curoverse software.
    fpm_build "$deppkg" "$outname" "" python3
done

# Build the API server package

cd "$WORKSPACE/services/api"

API_VERSION=$(version_from_git)
PACKAGE_NAME=arvados-api-server

if [[ ! -d "$WORKSPACE/services/api/tmp" ]]; then
  mkdir $WORKSPACE/services/api/tmp
fi


bundle install --path vendor/bundle >"$STDOUT_IF_DEBUG"

/usr/bin/git rev-parse HEAD > git-commit.version

cd $WORKSPACE/packages/$TARGET

# Annoyingly, we require a database.yml file for rake assets:precompile to work. So for now,
# we do that in the upgrade script.
# TODO: add bogus database.yml file so we can precompile the assets and put them in the
# package. Then remove that database.yml file again. It has to be a valid file though.
#RAILS_ENV=production RAILS_GROUPS=assets bundle exec rake assets:precompile

# This is the complete package with vendor/bundle included.
# It's big, so we do not build it by default.
if [[ "$BUILD_BUNDLE_PACKAGES" != 0 ]]; then
  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "--vendor='Curoverse, Inc.'" "--url='https://arvados.org'" "--description='Arvados API server - Arvados is a free and open source platform for big data science.'" "--license='GNU Affero General Public License, version 3.0'" "-s" "dir" "-t" "$FORMAT" "-n" "${PACKAGE_NAME}-with-bundle" "-v" "$API_VERSION" "-x" "var/www/arvados-api/current/tmp" "-x" "var/www/arvados-api/current/log" "-x" "var/www/arvados-api/current/vendor/cache/*" "-x" "var/www/arvados-api/current/coverage" "-x" "var/www/arvados-api/current/Capfile*" "-x" "var/www/arvados-api/current/config/deploy*" "--after-install=$RUN_BUILD_PACKAGES_PATH/arvados-api-server-extras/postinst.sh" "$WORKSPACE/services/api/=/var/www/arvados-api/current" "$RUN_BUILD_PACKAGES_PATH/arvados-api-server-extras/arvados-api-server-upgrade.sh=/usr/local/bin/arvados-api-server-upgrade.sh")

  debug_echo -e "\n${COMMAND_ARR[@]}\n"

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?
  fpm_verify $FPM_EXIT_CODE $FPM_RESULTS
fi

# Build the 'bare' package without vendor/bundle.
declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "--vendor='Curoverse, Inc.'" "--url='https://arvados.org'" "--description='Arvados API server - Arvados is a free and open source platform for big data science.'" "--license='GNU Affero General Public License, version 3.0'" "-s" "dir" "-t" "$FORMAT" "-n" "${PACKAGE_NAME}" "-v" "$API_VERSION" "-x" "var/www/arvados-api/current/tmp" "-x" "var/www/arvados-api/current/log" "-x" "var/www/arvados-api/current/vendor/bundle" "-x" "var/www/arvados-api/current/vendor/cache/*" "-x" "var/www/arvados-api/current/coverage" "-x" "var/www/arvados-api/current/Capfile*" "-x" "var/www/arvados-api/current/config/deploy*" "--after-install=$RUN_BUILD_PACKAGES_PATH/arvados-api-server-extras/postinst.sh" "$WORKSPACE/services/api/=/var/www/arvados-api/current" "$RUN_BUILD_PACKAGES_PATH/arvados-api-server-extras/arvados-api-server-upgrade.sh=/usr/local/bin/arvados-api-server-upgrade.sh")

debug_echo -e "\n${COMMAND_ARR[@]}\n"

FPM_RESULTS=$("${COMMAND_ARR[@]}")
FPM_EXIT_CODE=$?
fpm_verify $FPM_EXIT_CODE $FPM_RESULTS

# API server package build done

# Build the workbench server package

cd "$WORKSPACE/apps/workbench"

WORKBENCH_VERSION=$(version_from_git)
PACKAGE_NAME=arvados-workbench

if [[ ! -d "$WORKSPACE/apps/workbench/tmp" ]]; then
  mkdir $WORKSPACE/apps/workbench/tmp
fi

bundle install --path vendor/bundle >"$STDOUT_IF_DEBUG"

/usr/bin/git rev-parse HEAD > git-commit.version

# clear the tmp directory; the asset generation step will recreate tmp/cache/assets,
# and we want that in the package, so it's easier to not exclude the tmp directory
# from the package - empty it instead.
rm -rf $WORKSPACE/apps/workbench/tmp/*

# Set up application.yml and production.rb so that asset precompilation works
\cp config/application.yml.example config/application.yml -f
\cp config/environments/production.rb.example config/environments/production.rb -f
sed -i 's/secret_token: ~/secret_token: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/' config/application.yml

RAILS_ENV=production RAILS_GROUPS=assets bundle exec rake assets:precompile >/dev/null

if [[ "$?" != "0" ]]; then
  echo "ERROR: Asset precompilation failed"
  EXITCODE=1
fi

cd $WORKSPACE/packages/$TARGET

# This is the complete package with vendor/bundle included.
# It's big, so we do not build it by default.
if [[ "$BUILD_BUNDLE_PACKAGES" != 0 ]]; then

  declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "--vendor='Curoverse, Inc.'" "--url='https://arvados.org'" "--description='Arvados Workbench - Arvados is a free and open source platform for big data science.'" "--license='GNU Affero General Public License, version 3.0'" "-s" "dir" "-t" "$FORMAT" "-n" "${PACKAGE_NAME}-with-bundle" "-v" "$WORKBENCH_VERSION" "-x" "var/www/arvados-workbench/current/log" "-x" "var/www/arvados-workbench/current/vendor/cache/*" "-x" "var/www/arvados-workbench/current/coverage" "-x" "var/www/arvados-workbench/current/Capfile*" "-x" "var/www/arvados-workbench/current/config/deploy*" "--after-install=$RUN_BUILD_PACKAGES_PATH/arvados-workbench-extras/postinst.sh" "$WORKSPACE/apps/workbench/=/var/www/arvados-workbench/current" "$RUN_BUILD_PACKAGES_PATH/arvados-workbench-extras/arvados-workbench-upgrade.sh=/usr/local/bin/arvados-workbench-upgrade.sh")

  debug_echo -e "\n${COMMAND_ARR[@]}\n"

  FPM_RESULTS=$("${COMMAND_ARR[@]}")
  FPM_EXIT_CODE=$?
  fpm_verify $FPM_EXIT_CODE $FPM_RESULTS
fi

# Build the 'bare' package without vendor/bundle.

declare -a COMMAND_ARR=("fpm" "--maintainer=Ward Vandewege <ward@curoverse.com>" "--vendor='Curoverse, Inc.'" "--url='https://arvados.org'" "--description='Arvados Workbench - Arvados is a free and open source platform for big data science.'" "--license='GNU Affero General Public License, version 3.0'" "-s" "dir" "-t" "$FORMAT" "-n" "${PACKAGE_NAME}" "-v" "$WORKBENCH_VERSION" "-x" "var/www/arvados-workbench/current/log" "-x" "var/www/arvados-workbench/current/vendor/bundle" "-x" "var/www/arvados-workbench/current/vendor/cache/*" "-x" "var/www/arvados-workbench/current/coverage" "-x" "var/www/arvados-workbench/current/Capfile*" "-x" "var/www/arvados-workbench/current/config/deploy*" "--after-install=$RUN_BUILD_PACKAGES_PATH/arvados-workbench-extras/postinst.sh" "$WORKSPACE/apps/workbench/=/var/www/arvados-workbench/current" "$RUN_BUILD_PACKAGES_PATH/arvados-workbench-extras/arvados-workbench-upgrade.sh=/usr/local/bin/arvados-workbench-upgrade.sh")

debug_echo -e "\n${COMMAND_ARR[@]}\n"

FPM_RESULTS=$("${COMMAND_ARR[@]}")
FPM_EXIT_CODE=$?
fpm_verify $FPM_EXIT_CODE $FPM_RESULTS

# Workbench package build done
# clean up temporary GOPATH
rm -rf "$GOPATH"

exit $EXITCODE
