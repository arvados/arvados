#!/bin/bash

. `dirname "$(readlink -f "$0")"`/run-library.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados SSO server package

Syntax:
        WORKSPACE=/path/to/arvados-sso $(basename $0) [options]

Options:

--build-bundle-packages  (default: false)
    Build package with vendor/bundle included
--debug
    Output debug information (default: false)
--target
    Distribution to build packages for (default: debian7)

WORKSPACE=path         Path to the Arvados SSO source tree to build packages from

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

case "$TARGET" in
    debian7)
        FORMAT=deb
        ;;
    debian8)
        FORMAT=deb
        ;;
    ubuntu1204)
        FORMAT=deb
        ;;
    ubuntu1404)
        FORMAT=deb
        ;;
    centos6)
        FORMAT=rpm
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

if ! [[ -d "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: $WORKSPACE is not a directory"
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

RUN_BUILD_PACKAGES_PATH="`dirname \"$0\"`"
RUN_BUILD_PACKAGES_PATH="`( cd \"$RUN_BUILD_PACKAGES_PATH\" && pwd )`"  # absolutized and normalized
if [ -z "$RUN_BUILD_PACKAGES_PATH" ] ; then
  # error; for some reason, the path is not accessible
  # to the script (e.g. permissions re-evaled after suid)
  exit 1  # fail
fi

debug_echo "$0 is running from $RUN_BUILD_PACKAGES_PATH"
debug_echo "Workspace is $WORKSPACE"

if [[ -f /etc/profile.d/rvm.sh ]]; then
    source /etc/profile.d/rvm.sh
    GEM="rvm-exec default gem"
else
    GEM=gem
fi

if [[ "$TARGET" == "centos6" ]]; then
  # CentOS6 comes with git 1.7.1, but we want at least 1.7.6
  # because we use git status --ignore in fpm-info.sh
  cd /usr/src
  install_package libcurl-devel zlib-devel wget gettext
  wget https://www.kernel.org/pub/software/scm/git/git-1.8.5.6.tar.gz
  tar xzf git-1.8.5.6.tar.gz
  cd git-1.8.5.6
  make configure
  ./configure --prefix=/usr --without-tcltk
  make all
  make install
fi

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# More cleanup - make sure all executables that we'll package are 755
# No executables in the sso server package
#find -type d -name 'bin' |xargs -I {} find {} -type f |xargs -I {} chmod 755 {}

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

debug_echo "umask is" `umask`

if [[ ! -d "$WORKSPACE/packages/$TARGET" ]]; then
  mkdir -p $WORKSPACE/packages/$TARGET
fi

# Build the SSO server package
handle_rails_package arvados-sso-server "$WORKSPACE" \
    "$WORKSPACE/LICENCE" --url="https://arvados.org" \
    --description="Arvados SSO server - Arvados is a free and open source platform for big data science." \
    --license="Expat license"

exit $EXITCODE
