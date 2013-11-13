#! /bin/bash

/etc/init.d/postgresql start
rake -f /usr/src/arvados/services/api/Rakefile db:setup
rake -f /usr/src/arvados/services/api/Rakefile assets:precompile
