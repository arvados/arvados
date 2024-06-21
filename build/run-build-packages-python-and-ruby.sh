#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

COLUMNS=80

. `dirname "$(readlink -f "$0")"`/run-library.sh

read -rd "\000" helpmessage <<EOF
$(basename $0): Build Arvados Python packages and Ruby gems

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

Options:

--debug
    Output debug information (default: false)
--upload
    If the build and test steps are successful, upload the python
    packages to pypi and the gems to rubygems (default: false)
--ruby <true|false>
    Build ruby gems (default: true)
--python <true|false>
    Build python packages (default: true)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

exit_cleanly() {
    trap - INT
    report_outcomes
    exit ${#failures[@]}
}

gem_wrapper() {
  local gem_name="$1"; shift
  local gem_directory="$1"; shift

  title "Start $gem_name gem build"
  timer_reset

  cd "$gem_directory"
  handle_ruby_gem $gem_name

  checkexit $? "$gem_name gem build"
  title "End of $gem_name gem build (`timer`)"
}

handle_python_package () {
  # This function assumes the current working directory is the python package directory
  local -a pkg_fmts=()
  local version="$(nohash_version_from_git)"
  if [[ -z "$(find dist -name "*-$version.tar.gz" -print -quit)" ]]; then
    pkg_fmts+=(sdist)
  fi
  if [[ -z "$(find dist -name "*-$version-py*.whl" -print -quit)" ]]; then
    pkg_fmts+=(bdist_wheel)
  fi
  if [[ "${#pkg_fmts[@]}" -eq 0 ]]; then
    echo "This package doesn't need rebuilding."
  else
    python3 setup.py $DASHQ_UNLESS_DEBUG "${pkg_fmts[@]}"
  fi
}

python_wrapper() {
  local package_name="$1"; shift
  local package_directory="$1"; shift

  title "Start $package_name python package build"
  timer_reset

  cd "$package_directory"
  if [[ $DEBUG > 0 ]]; then
    echo `pwd`
  fi
  handle_python_package

  checkexit $? "$package_name python package build"
  title "End of $package_name python package build (`timer`)"
}

TARGET=
UPLOAD=0
RUBY=1
PYTHON=1
DEBUG=${ARVADOS_DEBUG:-0}

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,debug,ruby:,python:,upload,target: \
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
        --ruby)
            RUBY="$2"; shift
            if [ "$RUBY" != "true" ] && [ "$RUBY" != "1" ]; then
              RUBY=0
            else
              RUBY=1
            fi
            ;;
        --python)
            PYTHON="$2"; shift
            if [ "$PYTHON" != "true" ] && [ "$PYTHON" != "1" ]; then
              PYTHON=0
            else
              PYTHON=1
            fi
            ;;
        --upload)
            UPLOAD=1
            ;;
        --debug)
            DEBUG=1
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

if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

STDOUT_IF_DEBUG=/dev/null
STDERR_IF_DEBUG=/dev/null
DASHQ_UNLESS_DEBUG=-q
if [[ "$DEBUG" != 0 ]]; then
    STDOUT_IF_DEBUG=/dev/stdout
    STDERR_IF_DEBUG=/dev/stderr
    DASHQ_UNLESS_DEBUG=
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

if [ $RUBY -eq 0 ] && [ $PYTHON -eq 0 ]; then
  echo "Nothing to do!"
  exit 0
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

GEM_BUILD_FAILURES=0
if [ $RUBY -eq 1 ]; then
  debug_echo "Building Ruby gems"
  gem_wrapper arvados "$WORKSPACE/sdk/ruby"
  gem_wrapper arvados-cli "$WORKSPACE/sdk/cli"
  gem_wrapper arvados-login-sync "$WORKSPACE/services/login-sync"
  if [ ${#failures[@]} -ne 0 ]; then
    GEM_BUILD_FAILURES=${#failures[@]}
  fi
fi

PYTHON_BUILD_FAILURES=0
if [ $PYTHON -eq 1 ]; then
  debug_echo "Building Python packages"
  python_wrapper arvados-python-client "$WORKSPACE/sdk/python"
  python_wrapper arvados-cwl-runner "$WORKSPACE/sdk/cwl"
  python_wrapper arvados_fuse "$WORKSPACE/services/fuse"
  python_wrapper crunchstat_summary "$WORKSPACE/tools/crunchstat-summary"
  python_wrapper arvados-user-activity "$WORKSPACE/tools/user-activity"

  if [ $((${#failures[@]} - $GEM_BUILD_FAILURES)) -ne 0 ]; then
    PYTHON_BUILD_FAILURES=$((${#failures[@]} - $GEM_BUILD_FAILURES))
  fi
fi

if [ $UPLOAD -ne 0 ]; then
  echo "Uploading"

  if [ $DEBUG > 0 ]; then
    EXTRA_UPLOAD_FLAGS=" --verbose"
  else
    EXTRA_UPLOAD_FLAGS=""
  fi

  if [ ! -e "$WORKSPACE/packages" ]; then
    mkdir -p "$WORKSPACE/packages"
  fi

  if [ $PYTHON -eq 1 ]; then
    title "Start upload python packages"
    timer_reset

    if [ $PYTHON_BUILD_FAILURES -eq 0 ]; then
      /usr/local/arvados-dev/jenkins/run_upload_packages.py $EXTRA_UPLOAD_FLAGS --workspace $WORKSPACE python
    else
      echo "Skipping python packages upload, there were errors building the packages"
    fi
    checkexit $? "upload python packages"
    title "End of upload python packages (`timer`)"
  fi

  if [ $RUBY -eq 1 ]; then
    title "Start upload ruby gems"
    timer_reset

    if [ $GEM_BUILD_FAILURES -eq 0 ]; then
      /usr/local/arvados-dev/jenkins/run_upload_packages.py $EXTRA_UPLOAD_FLAGS --workspace $WORKSPACE gems
    else
      echo "Skipping ruby gem upload, there were errors building the packages"
    fi
    checkexit $? "upload ruby gems"
    title "End of upload ruby gems (`timer`)"
  fi
fi

exit_cleanly
