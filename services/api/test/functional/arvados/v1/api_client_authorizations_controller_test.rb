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

end
