#!/bin/bash


read -rd "\000" helpmessage <<EOF
$(basename $0): Deploy Arvados to a cluster

Syntax:
        $(basename $0) <identifier>

Options:

  identifier             Arvados cluster name

EOF


IDENTIFIER=$1

if [[ "$IDENTIFIER" == '' ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  exit 1
fi

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

echo $WORKSPACE

function run_puppet() {
  node=$1
  return_var=$2

  TMP_FILE=`mktemp`
  ssh -t -p2222 -o "StrictHostKeyChecking no" -o "ConnectTimeout 5" root@$node.$IDENTIFIER -C "/usr/bin/puppet agent -t" | tee $TMP_FILE

  ECODE=$?
  RESULT=$(cat $TMP_FILE)

  if [[ "$ECODE" != "255" && ! ("$RESULT" =~ 'already in progress') && "$ECODE" != "2" && "$ECODE" != "0"  ]]; then
    # Puppet exists 255 if the connection timed out. Just ignore that, it's possible that this node is
    #   a compute node that was being shut down.
    # Puppet exits 2 if there are changes. For real!
    # Puppet prints 'Notice: Run of Puppet configuration client already in progress' if another puppet process
    #   was already running
    echo "ERROR updating $node.$IDENTIFIER: exit code $ECODE"
  fi
  rm -f $TMP_FILE
  echo
  eval "$return_var=$ECODE"
}

# Deploy API server
title "Deploying API server"

SUM_ECODE=0

# Install updated debian packages
title "Deploying updated arvados debian packages"

ssh -p2222 root@$IDENTIFIER.arvadosapi.com -C "apt-get update && apt-get -qqy install arvados-src python-arvados-fuse python-arvados-python-client arvados-api-server"

ECODE=$?
SUM_ECODE=$(($SUM_ECODE + $ECODE))

ssh -p2222 root@$IDENTIFIER.arvadosapi.com -C "/usr/local/bin/arvados-api-server-upgrade.sh"

ECODE=$?
SUM_ECODE=$(($SUM_ECODE + $ECODE))

if [[ "$SUM_ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING DEBIAN PACKAGES FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $SUM_ECODE))
  exit $EXITCODE
fi

title "Deploying updated arvados debian packages complete"

# Install updated arvados gems
title "Deploying updated arvados gems"

ssh -p2222 root@$IDENTIFIER.arvadosapi.com -C "/usr/local/rvm/bin/rvm default do gem install arvados arvados-cli && /usr/local/rvm/bin/rvm default do gem clean arvados arvados-cli"

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING ARVADOS GEMS FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
  exit $EXITCODE
fi

title "Deploying updated arvados gems complete"
title "Deploying API server complete"

# Deploy Workbench
title "Deploying workbench"

# Install updated debian packages
title "Deploying updated arvados debian packages"

ssh -p2222 root@workbench.$IDENTIFIER.arvadosapi.com -C "apt-get update && apt-get -qqy install python-arvados-fuse python-arvados-python-client arvados-workbench"

ECODE=$?
SUM_ECODE=$(($SUM_ECODE + $ECODE))

ssh -p2222 root@workbench.$IDENTIFIER.arvadosapi.com -C "/usr/local/bin/arvados-workbench-upgrade.sh"

ECODE=$?
SUM_ECODE=$(($SUM_ECODE + $ECODE))

if [[ "$SUM_ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING DEBIAN PACKAGES FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $SUM_ECODE))
  exit $EXITCODE
fi

title "Deploying updated arvados debian packages complete"

title "Deploying workbench complete"

# Update compute node(s)
title "Update compute node(s)"

# Get list of nodes that are up
COMPRESSED_NODE_LIST=`ssh -p2222 root@$IDENTIFIER -C "sinfo --long -p crypto -r -o "%N" -h"`

if [[ "$COMPRESSED_NODE_LIST" != '' ]]; then
  COMPUTE_NODES=`ssh -p2222 root@$IDENTIFIER -C "scontrol show hostname $COMPRESSED_NODE_LIST"`

  SUM_ECODE=0
  for node in $COMPUTE_NODES; do
    echo "Updating $node.$IDENTIFIER"
    run_puppet $node ECODE
    SUM_ECODE=$(($SUM_ECODE + $ECODE))
  done

  if [[ "$SUM_ECODE" != "0" ]]; then
    title "!!!!!! Update compute node(s) FAILED !!!!!!"
    EXITCODE=$(($EXITCODE + $SUM_ECODE))
  fi
fi

title "Update compute node(s) complete"

title "Update shell"

run_puppet shell ECODE

if [[ "$ECODE" == "2" ]]; then
  # Puppet exits '2' if there are changes. For real!
  ECODE=0
fi

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! Update shell FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "Update shell complete"

title "Update keep0"

run_puppet keep0 ECODE

if [[ "$ECODE" == "2" ]]; then
  # Puppet exits '2' if there are changes. For real!
  ECODE=0
fi

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! Update keep0 FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "Update keep0 complete"

exit $EXITCODE
