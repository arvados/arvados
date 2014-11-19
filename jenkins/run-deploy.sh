#!/bin/bash


read -rd "\000" helpmessage <<EOF
$(basename $0): Deploy Arvados to a cluster

Syntax:
        WORKSPACE=/path/to/arvados $(basename $0) <identifier> <deploy_repo_name>

Options:

identifier             Arvados cluster name
deploy_repo_name       Name for the repository with the (capistrano) deploy scripts

WORKSPACE=path         Path to the Arvados source tree to deploy from

EOF


IDENTIFIER=$1
DEPLOY_REPO=$2

if [[ "$IDENTIFIER" == '' || "$DEPLOY_REPO" == '' ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  exit 1
fi

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
  echo >&2 "$helpmessage"
  echo >&2
  echo >&2 "Error: WORKSPACE environment variable not set"
  echo >&2
  exit 1
fi

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

# We only install capistrano in dev mode
export RAILS_ENV=development

source /etc/profile.d/rvm.sh
echo $WORKSPACE

# Weirdly, jenkins/rvm ties itself in a knot.
rvm use default

# Just say what version of ruby we're running
ruby --version

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

function ensure_symlink() {
  if [[ ! -L $WORKSPACE/$1 ]]; then
    ln -s $WORKSPACE/$DEPLOY_REPO/$1 $WORKSPACE/$1
  fi
}

# Check out/update the $DEPLOY_REPO repository
if [[ ! -d $DEPLOY_REPO ]]; then
  mkdir $DEPLOY_REPO
  git clone git@git.curoverse.com:$DEPLOY_REPO.git
else
  cd $DEPLOY_REPO
  git pull
fi

# Make sure the necessary symlinks are in place
cd "$WORKSPACE"
ensure_symlink "apps/workbench/Capfile.workbench.$IDENTIFIER"
ensure_symlink "apps/workbench/config/deploy.common.rb"
ensure_symlink "apps/workbench/config/deploy.curoverse.rb"
ensure_symlink "apps/workbench/config/deploy.workbench.$IDENTIFIER.rb"

ensure_symlink "services/api/Capfile.$IDENTIFIER"
ensure_symlink "services/api/config/deploy.common.rb"
ensure_symlink "services/api/config/deploy.$IDENTIFIER.rb"

# Deploy API server
title "Deploying API server"
cd "$WORKSPACE"
cd services/api

bundle install --deployment

# make sure we do not print the output of config:check
sed -i'' -e "s/RAILS_ENV=production #{rake} config:check/RAILS_ENV=production QUIET=true #{rake} config:check/" $WORKSPACE/$DEPLOY_REPO/services/api/config/deploy.common.rb

bundle exec cap deploy -f Capfile.$IDENTIFIER

ECODE=$?

# restore unaltered deploy.common.rb
cd $WORKSPACE/$DEPLOY_REPO
git checkout services/api/config/deploy.common.rb

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING API SERVER FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
  exit $EXITCODE
fi

title "Deploying API server complete"

# Install updated debian packages
title "Deploying updated arvados debian packages"

ssh -p2222 root@$IDENTIFIER.arvadosapi.com -C "apt-get update && apt-get -qqy install arvados-src python-arvados-fuse python-arvados-python-client"

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING DEBIAN PACKAGES FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
  exit $EXITCODE
fi

title "Deploying updated arvados debian packages complete"

# Install updated arvados gems
title "Deploying updated arvados gems"

ssh -p2222 root@$IDENTIFIER.arvadosapi.com -C "/usr/local/rvm/bin/rvm default do gem install arvados arvados-cli && /usr/local/rvm/bin/rvm default do gem clean arvados arvados-cli"

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING ARVADOS GEMS FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
  exit $EXITCODE
fi

title "Deploying updated arvados gems complete"

# Deploy Workbench
title "Deploying workbench"
cd "$WORKSPACE"
cd apps/workbench
bundle install --deployment

# make sure we do not print the output of config:check
sed -i'' -e "s/RAILS_ENV=production #{rake} config:check/RAILS_ENV=production QUIET=true #{rake} config:check/" $WORKSPACE/$DEPLOY_REPO/apps/workbench/config/deploy.common.rb

bundle exec cap deploy -f Capfile.workbench.$IDENTIFIER

ECODE=$?

# restore unaltered deploy.common.rb
cd $WORKSPACE/$DEPLOY_REPO
git checkout apps/workbench/config/deploy.common.rb

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! DEPLOYING WORKBENCH FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
  exit $EXITCODE
fi

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
