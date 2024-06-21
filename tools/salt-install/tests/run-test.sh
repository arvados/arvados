#!/bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

export ARVADOS_API_TOKEN=__SYSTEM_ROOT_TOKEN__
export ARVADOS_API_HOST=__CLUSTER__.__DOMAIN__:__CONTROLLER_EXT_SSL_PORT__
export ARVADOS_API_HOST_INSECURE=true

set -o pipefail

# First, validate that the CA is installed and that we can query it with no errors.
if ! curl -s -o /dev/null https://${ARVADOS_API_HOST}/users/welcome?return_to=%2F; then
  echo "The Arvados CA was not correctly installed. Although some components will work,"
  echo "others won't. Please verify that the CA cert file was installed correctly and"
  echo "retry running these tests."
  exit 1
fi

# Then, run a basic diagnostics test.
echo "Running arvados-client diagnostics..."
if ! arvados-client diagnostics -internal-client; then
  echo "Diagnostics run FAILED, exiting"
  exit 1
fi

# https://doc.arvados.org/v2.0/install/install-jobs-image.html
echo "Creating Arvados Standard Docker Images project"
uuid_prefix=$(arv --format=uuid user current | cut -d- -f1)
project_uuid=$(arv --format=uuid group list --filters '[["name", "=", "Arvados Standard Docker Images"]]')

if [ "x${project_uuid}" = "x" ]; then
  project_uuid=$(arv --format=uuid group create --group "{\"owner_uuid\": \"${uuid_prefix}-tpzed-000000000000000\", \"group_class\":\"project\", \"name\":\"Arvados Standard Docker Images\"}")

  read -rd $'\000' newlink <<EOF; arv link create --link "${newlink}"
{
  "tail_uuid":"${uuid_prefix}-j7d0g-fffffffffffffff",
  "head_uuid":"${project_uuid}",
  "link_class":"permission",
  "name":"can_read"
}
EOF
fi

echo "Arvados project uuid is '${project_uuid}'"

# Create the initial user
echo "Creating initial user '__INITIAL_USER__'"
user_uuid=$(arv --format=uuid user list --filters '[["email", "=", "__INITIAL_USER_EMAIL__"], ["username", "=", "__INITIAL_USER__"]]')

if [ "x${user_uuid}" = "x" ]; then
  user_uuid=$(arv --format=uuid user create --user '{"email": "__INITIAL_USER_EMAIL__", "username": "__INITIAL_USER__"}')
  echo "Setting up user '__INITIAL_USER__'"
  arv user setup --uuid "${user_uuid}"
fi

echo "Activating user '__INITIAL_USER__'"
arv user update --uuid "${user_uuid}" --user '{"is_active": true}'

echo "Getting the user API TOKEN"
user_api_token=$(arv api_client_authorization list | jq -r ".items[] | select( .owner_uuid == \"${user_uuid}\" ).api_token" | head -1)

if [ "x${user_api_token}" = "x" ]; then
  echo "No existing token found for user '__INITIAL_USER__' (user_uuid: '${user_uuid}'). Creating token"
  user_api_token=$(arv api_client_authorization create --api-client-authorization "{\"owner_uuid\": \"${user_uuid}\"}" | jq -r .api_token)
fi

echo "API TOKEN FOR user '__INITIAL_USER__': '${user_api_token}'."

# Change to the user's token and run the workflow
echo "Switching to user '__INITIAL_USER__'"
export ARVADOS_API_TOKEN="${user_api_token}"

echo "Running test CWL workflow"
cwl-runner --debug hasher-workflow.cwl hasher-workflow-job.yml
