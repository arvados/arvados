# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ContainerAuthTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "container token validate, Running, regular auth" do
    get "/arvados/v1/containers/current",
      params: {:format => :json},
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}/#{containers(:running).uuid}"}
    # Container is Running, token can be used
    assert_response :success
    assert_equal containers(:running).uuid, json_response['uuid']
  end

  test "container token validate, Locked, runtime_token" do
    get "/arvados/v1/containers/current",
      params: {:format => :json},
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:container_runtime_token).token}/#{containers(:runtime_token).uuid}"}
    # Container is Running, token can be used
    assert_response :success
    assert_equal containers(:runtime_token).uuid, json_response['uuid']
  end

  test "container token validate, Cancelled, runtime_token" do
    put "/arvados/v1/containers/#{containers(:runtime_token).uuid}",
      params: {
        :format => :json,
        :container => {:state => "Cancelled"}
      },
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:system_user).token}"}
    assert_response :success
    get "/arvados/v1/containers/current",
      params: {:format => :json},
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:container_runtime_token).token}/#{containers(:runtime_token).uuid}"}
    # Container is Queued, token cannot be used
    assert_response 401
  end

  test "container token validate, Running, without optional portion" do
    get "/arvados/v1/containers/current",
      params: {:format => :json},
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:running_container_auth).token}"}
    # Container is Running, token can be used
    assert_response :success
    assert_equal containers(:running).uuid, json_response['uuid']
  end

  test "container token validate, Locked, runtime_token, without optional portion" do
    get "/arvados/v1/containers/current",
      params: {:format => :json},
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:container_runtime_token).token}"}
    # runtime_token without container uuid won't return 'current'
    assert_response 404
  end

  test "container token validate, wrong container uuid" do
    get "/arvados/v1/containers/current",
      params: {:format => :json},
      headers: {'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:container_runtime_token).token}/#{containers(:running).uuid}"}
    # Container uuid mismatch, token can't be used
    assert_response 401
  end
end
