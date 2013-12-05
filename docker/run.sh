#!/bin/bash

ENABLE_SSH=false

function usage {
    echo >&2 "usage:"
    echo >&2 "$0 start [--ssh] [--doc] [--sso] [--api] [--workbench] [--keep]"
    echo >&2 "$0 stop"
    echo >&2 "$0 test"
    echo >&2 "If no switches are given, the default is to start all servers."
}

function ip_address {
    local container=$1
    echo `docker inspect $container  |grep IPAddress |cut -f4 -d\"`
}

function start_container {
    local args="-d -i -t"
    if [[ "$1" != '' ]]; then
      local port="$1"
      args="$args -p $port"
    fi
    if [[ "$2" != '' ]]; then
      local name="$2"
      args="$args -name $name"
    fi
    if [[ "$3" != '' ]]; then
      local volume="$3"
      args="$args -v $volume"
    fi
    if [[ "$4" != '' ]]; then
      local link="$4"
      args="$args -link $link"
    fi
    local image=$5

    if $ENABLE_SSH
    then
      args="$args -e ENABLE_SSH=$ENABLE_SSH"
    fi

    `docker ps |grep -P "$name[^/]" -q`
    if [[ "$?" == "0" ]]; then
      echo "You have a running container with name $name -- skipping."
      return
    fi

    # Remove any existing container by this name.
    docker rm "$name" 2>/dev/null

    echo "Starting container:"
    echo "  docker run $args $image"
    container=`docker run $args $image`
    if [[ "$?" != "0" ]]; then
      echo "Unable to start container"
      exit 1
    fi
    if $ENABLE_SSH
    then
      ip=$(ip_address $container )
      echo
      echo "You can ssh into the container with:"
      echo
      echo "    ssh root@$ip"
      echo
    fi
}

declare -a keep_volumes

# Initialize the global `keep_volumes' array. If any keep volumes
# already appear to exist (mounted volumes with a top-level "keep"
# directory), use them; create temporary volumes if necessary.
#
function make_keep_volumes () {
    # Mount a keep volume if we don't already have one
    for mountpoint in $(cut -d ' ' -f 2 /proc/mounts); do
      if [[ -d "$mountpoint/keep" && "$mountpoint" != "/" ]]; then
	keep_volumes+=($mountpoint)
      fi
    done

    # Create any keep volumes that do not yet exist.
    while [ ${#keep_volumes[*]} -lt 2 ]
    do
	new_keep=$(mktemp -d)
	echo >&2 "mounting 512M tmpfs keep volume in $new_keep"
	sudo mount -t tmpfs -o size=512M tmpfs $new_keep
	mkdir $new_keep/keep
	keep_volumes+=($new_keep)
    done
}

function do_start {
    local start_doc=false
    local start_sso=false
    local start_api=false
    local start_workbench=false
    local start_keep=false

    while [ $# -ge 1 ]
    do
        case $1 in
	    --doc)
		start_doc=true
		;;
	    --sso)
		start_sso=true
		;;
	    --api)
		start_api=true
		;;
	    --workbench)
		start_workbench=true
		;;
	    --keep)
		start_keep=true
		;;
	    --ssh)
		ENABLE_SSH=true
		;;
	    *)
		usage
		exit 1
		;;
	esac
	shift
    done

    # If no options were selected, then start all servers.
    if $start_doc || $start_sso || $start_api || $start_workbench || $start_keep
    then
      :
    else
      start_doc=true
      start_sso=true
      start_api=true
      start_workbench=true
      start_keep=true
    fi

    $start_doc && start_container "9898:80" "doc_server" '' '' "arvados/doc"
    $start_sso && start_container "9901:443" "sso_server" '' '' "arvados/sso"
    $start_api && start_container "9900:443" "api_server" '' "sso_server:sso" "arvados/api"
    $start_workbench && start_container "9899:80" "workbench_server" '' "api_server:api" "arvados/workbench"

    if $start_keep
    then
	# create `keep_volumes' array with a list of keep mount points
	# remove any stale metadata from those volumes before starting them
	make_keep_volumes
	for v in ${keep_volumes[*]}
	do
	    [ -f $v/.metadata.yml ] && rm $v/.metadata.yml
	done
	start_container "25107:25107" "keep_server_0" \
	    "${keep_volumes[0]}:/dev/keep-0" \
	    "api_server:api" \
	    "arvados/warehouse"
	start_container "25108:25107" "keep_server_1" \
	    "${keep_volumes[1]}:/dev/keep-0" \
	    "api_server:api" \
	    "arvados/warehouse"
    fi

    ARVADOS_API_HOST=$(ip_address "api_server")
    ARVADOS_API_HOST_INSECURE=yes
    ARVADOS_API_TOKEN=$(grep '^\w' api/generated/secret_token.rb | cut -d "'" -f 2)

    echo "To run a test suite:"
    echo "export ARVADOS_API_HOST=$ARVADOS_API_HOST"
    echo "export ARVADOS_API_HOST_INSECURE=$ARVADOS_API_HOST_INSECURE"
    echo "export ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN"
    echo "python -m unittest discover ../sdk/python"
}

function do_stop {
    docker stop api_server \
	sso_server \
	workbench_server \
	keep_server_0 \
	keep_server_1 2>/dev/null
}

if [ $# -lt 1 ]
then
  usage
  exit 1
fi

case $1 in
    start)
	shift
	do_start $@
	;;
    stop)
	shift
	do_stop $@
	;;
    *)
	usage
	exit 1
	;;
esac
