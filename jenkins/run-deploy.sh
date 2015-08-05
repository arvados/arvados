#!/bin/bash

DEBUG=0

function usage {
    echo >&2
    echo >&2 "usage: $0 [options] <identifier>"
    echo >&2
    echo >&2 "   <identifier>                 Arvados cluster name"
    echo >&2
    echo >&2 "$0 options:"
    echo >&2 "  -d, --debug                   Enable debug output"
    echo >&2 "  -h, --help                    Display this help and exit"
    echo >&2
    echo >&2 "Note: this script requires an arvados token created with these permissions:"
    echo >&2 '  arv api_client_authorization create_system_auth \'
    echo >&2 '    --scopes "[\"GET /arvados/v1/virtual_machines\",'
    echo >&2 '               \"GET /arvados/v1/keep_services/\",'
    echo >&2 '               \"GET /arvados/v1/groups\",'
    echo >&2 '               \"GET /arvados/v1/links\",'
    echo >&2 '               \"GET /arvados/v1/groups/\",'
    echo >&2 '               \"GET /arvados/v1/collections\",'
    echo >&2 '               \"POST /arvados/v1/collections\",'
    echo >&2 '               \"POST /arvados/v1/links\"]"'
    echo >&2
}

# NOTE: This requires GNU getopt (part of the util-linux package on Debian-based distros).
TEMP=`getopt -o hd \
    --long help,debug \
    -n "$0" -- "$@"`

if [ $? != 0 ] ; then echo "Use -h for help"; exit 1 ; fi
# Note the quotes around `$TEMP': they are essential!
eval set -- "$TEMP"

while [ $# -ge 1 ]
do
    case $1 in
        -d | --debug)
            DEBUG=1
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

IDENTIFIER=$1

if [[ "$IDENTIFIER" == '' ]]; then
  usage
  exit 1
fi

EXITCODE=0

COLUMNS=80

title () {
  date=`date +'%Y-%m-%d %H:%M:%S'`
  printf "$date $1\n"
}

function run_puppet() {
  node=$1
  return_var=$2

  title "Running puppet on $node"
  TMP_FILE=`mktemp`
  if [[ "$DEBUG" != "0" ]]; then
    ssh -t -p2222 -o "StrictHostKeyChecking no" -o "ConnectTimeout 5" root@$node -C "/usr/bin/puppet agent -t" | tee $TMP_FILE
  else
    ssh -t -p2222 -o "StrictHostKeyChecking no" -o "ConnectTimeout 5" root@$node -C "/usr/bin/puppet agent -t" > $TMP_FILE 2>&1
  fi

  ECODE=$?
  RESULT=$(cat $TMP_FILE)

  if [[ "$ECODE" != "255" && ! ("$RESULT" =~ 'already in progress') && "$ECODE" != "2" && "$ECODE" != "0"  ]]; then
    # Ssh exits 255 if the connection timed out. Just ignore that.
    # Puppet exits 2 if there are changes. For real!
    # Puppet prints 'Notice: Run of Puppet configuration client already in progress' if another puppet process
    #   was already running
    echo "ERROR running puppet on $node: exit code $ECODE"
    if [[ "$DEBUG" == "0" ]]; then
      title "Command output follows:"
      echo $RESULT
    fi
  fi
  if [[ "$ECODE" == "255" ]]; then
    title "Connection timed out"
    ECODE=0
  fi
  if [[ "$ECODE" == "2" ]]; then
    ECODE=0
  fi
  rm -f $TMP_FILE
  eval "$return_var=$ECODE"
}

function run_command() {
  node=$1
  return_var=$2
  command=$3

  title "Running '$command' on $node"
  TMP_FILE=`mktemp`
  if [[ "$DEBUG" != "0" ]]; then
    ssh -t -p2222 -o "StrictHostKeyChecking no" -o "ConnectTimeout 5" root@$node -C "$command" | tee $TMP_FILE
  else
    ssh -t -p2222 -o "StrictHostKeyChecking no" -o "ConnectTimeout 5" root@$node -C "$command" > $TMP_FILE 2>&1
  fi

  ECODE=$?
  RESULT=$(cat $TMP_FILE)

  if [[ "$ECODE" != "255" && "$ECODE" != "0"  ]]; then
    # Ssh exists 255 if the connection timed out. Just ignore that, it's possible that this node is
    #   a shell node that is down.
    title "ERROR running command on $node: exit code $ECODE"
    if [[ "$DEBUG" == "0" ]]; then
      title "Command output follows:"
      echo $RESULT
    fi
  fi
  if [[ "$ECODE" == "255" ]]; then
    title "Connection timed out"
    ECODE=0
  fi
  rm -f $TMP_FILE
  eval "$return_var=$ECODE"
}

title "Updating API server"
SUM_ECODE=0
run_puppet $IDENTIFIER.arvadosapi.com ECODE
SUM_ECODE=$(($SUM_ECODE + $ECODE))
run_command $IDENTIFIER.arvadosapi.com ECODE "/usr/local/bin/arvados-api-server-upgrade.sh"
SUM_ECODE=$(($SUM_ECODE + $ECODE))
run_command $IDENTIFIER.arvadosapi.com ECODE "dpkg -L arvados-mailchimp-plugin 2>/dev/null && apt-get install arvados-mailchimp-plugin --reinstall || echo"
SUM_ECODE=$(($SUM_ECODE + $ECODE))

if [[ "$SUM_ECODE" != "0" ]]; then
  title "ERROR: Updating API server FAILED"
  EXITCODE=$(($EXITCODE + $SUM_ECODE))
  exit $EXITCODE
fi

title "Loading ARVADOS_API_HOST and ARVADOS_API_TOKEN"
if [[ -f "$HOME/.config/arvados/$IDENTIFIER.arvadosapi.com.conf" ]]; then
  . $HOME/.config/arvados/$IDENTIFIER.arvadosapi.com.conf
else
  title "WARNING: $HOME/.config/arvados/$IDENTIFIER.arvadosapi.com.conf not found."
fi
if [[ "$ARVADOS_API_HOST" == "" ]] || [[ "$ARVADOS_API_TOKEN" == "" ]]; then
  title "ERROR: ARVADOS_API_HOST and/or ARVADOS_API_TOKEN environment variables are not set."
  exit 1
fi

title "Locating Arvados Standard Docker images project"

JSON_FILTER="[[\"name\", \"=\", \"Arvados Standard Docker Images\"], [\"owner_uuid\", \"=\", \"$IDENTIFIER-tpzed-000000000000000\"]]"
DOCKER_IMAGES_PROJECT=`ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN arv --format=uuid group list --filters="$JSON_FILTER"`

if [[ "$DOCKER_IMAGES_PROJECT" == "" ]]; then
  title "Warning: Arvados Standard Docker Images project not found. Creating it."

  DOCKER_IMAGES_PROJECT=`ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN arv --format=uuid group create --group "{\"owner_uuid\":\"$IDENTIFIER-tpzed-000000000000000\", \"name\":\"Arvados Standard Docker Images\", \"group_class\":\"project\"}"`
  ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN arv link create --link "{\"tail_uuid\":\"$IDENTIFIER-j7d0g-fffffffffffffff\", \"head_uuid\":\"$DOCKER_IMAGES_PROJECT\", \"link_class\":\"permission\", \"name\":\"can_read\" }"
  if [[ "$?" != "0" ]]; then
    title "ERROR: could not create standard Docker images project Please create it, cf. http://doc.arvados.org/install/create-standard-objects.html"
    exit 1
  fi
fi

title "Found Arvados Standard Docker Images project with uuid $DOCKER_IMAGES_PROJECT"
GIT_COMMIT=`ssh -o "StrictHostKeyChecking no" $IDENTIFIER cat /usr/local/arvados/src/git-commit.version`

if [[ "$?" != "0" ]] || [[ "$GIT_COMMIT" == "" ]]; then
  title "ERROR: unable to get arvados/jobs Docker image git revision"
  exit 1
else
  title "Found git commit for arvados/jobs Docker image: $GIT_COMMIT"
fi

run_command shell.$IDENTIFIER ECODE "ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN /usr/local/rvm/bin/rvm-exec default arv keep docker" |grep -q $GIT_COMMIT

if [[ "$?" == "0" ]]; then
  title "Found latest arvados/jobs Docker image, nothing to upload"
else
  title "Installing latest arvados/jobs Docker image"
  ssh -o "StrictHostKeyChecking no" shell.$IDENTIFIER "ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN /usr/local/rvm/bin/rvm-exec default arv keep docker --pull --project-uuid=$DOCKER_IMAGES_PROJECT arvados/jobs $GIT_COMMIT"
  if [[ "$?" -ne 0 ]]; then
    title "'git pull' failed exiting..."
    exit 1
  fi
fi

title "Gathering list of shell and Keep nodes"
SHELL_NODES=`ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN arv virtual_machine list |jq .items[].hostname -r`
KEEP_NODES=`ARVADOS_API_HOST=$ARVADOS_API_HOST ARVADOS_API_TOKEN=$ARVADOS_API_TOKEN arv keep_service list |jq .items[].service_host -r`

title "Updating workbench"
SUM_ECODE=0
if [[ `host workbench.$ARVADOS_API_HOST` != `host $ARVADOS_API_HOST` ]]; then
  # Workbench runs on a separate host. We need to run puppet there too.
  run_puppet workbench.$IDENTIFIER ECODE
  SUM_ECODE=$(($SUM_ECODE + $ECODE))
fi

run_command workbench.$IDENTIFIER ECODE "/usr/local/bin/arvados-workbench-upgrade.sh"
SUM_ECODE=$(($SUM_ECODE + $ECODE))

if [[ "$SUM_ECODE" != "0" ]]; then
  title "ERROR: Updating workbench FAILED"
  EXITCODE=$(($EXITCODE + $SUM_ECODE))
  exit $EXITCODE
fi

for n in manage $SHELL_NODES $KEEP_NODES; do
  ECODE=0
  if [[ $n =~ $ARVADOS_API_HOST$ ]]; then
    # e.g. keep.qr1hi.arvadosapi.com
    node=$n
  else
    # e.g. shell
    node=$n.$ARVADOS_API_HOST
  fi

  # e.g. keep.qr1hi
  node=${node%.arvadosapi.com}

  title "Updating $node"
  run_puppet $node ECODE
  if [[ "$ECODE" != "0" ]]; then
    title "ERROR: Updating $node node FAILED: exit code $ECODE"
    EXITCODE=$(($EXITCODE + $ECODE))
    exit $EXITCODE
  fi
done
