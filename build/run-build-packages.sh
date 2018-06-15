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

EXITCODE=0
DEBUG=${ARVADOS_DEBUG:-0}
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
    debian8)
        FORMAT=deb
        ;;
    debian9)
        FORMAT=deb
        ;;
    ubuntu1404)
        FORMAT=deb
        ;;
    ubuntu1604)
        FORMAT=deb
        ;;
    centos7)
        FORMAT=rpm
        PYTHON2_PACKAGE=$(rpm -qf "$(which python$PYTHON2_VERSION)" --queryformat '%{NAME}\n')
        PYTHON2_PKG_PREFIX=$PYTHON2_PACKAGE
        PYTHON2_INSTALL_LIB=lib/python$PYTHON2_VERSION/site-packages
        PYTHON3_PACKAGE=$(rpm -qf "$(which python$PYTHON3_VERSION)" --queryformat '%{NAME}\n')
        PYTHON3_PKG_PREFIX=$PYTHON3_PACKAGE
        PYTHON3_PREFIX=/opt/rh/python33/root/usr
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
        "Curoverse, Inc." dir "$(version_from_git)" install/man/=/usr/share/man \
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

cd "$WORKSPACE/sdk/pam"
handle_python_package

cd "$WORKSPACE/sdk/python"
handle_python_package

cd "$WORKSPACE/sdk/cwl"
handle_python_package

cd "$WORKSPACE/services/fuse"
handle_python_package

cd "$WORKSPACE/services/nodemanager"
handle_python_package

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
      fpm_build $SRC_BUILD_DIR/=/usr/local/arvados/src arvados-src 'Curoverse, Inc.' 'dir' "$PKG_VERSION" "--exclude=usr/local/arvados/src/.git" "--url=https://arvados.org" "--license=GNU Affero General Public License, version 3.0" "--description=The Arvados source code" "--architecture=all"

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
# Please resist the temptation to add --no-python-fix-name to the fpm call here
# (which would remove the python- prefix from the package name), because this
# package is a dependency of arvados-fuse, and fpm can not omit the python-
# prefix from only one of the dependencies of a package...  Maybe I could
# whip up a patch and send it upstream, but that will be for another day. Ward,
# 2014-05-15
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/sdk/python/build"
arvados_python_client_version=${ARVADOS_BUILDING_VERSION:-$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/python/arvados_python_client.egg-info/PKG-INFO)}
test_package_presence ${PYTHON2_PKG_PREFIX}-arvados-python-client "$arvados_python_client_version" python
if [[ "$?" == "0" ]]; then
  fpm_build $WORKSPACE/sdk/python "${PYTHON2_PKG_PREFIX}-arvados-python-client" 'Curoverse, Inc.' 'python' "$arvados_python_client_version" "--url=https://arvados.org" "--description=The Arvados Python SDK" --depends "${PYTHON2_PKG_PREFIX}-setuptools" --deb-recommends=git
fi

# cwl-runner
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/sdk/cwl/build"
arvados_cwl_runner_version=${ARVADOS_BUILDING_VERSION:-$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/cwl/arvados_cwl_runner.egg-info/PKG-INFO)}
declare -a iterargs=()
if [[ -z "$ARVADOS_BUILDING_VERSION" ]]; then
    arvados_cwl_runner_iteration=4
    iterargs+=(--iteration $arvados_cwl_runner_iteration)
else
    arvados_cwl_runner_iteration=
fi
test_package_presence ${PYTHON2_PKG_PREFIX}-arvados-cwl-runner "$arvados_cwl_runner_version" python "$arvados_cwl_runner_iteration"
if [[ "$?" == "0" ]]; then
  fpm_build $WORKSPACE/sdk/cwl "${PYTHON2_PKG_PREFIX}-arvados-cwl-runner" 'Curoverse, Inc.' 'python' "$arvados_cwl_runner_version" "--url=https://arvados.org" "--description=The Arvados CWL runner" --depends "${PYTHON2_PKG_PREFIX}-setuptools" --depends "${PYTHON2_PKG_PREFIX}-subprocess32 >= 3.5.0" --depends "${PYTHON2_PKG_PREFIX}-pathlib2" --depends "${PYTHON2_PKG_PREFIX}-scandir" "${iterargs[@]}"
fi

# schema_salad. This is a python dependency of arvados-cwl-runner,
# but we can't use the usual PYTHONPACKAGES way to build this package due to the
# intricacies of how version numbers get generated in setup.py: we need a specific version,
# e.g. 1.7.20160316203940. If we don't explicitly list that version with the -v
# argument to fpm, and instead specify it as schema_salad==1.7.20160316203940, we get
# a package with version 1.7. That's because our gittagger hack is not being
# picked up by self.distribution.get_version(), which is called from
# https://github.com/jordansissel/fpm/blob/master/lib/fpm/package/pyfpm/get_metadata.py
# by means of this command:
#
# python2.7 setup.py --command-packages=pyfpm get_metadata --output=metadata.json
#
# So we build this thing separately.
#
# Ward, 2016-03-17
saladversion=$(cat "$WORKSPACE/sdk/cwl/setup.py" | grep schema-salad== | sed "s/.*==\(.*\)'.*/\1/")
test_package_presence python-schema-salad "$saladversion" python 2
if [[ "$?" == "0" ]]; then
  fpm_build schema_salad "" "" python $saladversion --depends "${PYTHON2_PKG_PREFIX}-lockfile >= 1:0.12.2-2" --depends "${PYTHON2_PKG_PREFIX}-avro = 1.8.1-2" --iteration 2
fi

# And for cwltool we have the same problem as for schema_salad. Ward, 2016-03-17
cwltoolversion=$(cat "$WORKSPACE/sdk/cwl/setup.py" | grep cwltool== | sed "s/.*==\(.*\)'.*/\1/")
test_package_presence python-cwltool "$cwltoolversion" python 2
if [[ "$?" == "0" ]]; then
  fpm_build cwltool "" "" python $cwltoolversion --iteration 2
fi

# The PAM module
if [[ $TARGET =~ debian|ubuntu ]]; then
    cd $WORKSPACE/packages/$TARGET
    rm -rf "$WORKSPACE/sdk/pam/build"
    libpam_arvados_version=$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/pam/arvados_pam.egg-info/PKG-INFO)
    test_package_presence libpam-arvados "$libpam_arvados_version" python
    if [[ "$?" == "0" ]]; then
      fpm_build $WORKSPACE/sdk/pam libpam-arvados 'Curoverse, Inc.' 'python' "$libpam_arvados_version" "--url=https://arvados.org" "--description=PAM module for authenticating shell logins using Arvados API tokens" --depends libpam-python
    fi
fi

# The FUSE driver
# Please see comment about --no-python-fix-name above; we stay consistent and do
# not omit the python- prefix first.
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/services/fuse/build"
arvados_fuse_version=${ARVADOS_BUILDING_VERSION:-$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/fuse/arvados_fuse.egg-info/PKG-INFO)}
test_package_presence "${PYTHON2_PKG_PREFIX}-arvados-fuse" "$arvados_fuse_version" python
if [[ "$?" == "0" ]]; then
  fpm_build $WORKSPACE/services/fuse "${PYTHON2_PKG_PREFIX}-arvados-fuse" 'Curoverse, Inc.' 'python' "$arvados_fuse_version" "--url=https://arvados.org" "--description=The Keep FUSE driver" --depends "${PYTHON2_PKG_PREFIX}-setuptools"
fi

# The node manager
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/services/nodemanager/build"
nodemanager_version=${ARVADOS_BUILDING_VERSION:-$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/nodemanager/arvados_node_manager.egg-info/PKG-INFO)}
test_package_presence arvados-node-manager "$nodemanager_version" python
if [[ "$?" == "0" ]]; then
  fpm_build $WORKSPACE/services/nodemanager arvados-node-manager 'Curoverse, Inc.' 'python' "$nodemanager_version" "--url=https://arvados.org" "--description=The Arvados node manager" --depends "${PYTHON2_PKG_PREFIX}-setuptools"
fi

# The Docker image cleaner
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/services/dockercleaner/build"
dockercleaner_version=${ARVADOS_BUILDING_VERSION:-$(awk '($1 == "Version:"){print $2}' $WORKSPACE/services/dockercleaner/arvados_docker_cleaner.egg-info/PKG-INFO)}
iteration="${ARVADOS_BUILDING_ITERATION:-3}"
test_package_presence arvados-docker-cleaner "$dockercleaner_version" python "$iteration"
if [[ "$?" == "0" ]]; then
  fpm_build $WORKSPACE/services/dockercleaner arvados-docker-cleaner 'Curoverse, Inc.' 'python3' "$dockercleaner_version" "--url=https://arvados.org" "--description=The Arvados Docker image cleaner" --depends "${PYTHON3_PKG_PREFIX}-websocket-client = 0.37.0" --iteration "$iteration"
fi

# The Arvados crunchstat-summary tool
cd $WORKSPACE/packages/$TARGET
crunchstat_summary_version=${ARVADOS_BUILDING_VERSION:-$(awk '($1 == "Version:"){print $2}' $WORKSPACE/tools/crunchstat-summary/crunchstat_summary.egg-info/PKG-INFO)}
iteration="${ARVADOS_BUILDING_ITERATION:-2}"
test_package_presence "$PYTHON2_PKG_PREFIX"-crunchstat-summary "$crunchstat_summary_version" python "$iteration"
if [[ "$?" == "0" ]]; then
  rm -rf "$WORKSPACE/tools/crunchstat-summary/build"
  fpm_build $WORKSPACE/tools/crunchstat-summary ${PYTHON2_PKG_PREFIX}-crunchstat-summary 'Curoverse, Inc.' 'python' "$crunchstat_summary_version" "--url=https://arvados.org" "--description=Crunchstat-summary reads Arvados Crunch log files and summarize resource usage" --iteration "$iteration"
fi

# Forked libcloud
if test_package_presence "$PYTHON2_PKG_PREFIX"-apache-libcloud "$LIBCLOUD_PIN" python 2
then
  LIBCLOUD_DIR=$(mktemp -d)
  (
      cd $LIBCLOUD_DIR
      git clone $DASHQ_UNLESS_DEBUG https://github.com/curoverse/libcloud.git .
      git checkout $DASHQ_UNLESS_DEBUG apache-libcloud-$LIBCLOUD_PIN
      # libcloud is absurdly noisy without -q, so force -q here
      OLD_DASHQ_UNLESS_DEBUG=$DASHQ_UNLESS_DEBUG
      DASHQ_UNLESS_DEBUG=-q
      handle_python_package
      DASHQ_UNLESS_DEBUG=$OLD_DASHQ_UNLESS_DEBUG
  )

  # libcloud >= 2.3.0 now requires python-requests 2.4.3 or higher, otherwise
  # it throws
  #   ImportError: No module named packages.urllib3.poolmanager
  # when loaded. We only see this problem on ubuntu1404, because that is our
  # only supported distribution that ships with a python-requests older than
  # 2.4.3.
  fpm_build $LIBCLOUD_DIR "$PYTHON2_PKG_PREFIX"-apache-libcloud "" python "" --iteration 2 --depends 'python-requests >= 2.4.3'
  rm -rf $LIBCLOUD_DIR
fi

# Python 2 dependencies
declare -a PIP_DOWNLOAD_SWITCHES=(--no-deps)
# Add --no-use-wheel if this pip knows it.
pip wheel --help >/dev/null 2>&1
case "$?" in
    0) PIP_DOWNLOAD_SWITCHES+=(--no-use-wheel) ;;
    2) ;;
    *) echo "WARNING: `pip wheel` test returned unknown exit code $?" ;;
esac

while read -r line || [[ -n "$line" ]]; do
#  echo "Text read from file: $line"
  if [[ "$line" =~ ^# ]]; then
    continue
  fi
  IFS='|'; arr=($line); unset IFS

  dist=${arr[0]}

  IFS=',';dists=($dist); unset IFS

  MATCH=0
  for d in "${dists[@]}"; do
    if [[ "$d" == "$TARGET" ]] || [[ "$d" == "all" ]]; then
      MATCH=1
    fi
  done

  if [[ "$MATCH" != "1" ]]; then
    continue
  fi
  name=${arr[1]}
  version=${arr[2]}
  iteration=${arr[3]}
  pkgtype=${arr[4]}
  arch=${arr[5]}
  extra=${arr[6]}
  declare -a 'extra_arr=('"$extra"')'

  if [[ "$FORMAT" == "rpm" ]]; then
    if [[ "$arch" == "all" ]]; then
      arch="noarch"
    fi
    if [[ "$arch" == "amd64" ]]; then
      arch="x86_64"
    fi
  fi

  if [[ "$pkgtype" == "python" ]]; then
    outname=$(echo "$name" | sed -e 's/^python-//' -e 's/_/-/g' -e "s/^/${PYTHON2_PKG_PREFIX}-/")
  else
    outname=$(echo "$name" | sed -e 's/^python-//' -e 's/_/-/g' -e "s/^/${PYTHON3_PKG_PREFIX}-/")
  fi

  if [[ -n "$ONLY_BUILD" ]] && [[ "$outname" != "$ONLY_BUILD" ]] ; then
      continue
  fi

  case "$name" in
      httplib2|google-api-python-client)
          test_package_presence $outname $version $pkgtype $iteration $arch
          if [[ "$?" == "0" ]]; then
            # Work around 0640 permissions on some package files.
            # See #7591 and #7991.
            pyfpm_workdir=$(mktemp --tmpdir -d pyfpm-XXXXXX) && (
                set -e
                cd "$pyfpm_workdir"
                pip install "${PIP_DOWNLOAD_SWITCHES[@]}" --download . "$name==$version"
                # Sometimes pip gives us a tarball, sometimes a zip file...
                DOWNLOADED=`ls $name-*`
                [[ "$DOWNLOADED" =~ ".tar" ]] && tar -xf $DOWNLOADED
                [[ "$DOWNLOADED" =~ ".zip" ]] && unzip $DOWNLOADED
                cd "$name"-*/
                "python$PYTHON2_VERSION" setup.py $DASHQ_UNLESS_DEBUG egg_info build
                chmod -R go+rX .
                set +e
                fpm_build . "$outname" "" "$pkgtype" "$version" --iteration "$iteration" "${extra_arr[@]}"
                # The upload step uses the package timestamp to determine
                # if it is new.  --no-clobber plays nice with that.
                mv --no-clobber "$outname"*.$FORMAT "$WORKSPACE/packages/$TARGET"
            )
            if [ 0 != "$?" ]; then
                echo "ERROR: $name build process failed"
                EXITCODE=1
            fi
            if [ -n "$pyfpm_workdir" ]; then
                rm -rf "$pyfpm_workdir"
            fi
          fi
          ;;
      *)
          test_package_presence $outname $version $pkgtype $iteration $arch
          if [[ "$?" == "0" ]]; then
            fpm_build "$name" "$outname" "" "$pkgtype" "$version" --iteration "$iteration" "${extra_arr[@]}"
          fi
          ;;
  esac

done <`dirname "$(readlink -f "$0")"`"/build.list"

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
