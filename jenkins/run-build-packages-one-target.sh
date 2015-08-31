#!/bin/bash

read -rd "\000" helpmessage <<EOF
$(basename $0): Orchestrate run-build-packages.sh for one target

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) [options]

--target <target>
    Distribution to build packages for (default: debian7)
--command
    Build command to execute (default: use built-in Docker image command)

WORKSPACE=path         Path to the Arvados source tree to build packages from

EOF

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
    help,target:,command: \
    -- "" "$@")
if [ $? -ne 0 ]; then
    exit 1
fi

TARGET=debian7
COMMAND=

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

set -e

if [[ "$COMMAND" != "" ]]; then
  COMMAND="/usr/local/rvm/bin/rvm-exec default bash /jenkins/$COMMAND --target $TARGET"
fi

FINAL_EXITCODE=0
JENKINS_DIR=$(dirname "$(readlink -e "$0")")

run_docker() {
    local tag=$1; shift
    if docker run -v "$JENKINS_DIR:/jenkins" -v "$WORKSPACE:/arvados" \
          --env ARVADOS_DEBUG=1 "arvados/build:$tag" $COMMAND; then
        # Success - nothing more to do.
        true
    else
        FINAL_EXITCODE=$?
        echo "ERROR: $tag build failed with exit status $FINAL_EXITCODE." >&2
    fi
}

# In case it's needed, build the container. This costs just a few
# seconds when the container already exist, so it's not a big deal to
# do it on each run.
cd "$JENKINS_DIR/dockerfiles"
echo $TARGET
cd $TARGET
time docker build -t arvados/build:$TARGET .
cd ..

run_docker $TARGET

#for dockerfile_path in $(find -name Dockerfile); do
#    run_docker "$(basename $(dirname "$dockerfile_path"))"
#done

exit $FINAL_EXITCODE
