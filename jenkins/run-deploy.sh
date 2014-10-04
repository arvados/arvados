#!/bin/bash

IDENTIFIER=$1
DEPLOY_REPO=$2

if [[ "$IDENTIFIER" == '' ]]; then
  echo "Syntax: $0 <identifier> <deploy_repo_name>"
  exit 1
fi

if [[ "$DEPLOY_REPO" == '' ]]; then
  echo "Syntax: $0 <identifier> <deploy_repo_name>"
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

# Update compute0
title "Update compute0"

ssh -p2222 root@compute0.$IDENTIFIER -C "/usr/bin/puppet agent -t"

ECODE=$?

if [[ "$ECODE" == "2" ]]; then
  # Puppet exits '2' if there are changes. For real!
  ECODE=0
fi

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! Update compute0 FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "Update compute0 complete"

title "Update shell"

ssh -p2222 root@shell.$IDENTIFIER -C "/usr/bin/puppet agent -t"

ECODE=$?

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

ssh -p2222 root@keep0.$IDENTIFIER -C "/usr/bin/puppet agent -t"

ECODE=$?

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
