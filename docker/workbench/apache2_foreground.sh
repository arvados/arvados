#! /bin/bash

read pid cmd state ppid pgrp session tty_nr tpgid rest < /proc/self/stat
trap "kill -TERM -$pgrp; exit" EXIT TERM KILL SIGKILL SIGTERM SIGQUIT

# Override the default API server address if necessary.
if [[ "$API_PORT_443_TCP_ADDR" != "" ]]; then
  sed -i "s/localhost:9900/$API_PORT_443_TCP_ADDR/" /usr/src/arvados/apps/workbench/config/application.yml
fi

source /etc/apache2/envvars
/usr/sbin/apache2 -D FOREGROUND
