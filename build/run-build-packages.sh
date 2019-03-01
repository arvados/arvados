#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

. `dirname "$(readlink -f "$0")"`/run-library.sh
. `dirname "$(readlink -f "$0")"`/libcloud-pin.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados packages

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

Options:

--build-bundle-packages  (default: false)
    Build api server and workbench packages with vendor/bundle included
--debug
    Output debug information (default: false)
--target <target>
    Distribution to build packages for (default: debian8)
--only-build <package>
    Build only a specific package (or $ONLY_BUILD from environment)
--command
    Build command to execute (defaults to the run command defined in the
    Docker image)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

# Begin of user configuration

# set to --no-cache-dir to disable pip caching
CACHE_FLAG=

MAINTAINER="Ward Vandewege <wvandewege@veritasgenetics.com>"
VENDOR="Veritas Genetics, Inc."

# End of user configuration

DEBUG=${ARVADOS_DEBUG:-0}
EXITCODE=0
TARGET=debian8
COMMAND=

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,build-bundle-packages,debug,target:,only-build: \
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

declare -a PYTHON_BACKPORTS PYTHON3_BACKPORTS

PYTHON2_VERSION=2.7
PYTHON3_VERSION=$(python3 -c 'import sys; print("{v.major}.{v.minor}".format(v=sys.version_info))')

## These defaults are suitable for any Debian-based distribution.
# You can customize them as needed in distro sections below.
PYTHON2_PACKAGE=python$PYTHON2_VERSION
PYTHON2_PKG_PREFIX=python
PYTHON2_PREFIX=/usr
PYTHON2_INSTALL_LIB=lib/python$PYTHON2_VERSION/dist-packages

PYTHON3_PACKAGE=python$PYTHON3_VERSION
PYTHON3_PKG_PREFIX=python3
PYTHON3_PREFIX=/usr
PYTHON3_INSTALL_LIB=lib/python$PYTHON3_VERSION/dist-packages
## End Debian Python defaults.

case "$TARGET" in
    debian*)
        FORMAT=deb
        ;;
    ubuntu*)
        FORMAT=deb
        ;;
    centos*)
        FORMAT=rpm
        PYTHON2_PACKAGE=$(rpm -qf "$(which python$PYTHON2_VERSION)" --queryformat '%{NAME}\n')
        PYTHON2_PKG_PREFIX=$PYTHON2_PACKAGE
        PYTHON2_INSTALL_LIB=lib/python$PYTHON2_VERSION/site-packages
        PYTHON3_PACKAGE=$(rpm -qf "$(which python$PYTHON3_VERSION)" --queryformat '%{NAME}\n')
        PYTHON3_PKG_PREFIX=$PYTHON3_PACKAGE
        PYTHON3_PREFIX=/opt/rh/rh-python35/root/usr
        PYTHON3_INSTALL_LIB=lib/python$PYTHON3_VERSION/site-packages
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

PYTHON2_FPM_INSTALLER=(--python-easyinstall "$(find_python_program easy_install-$PYTHON2_VERSION easy_install)")
install3=$(find_python_program easy_install-$PYTHON3_VERSION easy_install3 pip-$PYTHON3_VERSION pip3)
if [[ $install3 =~ easy_ ]]; then
    PYTHON3_FPM_INSTALLER=(--python-easyinstall "$install3")
else
    PYTHON3_FPM_INSTALLER=(--python-pip "$install3")
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

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# More cleanup - make sure all executables that we'll package are 755
cd "$WORKSPACE"
find -type d -name 'bin' |xargs -I {} find {} -type f |xargs -I {} chmod 755 {}

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

debug_echo "umask is" `umask`

if [[ ! -d "$WORKSPACE/packages/$TARGET" ]]; then
  mkdir -p $WORKSPACE/packages/$TARGET
  chown --reference="$WORKSPACE" "$WORKSPACE/packages/$TARGET"
fi

# Perl packages
debug_echo -e "\nPerl packages\n"

if [[ -z "$ONLY_BUILD" ]] || [[ "libarvados-perl" = "$ONLY_BUILD" ]] ; then
  cd "$WORKSPACE/sdk/perl"
  libarvados_perl_version="$(version_from_git)"

  cd $WORKSPACE/packages/$TARGET
  test_package_presence libarvados-perl "$libarvados_perl_version"

  if [[ "$?" == "0" ]]; then
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
        dir "$(version_from_git)" install/man/=/usr/share/man \
        "$WORKSPACE/apache-2.0.txt=/usr/share/doc/libarvados-perl/apache-2.0.txt" && \
        mv --no-clobber libarvados-perl*.$FORMAT "$WORKSPACE/packages/$TARGET/"
  fi
fi

# Ruby gems
debug_echo -e "\nRuby gems\n"

FPM_GEM_PREFIX=$($GEM environment gemdir)

cd "$WORKSPACE/sdk/ruby"
handle_ruby_gem arvados

cd "$WORKSPACE/sdk/cli"
handle_ruby_gem arvados-cli

cd "$WORKSPACE/services/login-sync"
handle_ruby_gem arvados-login-sync

# Python packages
debug_echo -e "\nPython packages\n"

# arvados-src
(
    cd "$WORKSPACE"
    COMMIT_HASH=$(format_last_commit_here "%H")
    arvados_src_version="$(version_from_git)"

    cd $WORKSPACE/packages/$TARGET
    test_package_presence arvados-src $arvados_src_version src ""

    if [[ "$?" == "0" ]]; then
      cd "$WORKSPACE"
      SRC_BUILD_DIR=$(mktemp -d)
      # mktemp creates the directory with 0700 permissions by default
      chmod 755 $SRC_BUILD_DIR
      git clone $DASHQ_UNLESS_DEBUG "$WORKSPACE/.git" "$SRC_BUILD_DIR"
      cd "$SRC_BUILD_DIR"

      # go into detached-head state
      git checkout $DASHQ_UNLESS_DEBUG "$COMMIT_HASH"
      echo "$COMMIT_HASH" >git-commit.version

      cd "$SRC_BUILD_DIR"
      PKG_VERSION=$(version_from_git)
      cd $WORKSPACE/packages/$TARGET
      fpm_build $SRC_BUILD_DIR/=/usr/local/arvados/src arvados-src 'dir' "$PKG_VERSION" "--exclude=usr/local/arvados/src/.git" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

      rm -rf "$SRC_BUILD_DIR"
    fi
)

# Go binaries
cd $WORKSPACE/packages/$TARGET
export GOPATH=$(mktemp -d)
go get github.com/kardianos/govendor
package_go_binary cmd/arvados-client arvados-client \
    "Arvados command line tool (beta)"
package_go_binary cmd/arvados-server arvados-server \
    "Arvados server daemons"
package_go_binary cmd/arvados-server arvados-controller \
    "Arvados cluster controller daemon"
package_go_binary cmd/arvados-server arvados-dispatch-cloud \
    "Arvados cluster cloud dispatch"
package_go_binary sdk/go/crunchrunner crunchrunner \
    "Crunchrunner executes a command inside a container and uploads the output"
package_go_binary services/arv-git-httpd arvados-git-httpd \
    "Provide authenticated http access to Arvados-hosted git repositories"
package_go_binary services/crunch-dispatch-local crunch-dispatch-local \
    "Dispatch Crunch containers on the local system"
package_go_binary services/crunch-dispatch-slurm crunch-dispatch-slurm \
    "Dispatch Crunch containers to a SLURM cluster"
package_go_binary services/crunch-run crunch-run \
    "Supervise a single Crunch container"
package_go_binary services/crunchstat crunchstat \
    "Gather cpu/memory/network statistics of running Crunch jobs"
package_go_binary services/health arvados-health \
    "Check health of all Arvados cluster services"
package_go_binary services/keep-balance keep-balance \
    "Rebalance and garbage-collect data blocks stored in Arvados Keep"
package_go_binary services/keepproxy keepproxy \
    "Make a Keep cluster accessible to clients that are not on the LAN"
package_go_binary services/keepstore keepstore \
    "Keep storage daemon, accessible to clients on the LAN"
package_go_binary services/keep-web keep-web \
    "Static web hosting service for user data stored in Arvados Keep"
package_go_binary services/ws arvados-ws \
    "Arvados Websocket server"
package_go_binary tools/sync-groups arvados-sync-groups \
    "Synchronize remote groups into Arvados from an external source"
package_go_binary tools/keep-block-check keep-block-check \
    "Verify that all data from one set of Keep servers to another was copied"
package_go_binary tools/keep-rsync keep-rsync \
    "Copy all data from one set of Keep servers to another"
package_go_binary tools/keep-exercise keep-exercise \
    "Performance testing tool for Arvados Keep"

# The Python SDK
fpm_build_virtualenv "arvados-python-client" "sdk/python"
fpm_build_virtualenv "arvados-python-client" "sdk/python" "python3"

# Arvados cwl runner
fpm_build_virtualenv "arvados-cwl-runner" "sdk/cwl"

# The PAM module
fpm_build_virtualenv "libpam-arvados" "sdk/pam"

# The FUSE driver
fpm_build_virtualenv "arvados-fuse" "services/fuse"

# The node manager
fpm_build_virtualenv "arvados-node-manager" "services/nodemanager"

# The Docker image cleaner
fpm_build_virtualenv "arvados-docker-cleaner" "services/dockercleaner" "python3"

# The Arvados crunchstat-summary tool
fpm_build_virtualenv "crunchstat-summary" "tools/crunchstat-summary"

# Build the API server package
test_rails_package_presence arvados-api-server "$WORKSPACE/services/api"
if [[ "$?" == "0" ]]; then
  handle_rails_package arvados-api-server "$WORKSPACE/services/api" \
      "$WORKSPACE/agpl-3.0.txt" --url="https://arvados.org" \
      --description="Arvados API server - Arvados is a free and open source platform for big data science." \
      --license="GNU Affero General Public License, version 3.0"
fi

# Build the workbench server package
test_rails_package_presence arvados-workbench "$WORKSPACE/apps/workbench"
if [[ "$?" == "0" ]] ; then
  (
      set -e
      cd "$WORKSPACE/apps/workbench"

      # We need to bundle to be ready even when we build a package without vendor directory
      # because asset compilation requires it.
      bundle install --system >"$STDOUT_IF_DEBUG"

      # clear the tmp directory; the asset generation step will recreate tmp/cache/assets,
      # and we want that in the package, so it's easier to not exclude the tmp directory
      # from the package - empty it instead.
      rm -rf tmp
      mkdir tmp

      # Set up application.yml and production.rb so that asset precompilation works
      \cp config/application.yml.example config/application.yml -f
      \cp config/environments/production.rb.example config/environments/production.rb -f
      sed -i 's/secret_token: ~/secret_token: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/' config/application.yml
      sed -i 's/keep_web_url: false/keep_web_url: exampledotcom/' config/application.yml

      RAILS_ENV=production RAILS_GROUPS=assets bundle exec rake npm:install >/dev/null
      RAILS_ENV=production RAILS_GROUPS=assets bundle exec rake assets:precompile >/dev/null

      # Remove generated configuration files so they don't go in the package.
      rm config/application.yml config/environments/production.rb
  )

  if [[ "$?" != "0" ]]; then
    echo "ERROR: Asset precompilation failed"
    EXITCODE=1
  else
    handle_rails_package arvados-workbench "$WORKSPACE/apps/workbench" \
        "$WORKSPACE/agpl-3.0.txt" --url="https://arvados.org" \
        --description="Arvados Workbench - Arvados is a free and open source platform for big data science." \
        --license="GNU Affero General Public License, version 3.0"
  fi
fi

# clean up temporary GOPATH
rm -rf "$GOPATH"

exit $EXITCODE
