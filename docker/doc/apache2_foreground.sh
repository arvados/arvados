#! /bin/bash

read pid cmd state ppid pgrp session tty_nr tpgid rest < /proc/self/stat
trap "kill -TERM -$pgrp; exit" EXIT TERM KILL SIGKILL SIGTERM SIGQUIT

# Start ssh daemon if requested via the ENABLE_SSH env variable
if [[ ! "$ENABLE_SSH" =~ (0|false|no|f|^$) ]]; then
  /etc/init.d/ssh start
fi

source /etc/apache2/envvars
/usr/sbin/apache2 -D FOREGROUND
