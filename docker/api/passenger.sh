#!/bin/sh

cd /usr/src/arvados/services/api
export ARVADOS_WEBSOCKETS=1
export RAILS_ENV=production
/usr/local/rvm/bin/rvm-exec default bundle exec rake db:migrate
exec /usr/local/rvm/bin/rvm-exec default bundle exec passenger start -p443 --ssl --ssl-certificate=/etc/ssl/certs/ssl-cert-snakeoil.pem --ssl-certificate-key=/etc/ssl/private/ssl-cert-snakeoil.key
