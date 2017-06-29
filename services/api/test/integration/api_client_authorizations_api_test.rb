# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ApiClientAuthorizationsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "create system auth" do
    post "/arvados/v1/api_client_authorizations/create_system_auth", {:format => :json, :scopes => ['test'].to_json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin_trustedclient).api_token}"}
    assert_response :success
  end

  test "create token for different user" do
    post "/arvados/v1/api_client_authorizations", {
      :format => :json,
      :api_client_authorization => {
        :owner_uuid => users(:spectator).uuid
      }
    }, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin_trustedclient).api_token}"}
    assert_response :success

    get "/arvados/v1/users/current", {
      :format => :json
    }, {'HTTP_AUTHORIZATION' => "OAuth2 #{json_response['api_token']}"}
    @json_response = nil
    assert_equal users(:spectator).uuid, json_response['uuid']
  end

  test "refuse to create token for different user if not trusted client" do
    post "/arvados/v1/api_client_authorizations", {
      :format => :json,
      :api_client_authorization => {
        :owner_uuid => users(:spectator).uuid
      }
    }, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}
    assert_response 403
  end

  test "refuse to create token for different user if not admin" do
    post "/arvados/v1/api_client_authorizations", {
      :format => :json,
      :api_client_authorization => {
        :owner_uuid => users(:spectator).uuid
      }
    }, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active_trustedclient).api_token}"}
    assert_response 403
  end

end
