# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ContainerDispatchTest < ActionDispatch::IntegrationTest
  test "lock container with SystemRootToken" do
    Rails.configuration.SystemRootToken = "xyzzy-SystemRootToken"
    authheaders = {'HTTP_AUTHORIZATION' => "Bearer "+Rails.configuration.SystemRootToken}
    get("/arvados/v1/api_client_authorizations/current",
        headers: authheaders)
    assert_response 200

    system_auth_uuid = json_response['uuid']
    post("/arvados/v1/containers/#{containers(:queued).uuid}/lock",
         headers: authheaders)
    assert_response 200
    assert_equal system_auth_uuid, Container.find_by_uuid(containers(:queued).uuid).locked_by_uuid

    get("/arvados/v1/containers",
        params: {filters: SafeJSON.dump([['locked_by_uuid', '=', system_auth_uuid]])},
        headers: authheaders)
    assert_response 200
    assert_equal containers(:queued).uuid, json_response['items'][0]['uuid']
    assert_equal system_auth_uuid, json_response['items'][0]['locked_by_uuid']

    post("/arvados/v1/containers/#{containers(:queued).uuid}/unlock",
         headers: authheaders)
    assert_response 200
  end
end
