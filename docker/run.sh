#!/bin/bash

function usage {
  echo >&2
  echo >&2 "usage: $0 [options]"
  echo >&2 "  -d [port], --doc[=port]        Start documentation server (default port 9898)"
  echo >&2 "  -w [port], --workbench[=port]  Start workbench server (default port 9899)"
  echo >&2 "  -s [port], --sso[=port]        Start SSO server (default port 9901)"
  echo >&2 "  -a [port], --api[=port]        Start API server (default port 9900)"
  echo >&2 "  -k, --keep                     Start Keep servers"
  echo >&2 "  -h, --help                     Display this help and exit"
  echo >&2
  echo >&2 "If no switches are given, the default is to start all servers on the default"
  echo >&2 "ports."
  echo >&2
}

if [[ "$ENABLE_SSH" != "" ]]; then
  EXTRA=" -e ENABLE_SSH=$ENABLE_SSH"
else
  EXTRA=''
fi

start_doc=false
start_sso=false
start_api=false
start_workbench=false
start_keep=false

# NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
TEMP=`getopt -o d::s::a::w::kh --long doc::,sso::,api::,workbench::,keep,help \
             -n "$0" -- "$@"`

if [ $? != 0 ] ; then echo "Use -h for help"; exit 1 ; fi

# Note the quotes around `$TEMP': they are essential!
eval set -- "$TEMP"

# For optional argument, as we are in quoted mode,
# an empty parameter will be generated if its optional
# argument is not found.
while true; do
  case "$1" in
    -k | --keep ) start_keep=true; shift ;;
    -h | --help ) usage; exit ;;
    -d | --doc)
      case "$2" in
        "") start_doc=9898; shift 2 ;;
        *)  start_doc=$2; shift 2 ;;
      esac ;;
    -s | --sso)
      case "$2" in
        "") start_sso=9901; shift 2 ;;
        *)  start_sso=$2; shift 2 ;;
      esac ;;
    -a | --api)
      case "$2" in
        "") start_api=9900; shift 2 ;;
        *)  start_api=$2; shift 2 ;;
      esac ;;
    -a | --workbench)
      case "$2" in
        "") start_workbench=9899; shift 2 ;;
        *)  start_workbench=$2; shift 2 ;;
      esac ;;
    -- ) shift; break ;;
    * ) usage; exit ;;
  esac
done

# If no options were selected, then start all servers.
if [[ $start_doc != false || $start_sso != false || $start_api != false || $start_workbench != false || $start_keep != false ]]; then
    :
else
    start_doc=9898
    start_sso=9901
    start_api=9900
    start_workbench=9899
    start_keep=true
fi

function ip_address {
  local container=$1
  echo `docker inspect $container  |grep IPAddress |cut -f4 -d\"`
}

function start_container {
  local port="-p $1"
  if [[ "$2" != '' ]]; then
    local name="-name $2"
  fi
  if [[ "$3" != '' ]]; then
    local volume="-v $3"
  fi
  if [[ "$4" != '' ]]; then
    local link="-link $4"
  fi
  local image=$5

  `docker ps |grep -P "$2[^/]" -q`
  if [[ "$?" == "0" ]]; then
    echo "You have a running container with name $2 -- skipping."
    return
  fi

  echo "Starting container:"
  echo "  docker run -d -i -t$EXTRA $port $name $volume $link $image"
  container=`docker run -d -i -t$EXTRA $port $name $volume $link $image`
  if [[ "$?" != "0" ]]; then
    echo "Unable to start container"
    exit 1
  fi
  if [[ $EXTRA ]]; then
    ip=$(ip_address $container )
    echo
    echo "You can ssh into the container with:"
    echo
    echo "    ssh root@$ip"
    echo
  fi
}

function make_keep_volume {
  # Mount a keep volume if we don't already have one
  local keepvolume=""
  for mountpoint in $(cut -d ' ' -f 2 /proc/mounts); do
    if [[ -d "$mountpoint/keep" && "$mountpoint" != "/" ]]; then
      keepvolume=$mountpoint
    fi
  done

  if [[ "$keepvolume" == '' ]]; then
    keepvolume=$(mktemp -d)
    echo "mounting 512M tmpfs keep volume in $keepvolume"
    sudo mount -t tmpfs -o size=512M tmpfs $keepvolume
    mkdir $keepvolume/keep
  fi
  echo "$keepvolume"
}

if [[ $start_doc != false ]]; then
  start_container "$start_doc:80" "doc_server" '' '' "arvados/doc"
fi
if [[ $start_sso != false ]]; then
  start_container "$start_sso:443" "sso_server" '' '' "arvados/sso"
fi
if [[ $start_api != false ]]; then
  start_container "$start_api:443" "api_server" '' "sso_server:sso" "arvados/api"
fi
if [[ $start_workbench != false ]]; then
  start_container "$start_workbench:80" "workbench_server" '' "api_server:api" "arvados/workbench"
fi

if [[ $start_keep == true ]]; then
  keepvolume=$(make_keep_volume)
  start_container "25107:25107" "keep_server_0" "$keepvolume:/dev/keep-0" '' "arvados/warehouse"
  start_container "25108:25107" "keep_server_1" "$keepvolume:/dev/keep-0" '' "arvados/warehouse"
fi
