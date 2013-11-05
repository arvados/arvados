#! /bin/sh

# set up the RVM environment
source /usr/local/rvm/scripts/rvm

/usr/bin/service postgresql start
rake -f /usr/src/arvados/services/api/Rakefile db:setup

