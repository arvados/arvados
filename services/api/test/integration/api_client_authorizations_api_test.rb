# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ApiClientAuthorizationsApiTest < ActionDispatch::IntegrationTest
  include DbCurrentTime
  extend DbCurrentTime
  fixtures :all

  test "create system auth" do
    post "/arvados/v1/api_client_authorizations/create_system_auth",
      params: {:format => :json, :scopes => ['test'].to_json},
      headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin_trustedclient).api_token}"}
    assert_response :success
  end

  [:admin_trustedclient, :SystemRootToken].each do |tk|
    test "create token for different user using #{tk}" do
      if tk == :SystemRootToken
        token = "xyzzy-SystemRootToken"
        Rails.configuration.SystemRootToken = token
      else
        token = api_client_authorizations(tk).api_token
      end

      post "/arvados/v1/api_client_authorizations",
           params: {
             :format => :json,
             :api_client_authorization => {
               :owner_uuid => users(:spectator).uuid
             }
           },
           headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{token}"}
      assert_response :success

      get "/arvados/v1/users/current",
          params: {:format => :json},
          headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{json_response['api_token']}"}
      @json_response = nil
      assert_equal json_response['uuid'], users(:spectator).uuid
    end
  end

  test "System root token is system user" do
    token = "xyzzy-SystemRootToken"
    Rails.configuration.SystemRootToken = token
    get "/arvados/v1/users/current",
        params: {:format => :json},
        headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{token}"}
    assert_equal json_response['uuid'], system_user_uuid
  end

  test "refuse to create token for different user if not trusted client" do
    post "/arvados/v1/api_client_authorizations",
      params: {
        :format => :json,
        :api_client_authorization => {
          :owner_uuid => users(:spectator).uuid
        }
      },
      headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}
    assert_response 403
  end

  test "refuse to create token for different user if not admin" do
    post "/arvados/v1/api_client_authorizations",
      params: {
        :format => :json,
        :api_client_authorization => {
          :owner_uuid => users(:spectator).uuid
        }
      },
      headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active_trustedclient).api_token}"}
    assert_response 403
  end

  [nil, db_current_time + 2.hours].each do |desired_expiration|
    [false, true].each do |admin|
      test "expires_at gets clamped on #{admin ? 'admins' : 'non-admins'} when API.MaxTokenLifetime is set and desired expires_at #{desired_expiration.nil? ? 'is not set' : 'exceeds the limit'}" do
        Rails.configuration.API.MaxTokenLifetime = 1.hour
        token = api_client_authorizations(admin ? :admin_trustedclient : :active_trustedclient).api_token

        # Test token creation
        start_t = db_current_time
        post "/arvados/v1/api_client_authorizations",
             params: {
               :format => :json,
               :api_client_authorization => {
                 :owner_uuid => users(admin ? :admin : :active).uuid,
                 :expires_at => desired_expiration,
               }
             },
             headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{token}"}
        assert_response 200
        expiration_t = json_response['expires_at'].to_time
        if admin && desired_expiration
          assert_in_delta desired_expiration.to_f, expiration_t.to_f, 1
        else
          assert_in_delta (start_t + Rails.configuration.API.MaxTokenLifetime).to_f, expiration_t.to_f, 2
        end

        # Test token update
        previous_expiration = expiration_t
        token_uuid = json_response["uuid"]

        start_t = db_current_time
        patch "/arvados/v1/api_client_authorizations/#{token_uuid}",
            params: {
              :api_client_authorization => {
                :expires_at => desired_expiration
              }
            },
            headers: {'HTTP_AUTHORIZATION' => "OAuth2 #{token}"}
        assert_response 200
        expiration_t = json_response['expires_at'].to_time
        if admin && desired_expiration
          assert_in_delta desired_expiration.to_f, expiration_t.to_f, 1
        else
          assert_in_delta (start_t + Rails.configuration.API.MaxTokenLifetime).to_f, expiration_t.to_f, 2
        end
      end
    end
  end

  test "get current token using salted token" do
    salted = salt_token(fixture: :active, remote: 'abcde')
    get('/arvados/v1/api_client_authorizations/current',
        params: {remote: 'abcde'},
        headers: {'HTTP_AUTHORIZATION' => "Bearer #{salted}"})
    assert_response :success
    assert_equal(json_response['uuid'], api_client_authorizations(:active).uuid)
    assert_equal(json_response['scopes'], ['all'])
    assert_not_nil(json_response['expires_at'])
    assert_nil(json_response['api_token'])
  end
end
