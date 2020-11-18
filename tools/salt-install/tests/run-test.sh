#!/usr/bin/env /bin/bash
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

export ARVADOS_API_TOKEN=changemesystemroottoken
export ARVADOS_API_HOST=__CLUSTER__.__DOMAIN__:__HOST_SSL_PORT__
export ARVADOS_API_HOST_INSECURE=true


# https://doc.arvados.org/v2.0/install/install-jobs-image.html
echo "Creating Arvados Standard Docker Images project"
uuid_prefix=$(arv --format=uuid user current | cut -d- -f1)
project_uuid=$(arv --format=uuid group create --group "{\"owner_uuid\": \"${uuid_prefix}-tpzed-000000000000000\", \"group_class\":\"project\", \"name\":\"Arvados Standard Docker Images\"}")
echo "Arvados project uuid is '${project_uuid}'"
read -rd $'\000' newlink <<EOF; arv link create --link "${newlink}"
{
"tail_uuid":"${uuid_prefix}-j7d0g-fffffffffffffff",
"head_uuid":"${project_uuid}",
"link_class":"permission",
"name":"can_read"
}
EOF

echo "Uploading arvados/jobs' docker image to the project"
VERSION="2.1.1"
arv-keepdocker --pull arvados/jobs ${VERSION} --project-uuid ${project_uuid}

# Create the initial user
echo "Creating initial user ('__INITIAL_USER__')"
user=$(arv --format=uuid user create --user '{"email": "__INITIAL_USER_EMAIL__", "username": "__INITIAL_USER__"}')
echo "Setting up user ('__INITIAL_USER__')"
arv user setup --uuid ${user}
echo "Activating user '__INITIAL_USER__'"
arv user update --uuid ${user} --user '{"is_active": true}'

user_api_token=$(arv api_client_authorization create --api-client-authorization "{\"owner_uuid\": \"${user}\"}" | jq -r .api_token)

echo "Running test CWL workflow"
# Change to the user's token and run the workflow
export ARVADOS_API_TOKEN=${user_api_token}
cwl-runner hasher-workflow.cwl hasher-workflow-job.yml
