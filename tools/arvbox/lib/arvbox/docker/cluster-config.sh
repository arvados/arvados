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

uuid_prefix=$(cat /var/lib/arvados/api_uuid_prefix)
secret_token=$(cat /var/lib/arvados/api_secret_token)
blob_signing_key=$(cat /var/lib/arvados/blob_signing_key)
management_token=$(cat /var/lib/arvados/management_token)
sso_app_secret=$(cat /var/lib/arvados/sso_app_secret)
vm_uuid=$(cat /var/lib/arvados/vm-uuid)
database_pw=$(cat /var/lib/arvados/api_database_pw)

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
