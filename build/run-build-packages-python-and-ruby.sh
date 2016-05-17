#!/bin/bash

COLUMNS=80

. `dirname "$(readlink -f "$0")"`/run-library.sh
#. `dirname "$(readlink -f "$0")"`/libcloud-pin

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
DEBUG=${ARVADOS_DEBUG:-0}

PARSEDOPTS=$(getopt --name "$0" --longoptions \
    help,debug,upload,target: \
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

gem_wrapper arvados "$WORKSPACE/sdk/ruby"
gem_wrapper arvados-cli "$WORKSPACE/sdk/cli"
gem_wrapper arvados-login-sync "$WORKSPACE/services/login-sync"

GEM_BUILD_FAILURES=0
if [ ${#failures[@]} -ne 0 ]; then
  GEM_BUILD_FAILURES=${#failures[@]}
fi

python_wrapper arvados-pam "$WORKSPACE/sdk/pam"
python_wrapper arvados-python-client "$WORKSPACE/sdk/python"
python_wrapper arvados-cwl-runner "$WORKSPACE/sdk/cwl"
python_wrapper arvados_fuse "$WORKSPACE/services/fuse"
python_wrapper arvados-node-manager "$WORKSPACE/services/nodemanager"

PYTHON_BUILD_FAILURES=0
if [ $((${#failures[@]} - $GEM_BUILD_FAILURES)) -ne 0 ]; then
  PYTHON_BUILD_FAILURES=${#failures[@]} - $GEM_BUILD_FAILURES
fi

if [[ "$UPLOAD" != 0 ]]; then

  if [[ $DEBUG > 0 ]]; then
    EXTRA_UPLOAD_FLAGS=" --verbose"
  else
    EXTRA_UPLOAD_FLAGS=""
  fi

  if [[ ! -e "$WORKSPACE/packages" ]]; then
    mkdir -p "$WORKSPACE/packages"
  fi

  title "Start upload python packages"
  timer_reset

  if [ "$PYTHON_BUILD_FAILURES" -eq 0 ]; then
    /usr/local/arvados-dev/jenkins/run_upload_packages.py $EXTRA_UPLOAD_FLAGS --workspace $WORKSPACE python
  else
    echo "Skipping python packages upload, there were errors building the packages"
  fi
  checkexit $? "upload python packages"
  title "End of upload python packages (`timer`)"

  title "Start upload ruby gems"
  timer_reset

  if [ "$GEM_BUILD_FAILURES" -eq 0 ]; then
    /usr/local/arvados-dev/jenkins/run_upload_packages.py $EXTRA_UPLOAD_FLAGS --workspace $WORKSPACE gems
  else
    echo "Skipping ruby gem upload, there were errors building the packages"
  fi
  checkexit $? "upload ruby gems"
  title "End of upload ruby gems (`timer`)"

fi

exit_cleanly
