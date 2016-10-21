#!/bin/bash

. `dirname "$(readlink -f "$0")"`/run-library.sh
. `dirname "$(readlink -f "$0")"`/libcloud-pin

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
--command
    Build command to execute (defaults to the run command defined in the
    Docker image)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

EXITCODE=0
DEBUG=${ARVADOS_DEBUG:-0}
TARGET=debian7
COMMAND=

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
    debian7)
        FORMAT=deb
        PYTHON_BACKPORTS=(python-gflags==2.0 google-api-python-client==1.4.2 \
            oauth2client==1.5.2 pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py pykka six  \
            ciso8601 pycrypto backports.ssl_match_hostname llfuse==0.41.1 \
            'pycurl<7.21.5' contextlib2 pyyaml 'rdflib>=4.2.0' \
            shellescape mistune typing avro ruamel.ordereddict
            cachecontrol requests)
        PYTHON3_BACKPORTS=(docker-py==1.7.2 six requests websocket-client)
        ;;
    debian8)
        FORMAT=deb
        PYTHON_BACKPORTS=(python-gflags==2.0 google-api-python-client==1.4.2 \
            oauth2client==1.5.2 pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py pykka six  \
            ciso8601 pycrypto backports.ssl_match_hostname llfuse==0.41.1 \
            'pycurl<7.21.5' pyyaml 'rdflib>=4.2.0' \
            shellescape mistune typing avro ruamel.ordereddict
            cachecontrol)
        PYTHON3_BACKPORTS=(docker-py==1.7.2 six requests websocket-client)
        ;;
    ubuntu1204)
        FORMAT=deb
        PYTHON_BACKPORTS=(python-gflags==2.0 google-api-python-client==1.4.2 \
            oauth2client==1.5.2 pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py pykka six  \
            ciso8601 pycrypto backports.ssl_match_hostname llfuse==0.41.1 \
            contextlib2 'pycurl<7.21.5' pyyaml 'rdflib>=4.2.0' \
            shellescape mistune typing avro isodate ruamel.ordereddict
            cachecontrol requests)
        PYTHON3_BACKPORTS=(docker-py==1.7.2 six requests websocket-client)
        ;;
    ubuntu1404)
        FORMAT=deb
        PYTHON_BACKPORTS=(pyasn1==0.1.7 pyasn1-modules==0.0.5 llfuse==0.41.1 ciso8601 \
            google-api-python-client==1.4.2 six uritemplate oauth2client==1.5.2 httplib2 \
            rsa 'pycurl<7.21.5' backports.ssl_match_hostname pyyaml 'rdflib>=4.2.0' \
            shellescape mistune typing avro ruamel.ordereddict
            cachecontrol)
        PYTHON3_BACKPORTS=(docker-py==1.7.2 requests websocket-client)
        ;;
    centos6)
        FORMAT=rpm
        PYTHON2_PACKAGE=$(rpm -qf "$(which python$PYTHON2_VERSION)" --queryformat '%{NAME}\n')
        PYTHON2_PKG_PREFIX=$PYTHON2_PACKAGE
        PYTHON2_PREFIX=/opt/rh/python27/root/usr
        PYTHON2_INSTALL_LIB=lib/python$PYTHON2_VERSION/site-packages
        PYTHON3_PACKAGE=$(rpm -qf "$(which python$PYTHON3_VERSION)" --queryformat '%{NAME}\n')
        PYTHON3_PKG_PREFIX=$PYTHON3_PACKAGE
        PYTHON3_PREFIX=/opt/rh/python33/root/usr
        PYTHON3_INSTALL_LIB=lib/python$PYTHON3_VERSION/site-packages
        PYTHON_BACKPORTS=(python-gflags==2.0 google-api-python-client==1.4.2 \
            oauth2client==1.5.2 pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py pykka six  \
            ciso8601 pycrypto backports.ssl_match_hostname 'pycurl<7.21.5' \
            python-daemon llfuse==0.41.1 'pbr<1.0' pyyaml \
            'rdflib>=4.2.0' shellescape mistune typing avro requests \
            isodate pyparsing sparqlwrapper html5lib==0.9999999 keepalive \
            ruamel.ordereddict cachecontrol)
        PYTHON3_BACKPORTS=(docker-py==1.7.2 six requests websocket-client)
        export PYCURL_SSL_LIBRARY=nss
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
        PYTHON_BACKPORTS=(python-gflags==2.0 google-api-python-client==1.4.2 \
            oauth2client==1.5.2 pyasn1==0.1.7 pyasn1-modules==0.0.5 \
            rsa uritemplate httplib2 ws4py pykka  \
            ciso8601 pycrypto 'pycurl<7.21.5' \
            python-daemon llfuse==0.41.1 'pbr<1.0' pyyaml \
            'rdflib>=4.2.0' shellescape mistune typing avro \
            isodate pyparsing sparqlwrapper html5lib==0.9999999 keepalive \
            ruamel.ordereddict cachecontrol)
        PYTHON3_BACKPORTS=(docker-py==1.7.2 six requests websocket-client)
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
    "$WORKSPACE/LICENSE-2.0.txt=/usr/share/doc/libarvados-perl/LICENSE-2.0.txt" && \
    mv --no-clobber libarvados-perl*.$FORMAT "$WORKSPACE/packages/$TARGET/"

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
    set -e

    cd "$WORKSPACE"
    COMMIT_HASH=$(format_last_commit_here "%H")

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
)

# On older platforms we need to publish a backport of libfuse >=2.9.2,
# and we need to build and install it here in order to even build an
# llfuse package.
cd $WORKSPACE/packages/$TARGET
if [[ $TARGET =~ ubuntu1204 ]]; then
    # port libfuse 2.9.2 to Ubuntu 12.04
    LIBFUSE_DIR=$(mktemp -d)
    (
        cd $LIBFUSE_DIR
        # download fuse 2.9.2 ubuntu 14.04 source package
        file="fuse_2.9.2.orig.tar.xz" && curl -L -o "${file}" "http://archive.ubuntu.com/ubuntu/pool/main/f/fuse/${file}"
        file="fuse_2.9.2-4ubuntu4.14.04.1.debian.tar.xz" && curl -L -o "${file}" "http://archive.ubuntu.com/ubuntu/pool/main/f/fuse/${file}"
        file="fuse_2.9.2-4ubuntu4.14.04.1.dsc" && curl -L -o "${file}" "http://archive.ubuntu.com/ubuntu/pool/main/f/fuse/${file}"

        # install dpkg-source and dpkg-buildpackage commands
        apt-get install -y --no-install-recommends dpkg-dev

        # extract source and apply patches
        dpkg-source -x fuse_2.9.2-4ubuntu4.14.04.1.dsc
        rm -f fuse_2.9.2.orig.tar.xz fuse_2.9.2-4ubuntu4.14.04.1.debian.tar.xz fuse_2.9.2-4ubuntu4.14.04.1.dsc

        # add new version to changelog
        cd fuse-2.9.2
        (
            echo "fuse (2.9.2-5) precise; urgency=low"
            echo
            echo "  * Backported from trusty-security to precise"
            echo
            echo " -- Joshua Randall <jcrandall@alum.mit.edu>  Thu, 4 Feb 2016 11:31:00 -0000"
            echo
            cat debian/changelog
        ) > debian/changelog.new
        mv debian/changelog.new debian/changelog

        # install build-deps and build
        apt-get install -y --no-install-recommends debhelper dh-autoreconf libselinux-dev
        dpkg-buildpackage -rfakeroot -b
    )
    fpm_build "$LIBFUSE_DIR/fuse_2.9.2-5_amd64.deb" fuse "Ubuntu Developers" deb "2.9.2" --iteration 5
    fpm_build "$LIBFUSE_DIR/libfuse2_2.9.2-5_amd64.deb" libfuse2 "Ubuntu Developers" deb "2.9.2" --iteration 5
    fpm_build "$LIBFUSE_DIR/libfuse-dev_2.9.2-5_amd64.deb" libfuse-dev "Ubuntu Developers" deb "2.9.2" --iteration 5
    dpkg -i \
        "$WORKSPACE/packages/$TARGET/fuse_2.9.2-5_amd64.deb" \
        "$WORKSPACE/packages/$TARGET/libfuse2_2.9.2-5_amd64.deb" \
        "$WORKSPACE/packages/$TARGET/libfuse-dev_2.9.2-5_amd64.deb"
    apt-get -y --no-install-recommends -f install
    rm -rf $LIBFUSE_DIR
elif [[ $TARGET =~ centos6 ]]; then
    # port fuse 2.9.2 to centos 6
    # install tools to build rpm from source
    yum install -y rpm-build redhat-rpm-config
    LIBFUSE_DIR=$(mktemp -d)
    (
        cd "$LIBFUSE_DIR"
        # download fuse 2.9.2 centos 7 source rpm
        file="fuse-2.9.2-6.el7.src.rpm" && curl -L -o "${file}" "http://vault.centos.org/7.2.1511/os/Source/SPackages/${file}"
        (
            # modify source rpm spec to remove conflict on filesystem version
            mkdir -p /root/rpmbuild/SOURCES
            cd /root/rpmbuild/SOURCES
            rpm2cpio ${LIBFUSE_DIR}/fuse-2.9.2-6.el7.src.rpm | cpio -i
            perl -pi -e 's/Conflicts:\s*filesystem.*//g' fuse.spec
        )
        # build rpms from source
        rpmbuild -bb /root/rpmbuild/SOURCES/fuse.spec
        rm -f fuse-2.9.2-6.el7.src.rpm
        # move built RPMs to LIBFUSE_DIR
        mv "/root/rpmbuild/RPMS/x86_64/fuse-2.9.2-6.el6.x86_64.rpm" ${LIBFUSE_DIR}/
        mv "/root/rpmbuild/RPMS/x86_64/fuse-libs-2.9.2-6.el6.x86_64.rpm" ${LIBFUSE_DIR}/
        mv "/root/rpmbuild/RPMS/x86_64/fuse-devel-2.9.2-6.el6.x86_64.rpm" ${LIBFUSE_DIR}/
        rm -rf /root/rpmbuild
    )
    fpm_build "$LIBFUSE_DIR/fuse-libs-2.9.2-6.el6.x86_64.rpm" fuse-libs "Centos Developers" rpm "2.9.2" --iteration 5
    fpm_build "$LIBFUSE_DIR/fuse-2.9.2-6.el6.x86_64.rpm" fuse "Centos Developers" rpm "2.9.2" --iteration 5 --no-auto-depends
    fpm_build "$LIBFUSE_DIR/fuse-devel-2.9.2-6.el6.x86_64.rpm" fuse-devel "Centos Developers" rpm "2.9.2" --iteration 5 --no-auto-depends
    yum install -y \
        "$WORKSPACE/packages/$TARGET/fuse-libs-2.9.2-5.x86_64.rpm" \
        "$WORKSPACE/packages/$TARGET/fuse-2.9.2-5.x86_64.rpm" \
        "$WORKSPACE/packages/$TARGET/fuse-devel-2.9.2-5.x86_64.rpm"
fi

# Go binaries
cd $WORKSPACE/packages/$TARGET
export GOPATH=$(mktemp -d)
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
package_go_binary services/datamanager arvados-data-manager \
    "Ensure block replication levels, report disk usage, and determine which blocks should be deleted when space is needed"
package_go_binary services/keep-balance keep-balance \
    "Rebalance and garbage-collect data blocks stored in Arvados Keep"
package_go_binary services/keepproxy keepproxy \
    "Make a Keep cluster accessible to clients that are not on the LAN"
package_go_binary services/keepstore keepstore \
    "Keep storage daemon, accessible to clients on the LAN"
package_go_binary services/keep-web keep-web \
    "Static web hosting service for user data stored in Arvados Keep"
package_go_binary tools/keep-block-check keep-block-check \
    "Verify that all data from one set of Keep servers to another was copied"
package_go_binary tools/keep-rsync keep-rsync \
    "Copy all data from one set of Keep servers to another"

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

# cwl-runner
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/sdk/cwl/build"
fpm_build $WORKSPACE/sdk/cwl "${PYTHON2_PKG_PREFIX}-arvados-cwl-runner" 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/sdk/cwl/arvados_cwl_runner.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=The Arvados CWL runner" --depends "${PYTHON2_PKG_PREFIX}-setuptools" --iteration 3

fpm_build lockfile "" "" python 0.12.2 --epoch 1

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
fpm_build schema_salad "" "" python 1.18.20161005190847 --depends "${PYTHON2_PKG_PREFIX}-lockfile >= 1:0.12.2-2"

# And schema_salad now depends on ruamel-yaml, which apparently has a braindead setup.py that requires special arguments to build (otherwise, it aborts with 'error: you have to install with "pip install ."'). Sigh.
# Ward, 2016-05-26
fpm_build ruamel.yaml "" "" python 0.12.4 --python-setup-py-arguments "--single-version-externally-managed"

# Dependency of cwltool.  Fpm doesn't produce a package with the correct version
# number unless we build it explicitly
fpm_build cwltest "" "" python 1.0.20160907111242

# And for cwltool we have the same problem as for schema_salad. Ward, 2016-03-17
fpm_build cwltool "" "" python 1.0.20161007181528

# FPM eats the trailing .0 in the python-rdflib-jsonld package when built with 'rdflib-jsonld>=0.3.0'. Force the version. Ward, 2016-03-25
fpm_build rdflib-jsonld "" "" python 0.3.0

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

# The Arvados crunchstat-summary tool
cd $WORKSPACE/packages/$TARGET
rm -rf "$WORKSPACE/tools/crunchstat-summary/build"
fpm_build $WORKSPACE/tools/crunchstat-summary ${PYTHON2_PKG_PREFIX}-crunchstat-summary 'Curoverse, Inc.' 'python' "$(awk '($1 == "Version:"){print $2}' $WORKSPACE/tools/crunchstat-summary/crunchstat_summary.egg-info/PKG-INFO)" "--url=https://arvados.org" "--description=Crunchstat-summary reads Arvados Crunch log files and summarize resource usage"

# Forked libcloud
LIBCLOUD_DIR=$(mktemp -d)
(
    cd $LIBCLOUD_DIR
    git clone $DASHQ_UNLESS_DEBUG https://github.com/curoverse/libcloud.git .
    git checkout apache-libcloud-$LIBCLOUD_PIN
    # libcloud is absurdly noisy without -q, so force -q here
    OLD_DASHQ_UNLESS_DEBUG=$DASHQ_UNLESS_DEBUG
    DASHQ_UNLESS_DEBUG=-q
    handle_python_package
    DASHQ_UNLESS_DEBUG=$OLD_DASHQ_UNLESS_DEBUG
)
fpm_build $LIBCLOUD_DIR "$PYTHON2_PKG_PREFIX"-apache-libcloud
rm -rf $LIBCLOUD_DIR

# Python 2 dependencies
declare -a PIP_DOWNLOAD_SWITCHES=(--no-deps)
# Add --no-use-wheel if this pip knows it.
pip wheel --help >/dev/null 2>&1
case "$?" in
    0) PIP_DOWNLOAD_SWITCHES+=(--no-use-wheel) ;;
    2) ;;
    *) echo "WARNING: `pip wheel` test returned unknown exit code $?" ;;
esac

for deppkg in "${PYTHON_BACKPORTS[@]}"; do
    outname=$(echo "$deppkg" | sed -e 's/^python-//' -e 's/[<=>].*//' -e 's/_/-/g' -e "s/^/${PYTHON2_PKG_PREFIX}-/")
    case "$deppkg" in
        httplib2|google-api-python-client)
            # Work around 0640 permissions on some package files.
            # See #7591 and #7991.
            pyfpm_workdir=$(mktemp --tmpdir -d pyfpm-XXXXXX) && (
                set -e
                cd "$pyfpm_workdir"
                pip install "${PIP_DOWNLOAD_SWITCHES[@]}" --download . "$deppkg"
                # Sometimes pip gives us a tarball, sometimes a zip file...
                DOWNLOADED=`ls $deppkg-*`
                [[ "$DOWNLOADED" =~ ".tar" ]] && tar -xf $DOWNLOADED
                [[ "$DOWNLOADED" =~ ".zip" ]] && unzip $DOWNLOADED
                cd "$deppkg"-*/
                "python$PYTHON2_VERSION" setup.py $DASHQ_UNLESS_DEBUG egg_info build
                chmod -R go+rX .
                set +e
                fpm_build . "$outname" "" python "" --iteration 3
                # The upload step uses the package timestamp to determine
                # whether it's new.  --no-clobber plays nice with that.
                mv --no-clobber "$outname"*.$FORMAT "$WORKSPACE/packages/$TARGET"
            )
            if [ 0 != "$?" ]; then
                echo "ERROR: $deppkg build process failed"
                EXITCODE=1
            fi
            if [ -n "$pyfpm_workdir" ]; then
                rm -rf "$pyfpm_workdir"
            fi
            ;;
        *)
            fpm_build "$deppkg" "$outname"
            ;;
    esac
done

# Python 3 dependencies
for deppkg in "${PYTHON3_BACKPORTS[@]}"; do
    outname=$(echo "$deppkg" | sed -e 's/^python-//' -e 's/[<=>].*//' -e 's/_/-/g' -e "s/^/${PYTHON3_PKG_PREFIX}-/")
    # The empty string is the vendor argument: these aren't Curoverse software.
    fpm_build "$deppkg" "$outname" "" python3
done

# Build the API server package
handle_rails_package arvados-api-server "$WORKSPACE/services/api" \
    "$WORKSPACE/agpl-3.0.txt" --url="https://arvados.org" \
    --description="Arvados API server - Arvados is a free and open source platform for big data science." \
    --license="GNU Affero General Public License, version 3.0"

# Build the workbench server package
(
    set -e
    cd "$WORKSPACE/apps/workbench"

    # We need to bundle to be ready even when we build a package without vendor directory
    # because asset compilation requires it.
    bundle install --path vendor/bundle >"$STDOUT_IF_DEBUG"

    # clear the tmp directory; the asset generation step will recreate tmp/cache/assets,
    # and we want that in the package, so it's easier to not exclude the tmp directory
    # from the package - empty it instead.
    rm -rf tmp
    mkdir tmp

    # Set up application.yml and production.rb so that asset precompilation works
    \cp config/application.yml.example config/application.yml -f
    \cp config/environments/production.rb.example config/environments/production.rb -f
    sed -i 's/secret_token: ~/secret_token: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx/' config/application.yml

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

# clean up temporary GOPATH
rm -rf "$GOPATH"

exit $EXITCODE
