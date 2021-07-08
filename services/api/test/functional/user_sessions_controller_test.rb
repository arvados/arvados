# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UserSessionsControllerTest < ActionController::TestCase

  test "redirect to joshid" do
    api_client_page = 'http://client.example.com/home'
    get :login, params: {return_to: api_client_page}
    # Not supported any more
    assert_response 404
  end

  test "send token when user is already logged in" do
    authorize_with :inactive
    api_client_page = 'http://client.example.com/home'
    get :login, params: {return_to: api_client_page}
    assert_response :redirect
    assert_equal(0, @response.redirect_url.index(api_client_page + '?'),
                 'Redirect url ' + @response.redirect_url +
                 ' should start with ' + api_client_page + '?')
    assert_not_nil assigns(:api_client)
  end

  test "login creates token without expiration by default" do
    assert_equal Rails.configuration.Login.TokenLifetime, 0
    authorize_with :inactive
    api_client_page = 'http://client.example.com/home'
    get :login, params: {return_to: api_client_page}
    assert_response :redirect
    assert_not_nil assigns(:api_client)
    assert_nil assigns(:api_client_auth).expires_at
  end

  test "login creates token with configured lifetime" do
    token_lifetime = 1.hour
    Rails.configuration.Login.TokenLifetime = token_lifetime
    authorize_with :inactive
    api_client_page = 'http://client.example.com/home'
    get :login, params: {return_to: api_client_page}
    assert_response :redirect
    assert_not_nil assigns(:api_client)
    api_client_auth = assigns(:api_client_auth)
    assert_in_delta(api_client_auth.expires_at,
                    api_client_auth.updated_at + token_lifetime,
                    1.second)
  end

  [[0, 1.hour, 1.hour],
  [1.hour, 2.hour, 1.hour],
  [2.hour, 1.hour, 1.hour],
  [2.hour, nil, 2.hour],
  ].each do |config_lifetime, request_lifetime, expect_lifetime|
    test "login with TokenLifetime=#{config_lifetime} and request has expires_at=#{ request_lifetime.nil? ? "nil" : request_lifetime }" do
      Rails.configuration.Login.TokenLifetime = config_lifetime
      expected_expiration_time =  Time.now() + expect_lifetime
      authorize_with :inactive
      @request.headers['Authorization'] = 'Bearer '+Rails.configuration.SystemRootToken
      if request_lifetime.nil?
        get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: ',https://app.example'}
      else
        get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com", expires_at: Time.now() + request_lifetime}, return_to: ',https://app.example'}
      end
      assert_response :redirect
      api_client_auth = assigns(:api_client_auth)
      assert_not_nil api_client_auth
      assert_not_nil assigns(:api_client)
      assert_in_delta(api_client_auth.expires_at,
                      expected_expiration_time,
                      1.second)
    end
  end

  test "login with remote param returns a salted token" do
    authorize_with :inactive
    api_client_page = 'http://client.example.com/home'
    remote_prefix = 'zbbbb'
    get :login, params: {return_to: api_client_page, remote: remote_prefix}
    assert_response :redirect
    api_client_auth = assigns(:api_client_auth)
    assert_not_nil api_client_auth
    assert_includes(@response.redirect_url, 'api_token='+api_client_auth.salted_token(remote: remote_prefix))
  end

  test "login with malformed remote param returns an error" do
    authorize_with :inactive
    api_client_page = 'http://client.example.com/home'
    remote_prefix = 'invalid_cluster_id'
    get :login, params: {return_to: api_client_page, remote: remote_prefix}
    assert_response 400
  end

  test "login to LoginCluster" do
    Rails.configuration.Login.LoginCluster = 'zbbbb'
    Rails.configuration.RemoteClusters['zbbbb'] = ConfigLoader.to_OrderedOptions({'Host' => 'zbbbb.example.com'})
    api_client_page = 'http://client.example.com/home'
    get :login, params: {return_to: api_client_page}
    assert_response :redirect
    assert_equal("https://zbbbb.example.com/login?return_to=http%3A%2F%2Fclient.example.com%2Fhome", @response.redirect_url)
    assert_nil assigns(:api_client)
  end

  test "don't go into redirect loop if LoginCluster is self" do
    Rails.configuration.Login.LoginCluster = 'zzzzz'
    api_client_page = 'http://client.example.com/home'
    get :login, params: {return_to: api_client_page}
    # Doesn't redirect, just fail.
    assert_response 404
  end

  test "controller cannot create session without SystemRootToken" do
    get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: ',https://app.example'}
    assert_response 401
  end

  test "controller cannot create session with wrong SystemRootToken" do
    @request.headers['Authorization'] = 'Bearer blah'
    get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: ',https://app.example'}
    assert_response 401
  end

  test "controller can create session using SystemRootToken" do
    @request.headers['Authorization'] = 'Bearer '+Rails.configuration.SystemRootToken
    get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: ',https://app.example'}
    assert_response :redirect
    api_client_auth = assigns(:api_client_auth)
    assert_not_nil api_client_auth
    assert_includes(@response.redirect_url, 'api_token='+api_client_auth.token)
  end
end
