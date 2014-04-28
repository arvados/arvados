require 'test_helper'

class Arvados::V1::ApiClientAuthorizationsControllerTest < ActionController::TestCase
  test "should get index" do
    authorize_with :active_trustedclient
    get :index
    assert_response :success
  end

  test "should not get index with expired auth" do
    authorize_with :expired
    get :index, format: :json
    assert_response 401
  end

  test "should not get index from untrusted client" do
    authorize_with :active
    get :index
    assert_response 403
  end

  test "create system auth" do
    authorize_with :admin_trustedclient
    post :create_system_auth, scopes: '["test"]'
    assert_response :success
  end

  test "prohibit create system auth with token from non-trusted client" do
    authorize_with :admin
    post :create_system_auth, scopes: '["test"]'
    assert_response 403
  end

  test "prohibit create system auth by non-admin" do
    authorize_with :active
    post :create_system_auth, scopes: '["test"]'
    assert_response 403
  end

  def assert_found_tokens(auth, search_params, *expected_tokens)
    authorize_with auth
    expected_tokens.map! { |name| api_client_authorizations(name).api_token }
    get :index, search_params
    assert_response :success
    got_tokens = JSON.parse(@response.body)['items']
      .map { |auth| auth['api_token'] }
    assert_equal(expected_tokens.sort, got_tokens.sort,
                 "wrong results for #{search_params.inspect}")
  end

  # Three-tuples with auth to use, scopes to find, and expected tokens.
  # Make two tests for each tuple, one searching with where and the other
  # with filter.
  [[:admin_trustedclient, [], :admin_noscope],
   [:active_trustedclient, ["GET /arvados/v1/users"], :active_userlist],
   [:active_trustedclient,
    ["POST /arvados/v1/api_client_authorizations",
     "GET /arvados/v1/api_client_authorizations"],
    :active_apitokens],
  ].each do |auth, scopes, *expected|
    test "#{auth.to_s} can find auths where scopes=#{scopes.inspect}" do
      assert_found_tokens(auth, {where: {scopes: scopes}}, *expected)
    end

    test "#{auth.to_s} can find auths filtered with scopes=#{scopes.inspect}" do
      assert_found_tokens(auth, {filters: [['scopes', '=', scopes]]}, *expected)
    end
  end
end
