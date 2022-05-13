#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

. "$(dirname "$(readlink -f "$0")")"/run-library.sh || exit 1

read -rd "\000" helpmessage <<EOF
$(basename "$0"): Build Arvados packages

Syntax:
        WORKSPACE=/path/to/arvados $(basename "$0") [options]

Options:

--build-bundle-packages  (default: false)
    Build api server and workbench packages with vendor/bundle included
--debug
    Output debug information (default: false)
--target <target>
    Distribution to build packages for (default: debian10)
--only-build <package>
    Build only a specific package (or ONLY_BUILD from environment)
--arch <arch>
    Build a specific architecture (or ARCH from environment, defaults to native architecture)
--force-build
    Build even if the package exists upstream or if it has already been
    built locally
--command
    Build command to execute (defaults to the run command defined in the
    Docker image)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

# Begin of user configuration

# set to --no-cache-dir to disable pip caching
CACHE_FLAG=

MAINTAINER="Arvados Package Maintainers <packaging@arvados.org>"
VENDOR="The Arvados Project"

# End of user configuration

DEBUG=${ARVADOS_DEBUG:-0}
FORCE_BUILD=${FORCE_BUILD:-0}
EXITCODE=0
TARGET=debian10
COMMAND=

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,build-bundle-packages,debug,target:,only-build:,arch:,force-build \
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
        --only-build)
            ONLY_BUILD="$2"; shift
            ;;
        --force-build)
            FORCE_BUILD=1
            ;;
        --arch)
            ARCH="$2"; shift
            ;;
        --debug)
            DEBUG=1
            ;;
        --command)
            COMMAND="$2"; shift
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

if [[ "$COMMAND" != "" ]]; then
  COMMAND="/usr/local/rvm/bin/rvm-exec default bash /jenkins/$COMMAND --target $TARGET"
fi

STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
if [[ "$DEBUG" != 0 ]]; then
    STDOUT_IF_DEBUG=/dev/stdout
    STDERR_IF_DEBUG=/dev/stderr
    DASHQ_UNLESS_DEBUG=
fi

declare -a PYTHON3_BACKPORTS

PYTHON3_EXECUTABLE=python3
PYTHON3_VERSION=$($PYTHON3_EXECUTABLE -c 'import sys; print("{v.major}.{v.minor}".format(v=sys.version_info))')

## These defaults are suitable for any Debian-based distribution.
# You can customize them as needed in distro sections below.
PYTHON3_PACKAGE=python$PYTHON3_VERSION
PYTHON3_PKG_PREFIX=python3
PYTHON3_PREFIX=/usr
PYTHON3_INSTALL_LIB=lib/python$PYTHON3_VERSION/dist-packages
## End Debian Python defaults.

case "$TARGET" in
    debian*)
        FORMAT=deb
        ;;
    ubuntu1804)
        FORMAT=deb
        PYTHON3_EXECUTABLE=python3.8
        PYTHON3_VERSION=$($PYTHON3_EXECUTABLE -c 'import sys; print("{v.major}.{v.minor}".format(v=sys.version_info))')
        PYTHON3_PACKAGE=python$PYTHON3_VERSION
        PYTHON3_INSTALL_LIB=lib/python$PYTHON3_VERSION/dist-packages
        ;;
    ubuntu*)
        FORMAT=deb
        ;;
    centos*)
        FORMAT=rpm
        PYTHON3_PACKAGE=$(rpm -qf "$(which python"$PYTHON3_VERSION")" --queryformat '%{NAME}\n')
        PYTHON3_PKG_PREFIX=$PYTHON3_PACKAGE
        PYTHON3_PREFIX=/usr
        PYTHON3_INSTALL_LIB=lib/python$PYTHON3_VERSION/site-packages
        export PYCURL_SSL_LIBRARY=nss
        ;;
    *)
        echo -e "$0: Unknown target '$TARGET'.\n" >&2
        exit 1
        ;;
esac


if [[ -z "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

# Test for fpm
fpm --version >/dev/null 2>&1

if [[ $? -ne 0 ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: fpm not found"
  echo >&2
  exit 1
fi

RUN_BUILD_PACKAGES_PATH="$(dirname "$0")"
RUN_BUILD_PACKAGES_PATH="$(cd "$RUN_BUILD_PACKAGES_PATH" && pwd)"  # absolutized and normalized
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

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# More cleanup - make sure all executables that we'll package are 755
cd "$WORKSPACE" || exit 1
find . -type d -name 'bin' -print0 |xargs -0 -I {} find {} -type f -print0 |xargs -0 -I {} chmod 755 {}

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

debug_echo "umask is" "$(umask)"

if [[ ! -d "$WORKSPACE/packages/$TARGET" ]]; then
  mkdir -p "$WORKSPACE/packages/$TARGET"
  chown --reference="$WORKSPACE" "$WORKSPACE/packages/$TARGET"
fi

# Required due to CVE-2022-24765
git config --global --add safe.directory /arvados

# Perl packages
debug_echo -e "\nPerl packages\n"

handle_libarvados_perl

# Ruby gems
debug_echo -e "\nRuby gems\n"

FPM_GEM_PREFIX=$($GEM environment gemdir)

cd "$WORKSPACE/sdk/ruby" || exit 1
handle_ruby_gem arvados

cd "$WORKSPACE/sdk/cli" || exit 1
handle_ruby_gem arvados-cli

cd "$WORKSPACE/services/login-sync" || exit 1
handle_ruby_gem arvados-login-sync

# arvados-src
handle_arvados_src

# Go packages
debug_echo -e "\nGo packages\n"

# Go binaries
export GOPATH=~/go
package_go_binary cmd/arvados-client arvados-client "$FORMAT" "$ARCH" \
    "Arvados command line tool (beta)"
package_go_binary cmd/arvados-server arvados-server "$FORMAT" "$ARCH" \
    "Arvados server daemons"
package_go_binary cmd/arvados-server arvados-controller "$FORMAT" "$ARCH" \
    "Arvados cluster controller daemon"
package_go_binary cmd/arvados-server arvados-dispatch-cloud "$FORMAT" "$ARCH" \
    "Arvados cluster cloud dispatch"
package_go_binary cmd/arvados-server arvados-dispatch-lsf "$FORMAT" "$ARCH" \
    "Dispatch Arvados containers to an LSF cluster"
package_go_binary cmd/arvados-server arvados-git-httpd "$FORMAT" "$ARCH" \
    "Provide authenticated http access to Arvados-hosted git repositories"
package_go_binary services/crunch-dispatch-local crunch-dispatch-local "$FORMAT" "$ARCH" \
    "Dispatch Crunch containers on the local system"
package_go_binary services/crunch-dispatch-slurm crunch-dispatch-slurm "$FORMAT" "$ARCH" \
    "Dispatch Crunch containers to a SLURM cluster"
package_go_binary cmd/arvados-server crunch-run "$FORMAT" "$ARCH" \
    "Supervise a single Crunch container"
package_go_binary services/crunchstat crunchstat "$FORMAT" "$ARCH" \
    "Gather cpu/memory/network statistics of running Crunch jobs"
package_go_binary cmd/arvados-server arvados-health "$FORMAT" "$ARCH" \
    "Check health of all Arvados cluster services"
package_go_binary cmd/arvados-server keep-balance "$FORMAT" "$ARCH" \
    "Rebalance and garbage-collect data blocks stored in Arvados Keep"
package_go_binary cmd/arvados-server keepproxy "$FORMAT" "$ARCH" \
    "Make a Keep cluster accessible to clients that are not on the LAN"
package_go_binary cmd/arvados-server keepstore "$FORMAT" "$ARCH" \
    "Keep storage daemon, accessible to clients on the LAN"
package_go_binary cmd/arvados-server keep-web "$FORMAT" "$ARCH" \
    "Static web hosting service for user data stored in Arvados Keep"
package_go_binary cmd/arvados-server arvados-ws "$FORMAT" "$ARCH" \
    "Arvados Websocket server"
package_go_binary tools/sync-groups arvados-sync-groups "$FORMAT" "$ARCH" \
    "Synchronize remote groups into Arvados from an external source"
package_go_binary tools/keep-block-check keep-block-check "$FORMAT" "$ARCH" \
    "Verify that all data from one set of Keep servers to another was copied"
package_go_binary tools/keep-rsync keep-rsync "$FORMAT" "$ARCH" \
    "Copy all data from one set of Keep servers to another"
package_go_binary tools/keep-exercise keep-exercise "$FORMAT" "$ARCH" \
    "Performance testing tool for Arvados Keep"
package_go_so lib/pam pam_arvados.so libpam-arvados-go "$FORMAT" "$ARCH" \
    "Arvados PAM authentication module"

# Python packages
debug_echo -e "\nPython packages\n"

# The Python SDK - Python3 package
fpm_build_virtualenv "arvados-python-client" "sdk/python" "$FORMAT" "$ARCH"

# Arvados cwl runner - Python3 package
fpm_build_virtualenv "arvados-cwl-runner" "sdk/cwl" "$FORMAT" "$ARCH"

# The FUSE driver - Python3 package
fpm_build_virtualenv "arvados-fuse" "services/fuse" "$FORMAT" "$ARCH"

# The Arvados crunchstat-summary tool
fpm_build_virtualenv "crunchstat-summary" "tools/crunchstat-summary" "$FORMAT" "$ARCH"

# The Docker image cleaner
fpm_build_virtualenv "arvados-docker-cleaner" "services/dockercleaner" "$FORMAT" "$ARCH"

# The Arvados user activity tool
fpm_build_virtualenv "arvados-user-activity" "tools/user-activity" "$FORMAT" "$ARCH"

# The python->python3 metapackages
build_metapackage "arvados-fuse" "services/fuse"
build_metapackage "arvados-python-client" "services/fuse"
build_metapackage "arvados-cwl-runner" "sdk/cwl"
build_metapackage "crunchstat-summary" "tools/crunchstat-summary"
build_metapackage "arvados-docker-cleaner" "services/dockercleaner"
build_metapackage "arvados-user-activity" "tools/user-activity"

# The cwltest package, which lives out of tree
handle_cwltest "$FORMAT" "$ARCH"

# Rails packages
debug_echo -e "\nRails packages\n"

# The rails api server package
handle_api_server "$ARCH"
# The rails workbench package
handle_workbench "$ARCH"

# clean up temporary GOPATH
rm -rf "$GOPATH"

exit $EXITCODE
