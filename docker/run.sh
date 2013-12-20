#!/bin/bash

ENABLE_SSH=false

function usage {
    echo >&2
    echo >&2 "usage: $0 (start|stop|test) [options]"
    echo >&2
    echo >&2 "$0 start options:"
    echo >&2 "  -d [port], --doc[=port]        Start documentation server (default port 9898)"
    echo >&2 "  -w [port], --workbench[=port]  Start workbench server (default port 9899)"
    echo >&2 "  -s [port], --sso[=port]        Start SSO server (default port 9901)"
    echo >&2 "  -a [port], --api[=port]        Start API server (default port 9900)"
    echo >&2 "  -k, --keep                     Start Keep servers"
    echo >&2 "  --ssh                          Enable SSH access to server containers"
    echo >&2 "  -h, --help                     Display this help and exit"
    echo >&2
    echo >&2 "  If no switches are given, the default is to start all"
    echo >&2 "  servers on the default ports."
    echo >&2
    echo >&2 "$0 stop"
    echo >&2 "  Stop all servers."
    echo >&2
    echo >&2 "$0 test [testname] [testname] ..."
    echo >&2 "  By default, all tests are run."
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

    # NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
    local TEMP=`getopt -o d::s::a::w::kh \
                  --long doc::,sso::,api::,workbench::,keep,help,ssh \
                  -n "$0" -- "$@"`

    if [ $? != 0 ] ; then echo "Use -h for help"; exit 1 ; fi

    # Note the quotes around `$TEMP': they are essential!
    eval set -- "$TEMP"

    while [ $# -ge 1 ]
    do
        case $1 in
	    -d | --doc)
		case "$2" in
		    "") start_doc=9898; shift 2 ;;
		    *)  start_doc=$2; shift 2 ;;
		esac
		;;
	    -s | --sso)
		case "$2" in
		    "") start_sso=9901; shift 2 ;;
		    *)  start_sso=$2; shift 2 ;;
		esac
		;;
	    -a | --api)
		case "$2" in
		    "") start_api=9900; shift 2 ;;
		    *)  start_api=$2; shift 2 ;;
		esac
		;;
	    -w | --workbench)
		case "$2" in
		    "") start_workbench=9899; shift 2 ;;
		    *)  start_workbench=$2; shift 2 ;;
		esac
		;;
	    -k | --keep )
		start_keep=true
		shift
		;;
	    --ssh)
		# ENABLE_SSH is a global variable
		ENABLE_SSH=true
		shift
		;;
	    --)
		shift
		break
		;;
	    *)
		usage
		exit 1
		;;
	esac
    done

    # If no options were selected, then start all servers.
    if [[ $start_doc == false &&
	  $start_sso == false &&
	  $start_api == false &&
	  $start_workbench == false &&
	  $start_keep == false ]]
    then
	start_doc=9898
	start_sso=9901
	start_api=9900
	start_workbench=9899
	start_keep=true
    fi

    if [[ $start_doc != false ]]
    then
	start_container "9898:80" "doc_server" '' '' "arvados/doc"
    fi

    if [[ $start_sso != false ]]
    then
	start_container "9901:443" "sso_server" '' '' "arvados/sso"
    fi

    if [[ $start_api != false ]]
    then
	start_container "9900:443" "api_server" '' "sso_server:sso" "arvados/api"
    fi

    if [[ $start_workbench != false ]]
    then
	start_container "9899:80" "workbench_server" '' "api_server:api" "arvados/workbench"
    fi

    if [[ $start_keep != false ]]
    then
	# create `keep_volumes' array with a list of keep mount points
	# remove any stale metadata from those volumes before starting them
	make_keep_volumes
	for v in ${keep_volumes[*]}
	do
	    [ -f $v/keep/.metadata.yml ] && rm $v/keep/.metadata.yml
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
    ARVADOS_API_TOKEN=$(cat api/generated/superuser_token)

    echo "To run a test suite:"
    echo "export ARVADOS_API_HOST=$ARVADOS_API_HOST"
    echo "export ARVADOS_API_HOST_INSECURE=$ARVADOS_API_HOST_INSECURE"
    echo "export ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN"
    echo "python -m unittest discover ../sdk/python"
}

function do_stop {
    docker stop doc_server \
	api_server \
	sso_server \
	workbench_server \
	keep_server_0 \
	keep_server_1 2>/dev/null
}

function do_test {
    local alltests
    if [ $# -lt 1 ]
    then
	alltests="python-sdk api"
    else
	alltests="$@"
    fi

    for testname in $alltests
    do
	echo "testing $testname..."
	case $testname in
	    python-sdk)
		do_start --api --keep --sso
		export ARVADOS_API_HOST=$(ip_address "api_server")
		export ARVADOS_API_HOST_INSECURE=yes
		export ARVADOS_API_TOKEN=$(cat api/generated/superuser_token)
		python -m unittest discover ../sdk/python
		;;
	    api)
		docker run -t -i arvados/api \
		    /usr/src/arvados/services/api/script/rake_test.sh
		;;
	    *)
		echo >&2 "unknown test $testname"
		;;
	esac
    done
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
    test)
	shift
	do_test $@
	;;
    *)
	usage
	exit 1
	;;
esac
