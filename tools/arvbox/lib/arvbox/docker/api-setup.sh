#!/bin/bash

exec 2>&1
set -ex -o pipefail

. /usr/local/lib/arvbox/common.sh

cd /usr/src/arvados/services/api
export RAILS_ENV=development

set -u

if ! test -s /var/lib/arvados/api_uuid_prefix ; then
    ruby -e 'puts "#{rand(2**64).to_s(36)[0,5]}"' > /var/lib/arvados/api_uuid_prefix
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

# self signed key will be created by SSO server script.
test -s /var/lib/arvados/self-signed.key

sso_app_secret=$(cat /var/lib/arvados/sso_app_secret)

if test -s /var/lib/arvados/vm-uuid ; then
    vm_uuid=$(cat /var/lib/arvados/vm-uuid)
else
    vm_uuid=$uuid_prefix-2x53u-$(ruby -e 'puts rand(2**400).to_s(36)[0,15]')
    echo $vm_uuid > /var/lib/arvados/vm-uuid
fi

cat >config/application.yml <<EOF
development:
  uuid_prefix: $uuid_prefix
  secret_token: $secret_token
  blob_signing_key: $blob_signing_key
  sso_app_secret: $sso_app_secret
  sso_app_id: arvados-server
  sso_provider_url: "https://$localip:${services[sso]}"
  sso_insecure: true
  workbench_address: "http://$localip/"
  websocket_address: "ws://$localip:${services[websockets]}/websocket"
  git_repo_ssh_base: "git@$localip:"
  git_repo_https_base: "http://$localip:${services[arv-git-httpd]}/"
  new_users_are_active: true
  auto_admin_first_user: true
  auto_setup_new_users: true
  auto_setup_new_users_with_vm_uuid: $vm_uuid
  auto_setup_new_users_with_repository: true
  default_collection_replication: 1
EOF

(cd config && /usr/local/lib/arvbox/application_yml_override.py)

if ! test -f /var/lib/arvados/api_database_pw ; then
    ruby -e 'puts rand(2**128).to_s(36)' > /var/lib/arvados/api_database_pw
fi
database_pw=$(cat /var/lib/arvados/api_database_pw)

if ! (psql postgres -c "\du" | grep "^ arvados ") >/dev/null ; then
    psql postgres -c "create user arvados with password '$database_pw'"
    psql postgres -c "ALTER USER arvados CREATEDB;"
fi

sed "s/password:.*/password: $database_pw/" <config/database.yml.example >config/database.yml

if ! test -f /var/lib/arvados/api_database_setup ; then
   bundle exec rake db:setup
   touch /var/lib/arvados/api_database_setup
fi

if ! test -s /var/lib/arvados/superuser_token ; then
    bundle exec ./script/create_superuser_token.rb > /var/lib/arvados/superuser_token
fi

rm -rf tmp

bundle exec rake db:migrate
