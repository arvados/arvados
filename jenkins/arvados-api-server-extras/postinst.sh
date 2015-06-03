#!/bin/bash

cd /var/www/arvados-api

chown -R www-data:www-data tmp >/dev/null 2>&1
chown -R www-data:www-data log >/dev/null 2>&1
chown www-data:www-data db/structure.sql >/dev/null 2>&1
chmod 644 log/* >/dev/null 2>&1

# Errors above are not serious
exit 0

