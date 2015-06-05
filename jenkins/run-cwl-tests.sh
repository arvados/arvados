#!/bin/bash

read -rd "\000" helpmessage <<EOF
$(basename $0): Test cwl tool and (optionally) upload to PyPi and Docker Hub.

Syntax:
        WORKSPACE=/path/to/common-workflow-language $(basename $0) [options]

Options:

--upload-pypi          Upload package to pypi (default: false)
--upload-docker        Upload packages to docker hub (default: false)
--debug                Output debug information (default: false)

WORKSPACE=path         Path to the common-workflow-language source tree

EOF

EXITCODE=0
CALL_FREIGHT=0

DEBUG=0
UPLOAD_PYPI=0
UPLOAD_DOCKER=0

VENVDIR=

leave_temp=

declare -A leave_temp

set -e

clear_temp() {
    leaving=""
    for var in VENVDIR
    do
        if [[ -z "${leave_temp[$var]}" ]]
        then
            if [[ -n "${!var}" ]]
            then
                rm -rf "${!var}"
            fi
        else
            leaving+=" $var=\"${!var}\""
        fi
    done
    if [[ -n "$leaving" ]]; then
        echo "Leaving behind temp dirs: $leaving"
    fi
}

fatal() {
    clear_temp
    echo >&2 "Fatal: $* (encountered in ${FUNCNAME[1]} at ${BASH_SOURCE[1]} line ${BASH_LINENO[0]})"
    exit 1
}

trap clear_temp INT EXIT

# Set up temporary install dirs (unless existing dirs were supplied)
for tmpdir in VENVDIR
do
    if [[ -n "${!tmpdir}" ]]; then
        leave_temp[$tmpdir]=1
    else
        eval $tmpdir=$(mktemp -d)
    fi
done


while [[ -n "$1" ]]
do
    arg="$1"; shift
    case "$arg" in
        --help)
            echo >&2 "$helpmessage"
            echo >&2
            exit 1
            ;;
        --debug)
            DEBUG=1
            ;;
        --upload-pypi)
            UPLOAD_PYPI=1
            ;;
        --upload-docker)
            UPLOAD_DOCKER=1
            ;;
        --leave-temp)
            leave_temp[VENVDIR]=1
            ;;
        *=*)
            eval export $(echo $arg | cut -d= -f1)=\"$(echo $arg | cut -d= -f2-)\"
            ;;
        *)
            echo >&2 "$0: Unrecognized option: '$arg'. Try: $0 --help"
            exit 1
            ;;
    esac
done

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

if [[ "$DEBUG" != 0 ]]; then
  echo "Workspace is $WORKSPACE"
fi

virtualenv --setuptools "$VENVDIR" || fatal "virtualenv $VENVDIR failed"
. "$VENVDIR/bin/activate"

handle_python_package () {
  # This function assumes the current working directory is the python package directory
  if [[ "$UPLOAD_PYPI" != 0 ]]; then
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

# Make all files world-readable -- jenkins runs with umask 027, and has checked
# out our git tree here
chmod o+r "$WORKSPACE" -R

# Now fix our umask to something better suited to building and publishing
# gems and packages
umask 0022

if [[ "$DEBUG" != 0 ]]; then
  echo "umask is" `umask`
fi

# Python packages
if [[ "$DEBUG" != 0 ]]; then
  echo
  echo "Python packages"
  echo
fi

cd "$WORKSPACE"

pushd reference
python setup.py install
python setup.py test
./build-node-docker.sh
popd

pushd conformance
pwd
./run_test.sh
popd

cd reference
handle_python_package

./build-cwl-docker.sh

if [[ "$UPLOAD_DOCKER" != 0 ]]; then
    docker push commonworkflowlanguage/cwltool_module
    docker push commonworkflowlanguage/cwltool
    docker push commonworkflowlanguage/nodejs-engine
fi
