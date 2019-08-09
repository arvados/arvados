#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec 2>&1
set -ex -o pipefail

if [[ -s /etc/arvados/config.yml ]] ; then
   exit
fi

. /usr/local/lib/arvbox/common.sh

set -u

if ! test -s /var/lib/arvados/api_uuid_prefix ; then
  ruby -e 'puts "x#{rand(2**64).to_s(36)[0,4]}"' > /var/lib/arvados/api_uuid_prefix
fi
uuid_prefix=$(cat /var/lib/arvados/api_uuid_prefix)

if ! test -s /var/lib/arvados/api_secret_token ; then
    ruby -e 'puts rand(2**400).to_s(36)' > /var/lib/arvados/api_secret_token
fi
secret_token=$(cat /var/lib/arvados/api_secret_token)

if ! test -s /var/lib/arvados/blob_signing_key ; then
    ruby -e 'puts rand(2**400).to_s(36)' > /var/lib/arvados/blob_signing_key
fi
blob_signing_key=$(cat /var/lib/arvados/blob_signing_key)

if ! test -s /var/lib/arvados/management_token ; then
    ruby -e 'puts rand(2**400).to_s(36)' > /var/lib/arvados/management_token
fi
management_token=$(cat /var/lib/arvados/management_token)

if ! test -s /var/lib/arvados/sso_app_secret ; then
    ruby -e 'puts rand(2**400).to_s(36)' > /var/lib/arvados/sso_app_secret
fi
sso_app_secret=$(cat /var/lib/arvados/sso_app_secret)

if ! test -s /var/lib/arvados/vm-uuid ; then
    echo $uuid_prefix-2x53u-$(ruby -e 'puts rand(2**400).to_s(36)[0,15]') > /var/lib/arvados/vm-uuid
fi
vm_uuid=$(cat /var/lib/arvados/vm-uuid)

if ! test -f /var/lib/arvados/api_database_pw ; then
    ruby -e 'puts rand(2**128).to_s(36)' > /var/lib/arvados/api_database_pw
fi
database_pw=$(cat /var/lib/arvados/api_database_pw)

if ! (psql postgres -c "\du" | grep "^ arvados ") >/dev/null ; then
    psql postgres -c "create user arvados with password '$database_pw'"
fi
psql postgres -c "ALTER USER arvados WITH SUPERUSER;"

if ! test -s /var/lib/arvados/workbench_secret_token ; then
  ruby -e 'puts rand(2**400).to_s(36)' > /var/lib/arvados/workbench_secret_token
fi
workbench_secret_key_base=$(cat /var/lib/arvados/workbench_secret_token)

if test -s /var/lib/arvados/api_rails_env ; then
  database_env=$(cat /var/lib/arvados/api_rails_env)
else
  database_env=development
fi

cat >/var/lib/arvados/cluster_config.yml <<EOF
Clusters:
  ${uuid_prefix}:
    ManagementToken: $management_token
    Services:
      Workbench1:
        ExternalURL: "https://$localip:${services[workbench]}"
      Workbench2:
        ExternalURL: "https://$localip:${services[workbench2-ssl]}"
      SSO:
        ExternalURL: "https://$localip:${services[sso]}"
      Keepproxy:
        InternalURLs:
          "http://localhost:${services[keepproxy]}/": {}
        ExternalURL: "http://$localip:${services[keepproxy-ssl]}/"
      Websocket:
        ExternalURL: "wss://$localip:${services[websockets-ssl]}/websocket"
      GitSSH:
        ExternalURL: "ssh://git@$localip:"
      GitHTTP:
        ExternalURL: "http://$localip:${services[arv-git-httpd]}/"
      WebDAV:
        ExternalURL: "https://$localip:${services[keep-web-ssl]}/"
      Composer:
        ExternalURL: "http://$localip:${services[composer]}"
      Controller:
        ExternalURL: "https://$localip:${services[controller-ssl]}"
    NodeProfiles:  # to be deprecated in favor of "Services" section
      "*":
        arvados-controller:
          Listen: ":${services[controller]}" # choose a port
        arvados-api-server:
          Listen: ":${services[api]}" # must match Rails server port in your Nginx config
    PostgreSQL:
      ConnectionPool: 32 # max concurrent connections per arvados server daemon
      Connection:
        # All parameters here are passed to the PG client library in a connection string;
        # see https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-PARAMKEYWORDS
        host: localhost
        user: arvados
        password: ${database_pw}
        dbname: arvados_${database_env}
        client_encoding: utf8
    API:
      RailsSessionSecretToken: $secret_token
    Collections:
      BlobSigningKey: $blob_signing_key
      DefaultReplication: 1
    Login:
      ProviderAppSecret: $sso_app_secret
      ProviderAppID: arvados-server
    Users:
      NewUsersAreActive: true
      AutoAdminFirstUser: true
      AutoSetupNewUsers: true
      AutoSetupNewUsersWithVmUUID: $vm_uuid
      AutoSetupNewUsersWithRepository: true
    Workbench:
      SecretKeyBase: $workbench_secret_key_base
      ArvadosDocsite: http://$localip:${services[doc]}/
EOF

/usr/local/lib/arvbox/yml_override.py /var/lib/arvados/cluster_config.yml

cp /var/lib/arvados/cluster_config.yml /etc/arvados/config.yml

mkdir -p /var/lib/arvados/run_tests
cat >/var/lib/arvados/run_tests/config.yml <<EOF
Clusters:
  zzzzz:
    PostgreSQL:
      Connection:
        host: localhost
        user: arvados
        password: ${database_pw}
        dbname: arvados_test
        client_encoding: utf8
EOF
