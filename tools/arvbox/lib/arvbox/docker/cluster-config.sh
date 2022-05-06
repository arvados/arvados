#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec 2>&1
set -ex -o pipefail

export ARVADOS_CONTAINER_PATH=/var/lib/arvados-arvbox

if [[ -s /etc/arvados/config.yml ]] && [[ $ARVADOS_CONTAINER_PATH/cluster_config.yml.override -ot /etc/arvados/config.yml ]] ; then
   exit
fi

. /usr/local/lib/arvbox/common.sh

set -u

if ! test -s $ARVADOS_CONTAINER_PATH/api_uuid_prefix ; then
  ruby -e 'puts "x#{rand(2**64).to_s(36)[0,4]}"' > $ARVADOS_CONTAINER_PATH/api_uuid_prefix
fi
uuid_prefix=$(cat $ARVADOS_CONTAINER_PATH/api_uuid_prefix)

if ! test -s $ARVADOS_CONTAINER_PATH/api_secret_token ; then
    ruby -e 'puts rand(2**400).to_s(36)' > $ARVADOS_CONTAINER_PATH/api_secret_token
fi
secret_token=$(cat $ARVADOS_CONTAINER_PATH/api_secret_token)

if ! test -s $ARVADOS_CONTAINER_PATH/blob_signing_key ; then
    ruby -e 'puts rand(2**400).to_s(36)' > $ARVADOS_CONTAINER_PATH/blob_signing_key
fi
blob_signing_key=$(cat $ARVADOS_CONTAINER_PATH/blob_signing_key)

if ! test -s $ARVADOS_CONTAINER_PATH/management_token ; then
    ruby -e 'puts rand(2**400).to_s(36)' > $ARVADOS_CONTAINER_PATH/management_token
fi
management_token=$(cat $ARVADOS_CONTAINER_PATH/management_token)

if ! test -s $ARVADOS_CONTAINER_PATH/system_root_token ; then
    ruby -e 'puts rand(2**400).to_s(36)' > $ARVADOS_CONTAINER_PATH/system_root_token
fi
system_root_token=$(cat $ARVADOS_CONTAINER_PATH/system_root_token)

if ! test -s $ARVADOS_CONTAINER_PATH/vm-uuid ; then
    echo $uuid_prefix-2x53u-$(ruby -e 'puts rand(2**400).to_s(36)[0,15]') > $ARVADOS_CONTAINER_PATH/vm-uuid
fi
vm_uuid=$(cat $ARVADOS_CONTAINER_PATH/vm-uuid)

if ! test -f $ARVADOS_CONTAINER_PATH/api_database_pw ; then
    ruby -e 'puts rand(2**128).to_s(36)' > $ARVADOS_CONTAINER_PATH/api_database_pw
fi
database_pw=$(cat $ARVADOS_CONTAINER_PATH/api_database_pw)

if ! (psql postgres -c "\du" | grep "^ arvados ") >/dev/null ; then
    psql postgres -c "create user arvados with password '$database_pw'"
fi
psql postgres -c "ALTER USER arvados WITH SUPERUSER;"

if ! test -s $ARVADOS_CONTAINER_PATH/workbench_secret_token ; then
  ruby -e 'puts rand(2**400).to_s(36)' > $ARVADOS_CONTAINER_PATH/workbench_secret_token
fi
workbench_secret_key_base=$(cat $ARVADOS_CONTAINER_PATH/workbench_secret_token)

if test -s $ARVADOS_CONTAINER_PATH/api_rails_env ; then
  database_env=$(cat $ARVADOS_CONTAINER_PATH/api_rails_env)
else
  database_env=development
fi

cat >$ARVADOS_CONTAINER_PATH/cluster_config.yml <<EOF
Clusters:
  ${uuid_prefix}:
    SystemRootToken: $system_root_token
    ManagementToken: $management_token
    Services:
      RailsAPI:
        InternalURLs:
          "http://localhost:${services[api]}": {}
      Workbench1:
        ExternalURL: "https://$localip:${services[workbench]}"
      Workbench2:
        ExternalURL: "https://$localip:${services[workbench2-ssl]}"
      Keepproxy:
        ExternalURL: "https://$localip:${services[keepproxy-ssl]}"
        InternalURLs:
          "http://localhost:${services[keepproxy]}": {}
      Keepstore:
        InternalURLs:
          "http://localhost:${services[keepstore0]}": {}
          "http://localhost:${services[keepstore1]}": {}
      Websocket:
        ExternalURL: "wss://$localip:${services[websockets-ssl]}/websocket"
        InternalURLs:
          "http://localhost:${services[websockets]}": {}
      GitSSH:
        ExternalURL: "ssh://git@$localip:"
      GitHTTP:
        InternalURLs:
          "http://localhost:${services[arv-git-httpd]}/": {}
        ExternalURL: "https://$localip:${services[arv-git-httpd-ssl]}/"
      WebDAV:
        InternalURLs:
          "http://localhost:${services[keep-web]}/": {}
        ExternalURL: "https://$localip:${services[keep-web-ssl]}/"
      WebDAVDownload:
        InternalURLs:
          "http://localhost:${services[keep-web]}/": {}
        ExternalURL: "https://$localip:${services[keep-web-dl-ssl]}/"
      Controller:
        ExternalURL: "https://$localip:${services[controller-ssl]}"
        InternalURLs:
          "http://localhost:${services[controller]}": {}
      WebShell:
        InternalURLs: {}
        ExternalURL: "https://$localip:${services[webshell-ssl]}"
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
    Collections:
      BlobSigningKey: $blob_signing_key
      DefaultReplication: 1
      TrustAllContent: true
    Containers:
      RuntimeEngine: singularity
    Login:
      Test:
        Enable: true
    Users:
      NewUsersAreActive: true
      AutoAdminFirstUser: true
      AutoSetupNewUsers: true
      AutoSetupNewUsersWithVmUUID: $vm_uuid
      AutoSetupNewUsersWithRepository: true
    Workbench:
      SecretKeyBase: $workbench_secret_key_base
      ArvadosDocsite: http://$localip:${services[doc]}/
    Git:
      GitCommand: /usr/share/gitolite3/gitolite-shell
      GitoliteHome: $ARVADOS_CONTAINER_PATH/git
      Repositories: $ARVADOS_CONTAINER_PATH/git/repositories
    Volumes:
      ${uuid_prefix}-nyw5e-000000000000000:
        Driver: Directory
        DriverParameters:
          Root: $ARVADOS_CONTAINER_PATH/keep0
        AccessViaHosts:
          "http://localhost:${services[keepstore0]}": {}
      ${uuid_prefix}-nyw5e-111111111111111:
        Driver: Directory
        DriverParameters:
          Root: $ARVADOS_CONTAINER_PATH/keep1
        AccessViaHosts:
          "http://localhost:${services[keepstore1]}": {}
EOF

/usr/local/lib/arvbox/yml_override.py $ARVADOS_CONTAINER_PATH/cluster_config.yml

cp $ARVADOS_CONTAINER_PATH/cluster_config.yml /etc/arvados/config.yml

# Do not abort if certain optional files don't exist (e.g. cluster_config.yml.override)
set +e
chmod og-rw \
      $ARVADOS_CONTAINER_PATH/cluster_config.yml.override \
      $ARVADOS_CONTAINER_PATH/cluster_config.yml \
      /etc/arvados/config.yml \
      $ARVADOS_CONTAINER_PATH/api_secret_token \
      $ARVADOS_CONTAINER_PATH/blob_signing_key \
      $ARVADOS_CONTAINER_PATH/management_token \
      $ARVADOS_CONTAINER_PATH/system_root_token \
      $ARVADOS_CONTAINER_PATH/api_database_pw \
      $ARVADOS_CONTAINER_PATH/workbench_secret_token \
      $ARVADOS_CONTAINER_PATH/superuser_token \
set -e

mkdir -p $ARVADOS_CONTAINER_PATH/run_tests
cat >$ARVADOS_CONTAINER_PATH/run_tests/config.yml <<EOF
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
