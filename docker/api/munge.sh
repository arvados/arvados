#!/bin/sh
rm -rf /var/run/munge
exec /etc/init.d/munge start
