#! /bin/bash

read pid cmd state ppid pgrp session tty_nr tpgid rest < /proc/self/stat
trap "kill -HUP -$pgrp" HUP
trap "kill -TERM -$pgrp; exit" EXIT TERM QUIT

source /etc/apache2/envvars
/usr/sbin/apache2 -D FOREGROUND
