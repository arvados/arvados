#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

read -rd "\000" helpmessage <<EOF
$(basename $0): Orchestrate run-build-packages.sh for one target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

--target <target>
    Distribution to build packages for (default: debian10)
--command
    Build command to execute (default: use built-in Docker image command)
--test-packages
    Run package install test script "test-packages-[target].sh"
--debug
    Output debug information (default: false)
--only-build <package>
    Build only a specific package
--only-test <package>
    Test only a specific package
--arch <arch>
    Build a specific architecture (amd64 or arm64, defaults to native architecture)
--force-build
    Build even if the package exists upstream or if it has already been
    built locally
--force-test
    Test even if there is no new untested package
--build-version <string>
    Version to build (default:
    \$ARVADOS_BUILDING_VERSION-\$ARVADOS_BUILDING_ITERATION or
    0.1.timestamp.commithash)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

set -e

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

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,debug,test-packages,target:,command:,only-test:,force-test,only-build:,force-build,arch:,build-version: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

TARGET=debian10
FORCE_BUILD=0
COMMAND=
DEBUG=

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
        --only-test)
            test_packages=1
            testing_one_package=1
            packages="$2"; shift
            ;;
        --force-test)
            FORCE_TEST=true
            ;;
        --force-build)
            FORCE_BUILD=1
            ;;
        --only-build)
            ONLY_BUILD="$2"; shift
            ;;
        --arch)
            ARCH="$2"; shift
            ;;
        --debug)
            DEBUG=" --debug"
            ARVADOS_DEBUG="1"
            ;;
        --command)
            COMMAND="$2"; shift
            ;;
        --test-packages)
            test_packages=1
            ;;
        --build-version)
            if [[ -z "$2" ]]; then
                :
            elif ! [[ "$2" =~ (.*)-(.*) ]]; then
                echo >&2 "FATAL: --build-version '$2' does not include an iteration. Try '${2}-1'?"
                exit 1
            elif ! [[ "$2" =~ ^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+|)(~rc[0-9]+|~dev[0-9]+|)-[0-9]+$ ]]; then
                echo >&2 "FATAL: --build-version '$2' is invalid, must match pattern ^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+|)(~rc[0-9]+|~dev[0-9]+|)-[0-9]+$"
                exit 1
            else
                [[ "$2" =~ (.*)-(.*) ]]
                ARVADOS_BUILDING_VERSION="${BASH_REMATCH[1]}"
                ARVADOS_BUILDING_ITERATION="${BASH_REMATCH[2]}"
            fi
            shift
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

set -e

if [[ -n "$ARVADOS_BUILDING_VERSION" ]]; then
    echo "build version='$ARVADOS_BUILDING_VERSION', package iteration='$ARVADOS_BUILDING_ITERATION'"
fi

if [[ -n "$test_packages" ]]; then
  if [[ -n "$(find $WORKSPACE/packages/$TARGET -name '*.rpm')" ]] ; then
    set +e
    /usr/bin/which createrepo >/dev/null
    if [[ "$?" != "0" ]]; then
      echo >&2
      echo >&2 "Error: please install createrepo. E.g. sudo apt-get install createrepo"
      echo >&2
      exit 1
    fi
    set -e
    createrepo $WORKSPACE/packages/$TARGET
  fi

  if [[ -n "$(find $WORKSPACE/packages/$TARGET -name '*.deb')" ]] ; then
    set +e
    /usr/bin/which dpkg-scanpackages >/dev/null
    if [[ "$?" != "0" ]]; then
      echo >&2
      echo >&2 "Error: please install dpkg-dev. E.g. sudo apt-get install dpkg-dev"
      echo >&2
      exit 1
    fi
    /usr/bin/which apt-ftparchive >/dev/null
    if [[ "$?" != "0" ]]; then
      echo >&2
      echo >&2 "Error: please install apt-utils. E.g. sudo apt-get install apt-utils"
      echo >&2
      exit 1
    fi
    set -e
    (cd $WORKSPACE/packages/$TARGET
      dpkg-scanpackages --multiversion .  2> >(grep -v 'warning' 1>&2) | tee Packages | gzip -c > Packages.gz
      apt-ftparchive -o APT::FTPArchive::Release::Origin=Arvados release . > Release
    )
  fi

  COMMAND="/jenkins/package-testing/test-packages-$TARGET.sh"
  IMAGE="arvados/package-test:$TARGET"
else
  IMAGE="arvados/build:$TARGET"
  if [[ "$COMMAND" != "" ]]; then
    COMMAND="/usr/local/rvm/bin/rvm-exec default bash /jenkins/$COMMAND --target $TARGET$DEBUG"
  fi
fi

JENKINS_DIR=$(dirname "$(readlink -e "$0")")

if [[ -n "$test_packages" ]]; then
    pushd "$JENKINS_DIR/package-test-dockerfiles"
    make "$TARGET/generated"
else
    pushd "$JENKINS_DIR/package-build-dockerfiles"
    make "$TARGET/generated"
fi

GOVERSION=$(grep 'const goversion =' $WORKSPACE/lib/install/deps.go |awk -F'"' '{print $2}')

echo $TARGET
cd $TARGET
time docker build --tag "$IMAGE" \
  --build-arg HOSTTYPE=$HOSTTYPE \
  --build-arg BRANCH=$(git rev-parse --abbrev-ref HEAD) \
  --build-arg GOVERSION=$GOVERSION --no-cache .
popd

if test -z "$packages" ; then
    packages="arvados-api-server
        arvados-client
        arvados-controller
        arvados-dispatch-cloud
        arvados-dispatch-lsf
        arvados-docker-cleaner
        arvados-git-httpd
        arvados-health
        arvados-server
        arvados-src
        arvados-sync-groups
        arvados-sync-users
        arvados-workbench
        arvados-workbench2
        arvados-ws
        crunch-dispatch-local
        crunch-dispatch-slurm
        crunch-run
        crunchstat
        keepproxy
        keepstore
        keep-balance
        keep-block-check
        keep-rsync
        keep-exercise
        keep-rsync
        keep-block-check
        keep-web
        libarvados-perl
        libpam-arvados-go
        python3-cwltest
        python3-arvados-fuse
        python3-arvados-python-client
        python3-arvados-cwl-runner
        python3-crunchstat-summary
        python3-arvados-user-activity"
fi

FINAL_EXITCODE=0

package_fails=""

mkdir -p "$WORKSPACE/apps/workbench/vendor/cache-$TARGET"
mkdir -p "$WORKSPACE/services/api/vendor/cache-$TARGET"

docker_volume_args=(
    -v "$JENKINS_DIR:/jenkins"
    -v "$WORKSPACE:/arvados"
    -v /arvados/services/api/vendor/bundle
    -v /arvados/apps/workbench/vendor/bundle
    -v "$WORKSPACE/services/api/vendor/cache-$TARGET:/arvados/services/api/vendor/cache"
    -v "$WORKSPACE/apps/workbench/vendor/cache-$TARGET:/arvados/apps/workbench/vendor/cache"
)

if [[ -n "$test_packages" ]]; then
    for p in $packages ; do
        if [[ -n "$ONLY_BUILD" ]] && [[ "$p" != "$ONLY_BUILD" ]]; then
            continue
        fi
        if [[ -e "${WORKSPACE}/packages/.last_test_${TARGET}" ]] && [[ -z "$FORCE_TEST" ]]; then
          MATCH=`find ${WORKSPACE}/packages/ -newer ${WORKSPACE}/packages/.last_test_${TARGET} -regex .*${TARGET}/$p.*`
          if [[ "$MATCH" == "" ]]; then
            # No new package has been built that needs testing
            echo "Skipping $p test because no new package was built since the last test."
            continue
          fi
        fi
        # If we're testing all packages, we should not error out on packages that don't exist.
        # If we are testing one specific package only (i.e. --only-test was given), we should
        # error out if that package does not exist.
        if [[ -z "$testing_one_package" ]]; then
          MATCH=`find ${WORKSPACE}/packages/ -regextype posix-extended -regex .*${TARGET}/$p.*\\(deb\\|rpm\\)`
          if [[ "$MATCH" == "" ]]; then
            # No new package has been built that needs testing
            echo "Skipping $p test because no package file is available to test."
            continue
          fi
        fi
        echo
        echo "START: $p test on $IMAGE" >&2
        if docker run \
            --rm \
            "${docker_volume_args[@]}" \
            --env ARVADOS_DEBUG=$ARVADOS_DEBUG \
            --env "TARGET=$TARGET" \
            --env "WORKSPACE=/arvados" \
            "$IMAGE" $COMMAND $p
        then
            echo "OK: $p test on $IMAGE succeeded" >&2
        else
            FINAL_EXITCODE=$?
            package_fails="$package_fails $p"
            echo "ERROR: $p test on $IMAGE failed with exit status $FINAL_EXITCODE" >&2
        fi
    done

    if [[ "$FINAL_EXITCODE" == "0" ]]; then
      touch ${WORKSPACE}/packages/.last_test_${TARGET}
    fi
else
    echo
    echo "START: build packages on $IMAGE" >&2
    # Move existing packages and other files into the processed/ subdirectory
    if [[ ! -e "${WORKSPACE}/packages/${TARGET}/processed" ]]; then
      mkdir -p "${WORKSPACE}/packages/${TARGET}/processed"
    fi
    set +e
    mv -f ${WORKSPACE}/packages/${TARGET}/* ${WORKSPACE}/packages/${TARGET}/processed/ 2>/dev/null
    set -e
    # give bundle (almost) all the cores. See also the MAKE env var that is passed into the
    # docker run command below.
    # Cf. https://build.betterup.com/one-weird-trick-that-will-speed-up-your-bundle-install/
    tmpfile=$(mktemp /tmp/run-build-packages-one-target.XXXXXX)
    cores=$(let a=$(grep -c processor /proc/cpuinfo )-1; echo $a)
    printf -- "---\nBUNDLE_JOBS: \"$cores\"" > $tmpfile
    # Build packages.
    if docker run \
        --rm \
        "${docker_volume_args[@]}" \
        -v $tmpfile:/root/.bundle/config \
        --env ARVADOS_BUILDING_VERSION="$ARVADOS_BUILDING_VERSION" \
        --env ARVADOS_BUILDING_ITERATION="$ARVADOS_BUILDING_ITERATION" \
        --env ARVADOS_DEBUG=$ARVADOS_DEBUG \
        --env "ONLY_BUILD=$ONLY_BUILD" \
        --env "FORCE_BUILD=$FORCE_BUILD" \
        --env "ARCH=$ARCH" \
        --env "MAKE=make --jobs $cores" \
        "$IMAGE" $COMMAND
    then
        echo
        echo "OK: build packages on $IMAGE succeeded" >&2
    else
        FINAL_EXITCODE=$?
        echo "ERROR: build packages on $IMAGE failed with exit status $FINAL_EXITCODE" >&2
    fi
    # Clean up the bundle config file
    rm -f $tmpfile
fi

if test -n "$package_fails" ; then
    echo "Failed package tests:$package_fails" >&2
fi

exit $FINAL_EXITCODE
