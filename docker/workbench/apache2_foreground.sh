#! /bin/bash

read pid cmd state ppid pgrp session tty_nr tpgid rest < /proc/self/stat
trap "kill -TERM -$pgrp; exit" EXIT TERM KILL SIGKILL SIGTERM SIGQUIT

# Override the default API server address if necessary.
if [[ "$API_PORT_443_TCP_ADDR" != "" ]]; then
    sed -i "s/arvados_login_base: '.*'/arvados_login_base: 'https:\/\/$API_PORT_443_TCP_ADDR\/login'/" /usr/src/arvados/apps/workbench/config/application.yml
    sed -i "s/arvados_v1_base: '.*'/arvados_v1_base: 'https:\/\/$API_PORT_443_TCP_ADDR\/arvados\/v1'/" /usr/src/arvados/apps/workbench/config/application.yml
fi

source /etc/apache2/envvars
/usr/sbin/apache2 -D FOREGROUND
