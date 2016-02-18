#!/bin/bash

EXITCODE=0

COLUMNS=80

title () {
  printf "\n%*s\n\n" $(((${#title}+$COLUMNS)/2)) "********** $1 **********"
}

source /etc/profile.d/rvm.sh

# This shouldn't really be necessary... but the jenkins/rvm integration seems a
# bit wonky occasionally.
rvm use ree

echo $WORKSPACE

# Tapestry
title "Starting tapestry tests"
cd "$WORKSPACE"

# There are a few submodules
git submodule init && git submodule update

# Use sqlite for testing
sed -i'' -e "s:mysql:sqlite3:" Gemfile

# Tapestry is not set up yet to use --deployment
#bundle install --deployment
bundle install

rm -f config/database.yml
rm -f config/environments/test.rb
cp $HOME/tapestry/test.rb config/environments/
cp $HOME/tapestry/database.yml config/

export RAILS_ENV=test

bundle exec rake db:drop
bundle exec rake db:create
bundle exec rake db:setup
bundle exec rake test

ECODE=$?

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! TAPESTRY TESTS FAILED !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
fi

title "Tapestry tests complete"

exit $EXITCODE
