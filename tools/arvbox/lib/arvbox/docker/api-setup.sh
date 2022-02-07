#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

exec 2>&1
set -ex -o pipefail

. /usr/local/lib/arvbox/common.sh
. /usr/local/lib/arvbox/go-setup.sh

cd /usr/src/arvados/services/api

if test -s $ARVADOS_CONTAINER_PATH/api_rails_env ; then
  export RAILS_ENV=$(cat $ARVADOS_CONTAINER_PATH/api_rails_env)
else
  export RAILS_ENV=development
fi

set -u

flock $ARVADOS_CONTAINER_PATH/cluster_config.yml.lock /usr/local/lib/arvbox/cluster-config.sh

if test -a /usr/src/arvados/services/api/config/arvados_config.rb ; then
    rm -f config/application.yml config/database.yml
else
    uuid_prefix=$(cat $ARVADOS_CONTAINER_PATH/api_uuid_prefix)
    secret_token=$(cat $ARVADOS_CONTAINER_PATH/api_secret_token)
    blob_signing_key=$(cat $ARVADOS_CONTAINER_PATH/blob_signing_key)
    management_token=$(cat $ARVADOS_CONTAINER_PATH/management_token)
    database_pw=$(cat $ARVADOS_CONTAINER_PATH/api_database_pw)
    vm_uuid=$(cat $ARVADOS_CONTAINER_PATH/vm-uuid)

    cat >config/application.yml <<EOF
$RAILS_ENV:
  uuid_prefix: $uuid_prefix
  secret_token: $secret_token
  blob_signing_key: $blob_signing_key
  workbench_address: "https://$localip/"
  websocket_address: "wss://$localip:${services[websockets-ssl]}/websocket"
  git_repo_ssh_base: "git@$localip:"
  git_repo_https_base: "http://$localip:${services[arv-git-httpd]}/"
  new_users_are_active: true
  auto_admin_first_user: true
  auto_setup_new_users: true
  auto_setup_new_users_with_vm_uuid: $vm_uuid
  auto_setup_new_users_with_repository: true
  default_collection_replication: 1
  docker_image_formats: ["v2"]
  keep_web_service_url: https://$localip:${services[keep-web-ssl]}/
  ManagementToken: $management_token
EOF

    (cd config && /usr/local/lib/arvbox/yml_override.py application.yml)
    sed "s/password:.*/password: $database_pw/" <config/database.yml.example >config/database.yml
fi

if ! test -f $ARVADOS_CONTAINER_PATH/api_database_setup ; then
   flock $GEM_HOME/gems.lock bin/rake db:setup
   touch $ARVADOS_CONTAINER_PATH/api_database_setup
fi

if ! test -s $ARVADOS_CONTAINER_PATH/superuser_token ; then
    superuser_tok=$(flock $GEM_HOME/gems.lock bin/bundle exec ./script/create_superuser_token.rb)
    echo "$superuser_tok" > $ARVADOS_CONTAINER_PATH/superuser_token
fi

rm -rf tmp
mkdir -p tmp/cache

flock $GEM_HOME/gems.lock bin/rake db:migrate
