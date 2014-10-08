#!/bin/bash

EXITCODE=0

INSTANCE=$1
REVISION=$2

if [[ "$INSTANCE" == '' ]]; then
  echo "Syntax: $0 <instance> [revision]"
  exit 1
fi

# Sanity check
if ! [[ -n "$WORKSPACE" ]]; then
  echo "WORKSPACE environment variable not set"
  exit 1
fi

title () {
    txt="********** $1 **********"
    printf "\n%*s%s\n\n" $((($COLUMNS-${#txt})/2)) "" "$txt"
}

timer_reset() {
    t0=$SECONDS
}

timer() {
    echo -n "$(($SECONDS - $t0))s"
}

source /etc/profile.d/rvm.sh
echo $WORKSPACE

title "Starting diagnostics"
timer_reset

cd $WORKSPACE

if [[ "$REVISION" != '' ]]; then
  git checkout $REVISION
fi

cp -f /home/jenkins/diagnostics/arvados-workbench/$INSTANCE-application.yml $WORKSPACE/apps/workbench/config/application.yml

cd $WORKSPACE/apps/workbench

HOME="$GEMHOME" bundle install --no-deployment

if [[ ! -d tmp ]]; then
  mkdir tmp
fi

RAILS_ENV=diagnostics bundle exec rake TEST=test/diagnostics/pipeline_test.rb

ECODE=$?

if [[ "$REVISION" != '' ]]; then
  git checkout master
fi

if [[ "$ECODE" != "0" ]]; then
  title "!!!!!! DIAGNOSTICS FAILED (`timer`) !!!!!!"
  EXITCODE=$(($EXITCODE + $ECODE))
  exit $EXITCODE
fi

title "Diagnostics complete (`timer`)"

exit $EXITCODE
