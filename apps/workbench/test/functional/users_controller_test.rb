require 'test_helper'

class UsersControllerTest < ActionController::TestCase
  test "valid token works in functional test" do
    get :index, {}, session_for(:active)
    assert_response :success
  end

  test "ignore previously valid token (for deleted user), don't crash" do
    get :activity, {}, session_for(:valid_token_deleted_user)
    assert_response :redirect
    assert_match /^#{Rails.configuration.arvados_login_base}/, @response.redirect_url
    assert_nil assigns(:my_jobs)
    assert_nil assigns(:my_ssh_keys)
  end

  test "expired token redirects to api server login" do
    get :show, {
      id: api_fixture('users')['active']['uuid']
    }, session_for(:expired_trustedclient)
    assert_response :redirect
    assert_match /^#{Rails.configuration.arvados_login_base}/, @response.redirect_url
    assert_nil assigns(:my_jobs)
    assert_nil assigns(:my_ssh_keys)
  end

  test "show welcome page if no token provided" do
    get :index, {}
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
  end
end
