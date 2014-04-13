require 'test_helper'

class UsersControllerTest < ActionController::TestCase
  test "valid token for deleted user ignored instead of crashing" do
    skip
    get :welcome, {}, session_for(:valid_token_deleted_user)
    assert_response :success
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
end
