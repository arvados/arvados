# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ApiClientAuthorizationsControllerTest < ActionController::TestCase
  test "should get index" do
    authorize_with :active_trustedclient
    get :index
    assert_response :success
  end

  test "should not get index with expired auth" do
    authorize_with :expired
    get :index, params: {format: :json}
    assert_response 401
  end

  test "should not get index from untrusted client" do
    authorize_with :active
    get :index
    assert_response 403
  end

  test "create system auth" do
    authorize_with :admin_trustedclient
    post :create_system_auth, params: {scopes: '["test"]'}
    assert_response :success
    assert_not_nil JSON.parse(@response.body)['uuid']
  end

  test "prohibit create system auth with token from non-trusted client" do
    authorize_with :admin
    post :create_system_auth, params: {scopes: '["test"]'}
    assert_response 403
  end

  test "prohibit create system auth by non-admin" do
    authorize_with :active
    post :create_system_auth, params: {scopes: '["test"]'}
    assert_response 403
  end

  def assert_found_tokens(auth, search_params, expected)
    authorize_with auth
    expected_tokens = expected.map do |name|
      api_client_authorizations(name).api_token
    end
    get :index, params: search_params
    assert_response :success
    got_tokens = JSON.parse(@response.body)['items']
      .map { |a| a['api_token'] }
    assert_equal(expected_tokens.sort, got_tokens.sort,
                 "wrong results for #{search_params.inspect}")
  end

  # Three-tuples with auth to use, scopes to find, and expected tokens.
  # Make two tests for each tuple, one searching with where and the other
  # with filter.
  [[:admin_trustedclient, [], [:admin_noscope]],
   [:active_trustedclient, ["GET /arvados/v1/users"], [:active_userlist]],
   [:active_trustedclient,
    ["POST /arvados/v1/api_client_authorizations",
     "GET /arvados/v1/api_client_authorizations"],
    [:active_apitokens]],
  ].each do |auth, scopes, expected|
    test "#{auth.to_s} can find auths where scopes=#{scopes.inspect}" do
      assert_found_tokens(auth, {where: {scopes: scopes}}, expected)
    end

    test "#{auth.to_s} can find auths filtered with scopes=#{scopes.inspect}" do
      assert_found_tokens(auth, {filters: [['scopes', '=', scopes]]}, expected)
    end

    test "#{auth.to_s} offset works with filter scopes=#{scopes.inspect}" do
      assert_found_tokens(auth, {
                            offset: expected.length,
                            filters: [['scopes', '=', scopes]]
                          }, [])
    end
  end

  [:admin, :active].each do |token|
    test "using '#{token}', get token details via 'current'" do
      authorize_with token
      get :current
      assert_response 200
      assert_equal json_response['scopes'], ['all']
    end
  end

  [# anyone can look up the token they're currently using
   [:admin, :admin, 200, 200, 1],
   [:active, :active, 200, 200, 1],
   # cannot look up other tokens (even for same user) if not trustedclient
   [:admin, :active, 403, 403],
   [:admin, :admin_vm, 403, 403],
   [:active, :admin, 403, 403],
   # cannot look up other tokens for other users, regardless of trustedclient
   [:admin_trustedclient, :active, 404, 200, 0],
   [:active_trustedclient, :admin, 404, 200, 0],
  ].each do |user, token, expect_get_response, expect_list_response, expect_list_items|
    test "using '#{user}', get '#{token}' by uuid" do
      authorize_with user
      get :show, params: {
        id: api_client_authorizations(token).uuid,
      }
      assert_response expect_get_response
    end

    test "using '#{user}', update '#{token}' by uuid" do
      authorize_with user
      put :update, params: {
        id: api_client_authorizations(token).uuid,
        api_client_authorization: {},
      }
      assert_response expect_get_response
    end

    test "using '#{user}', delete '#{token}' by uuid" do
      authorize_with user
      post :destroy, params: {
        id: api_client_authorizations(token).uuid,
      }
      assert_response expect_get_response
    end

    test "using '#{user}', list '#{token}' by uuid" do
      authorize_with user
      get :index, params: {
        filters: [['uuid','=',api_client_authorizations(token).uuid]],
      }
      assert_response expect_list_response
      if expect_list_items
        assert_equal assigns(:objects).length, expect_list_items
        assert_equal json_response['items_available'], expect_list_items
      end
    end

    if expect_list_items
      test "using '#{user}', list '#{token}' by uuid with offset" do
        authorize_with user
        get :index, params: {
          filters: [['uuid','=',api_client_authorizations(token).uuid]],
          offset: expect_list_items,
        }
        assert_response expect_list_response
        assert_equal json_response['items_available'], expect_list_items
        assert_equal json_response['items'].length, 0
      end
    end

    test "using '#{user}', list '#{token}' by token" do
      authorize_with user
      get :index, params: {
        filters: [['api_token','=',api_client_authorizations(token).api_token]],
      }
      assert_response expect_list_response
      if expect_list_items
        assert_equal assigns(:objects).length, expect_list_items
        assert_equal json_response['items_available'], expect_list_items
      end
    end
  end

  test "scoped token cannot change its own scopes" do
    authorize_with :admin_vm
    put :update, params: {
      id: api_client_authorizations(:admin_vm).uuid,
      api_client_authorization: {scopes: ['all']},
    }
    assert_response 403
  end

  test "token cannot change its own uuid" do
    authorize_with :admin
    put :update, params: {
      id: api_client_authorizations(:admin).uuid,
      api_client_authorization: {uuid: 'zzzzz-gj3su-zzzzzzzzzzzzzzz'},
    }
    assert_response 403
  end

  test "get current token" do
    authorize_with :active
    get :current
    assert_response :success
    assert_equal(json_response['api_token'],
                 api_client_authorizations(:active).api_token)
  end

  test "get current token using SystemRootToken" do
    Rails.configuration.SystemRootToken = "xyzzy-systemroottoken"
    authorize_with_token Rails.configuration.SystemRootToken
    get :current
    assert_response :success
    assert_equal(Rails.configuration.SystemRootToken, json_response['api_token'])
    assert_not_empty(json_response['uuid'])
  end

  test "get current token, no auth" do
    get :current
    assert_response 401
  end
end
