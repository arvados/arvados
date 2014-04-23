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

  test "admin search filters where scopes exactly match" do
    def check_tokens_by_scopes(scopes, *expected_tokens)
      expected_tokens.map! { |name| api_client_authorizations(name).api_token }
      get :index, where: {scopes: scopes}
      assert_response :success
      got_tokens = JSON.parse(@response.body)['items']
        .map { |auth| auth['api_token'] }
      assert_equal(expected_tokens.sort, got_tokens.sort,
                   "wrong results for scopes = #{scopes}")
    end
    authorize_with :admin_trustedclient
    check_tokens_by_scopes([], :admin_noscope)
    authorize_with :active_trustedclient
    check_tokens_by_scopes(["GET /arvados/v1/users"], :active_userlist)
    check_tokens_by_scopes(["POST /arvados/v1/api_client_authorizations",
                            "GET /arvados/v1/api_client_authorizations"],
                           :active_apitokens)
  end
end
