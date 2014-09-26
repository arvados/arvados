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

  test "show repositories with read, write, or manage permission" do
    get :manage_account, {}, session_for(:active)
    assert_response :success
    repos = assigns(:my_repositories)
    assert repos
    assert_not_empty repos, "my_repositories should not be empty"
    editables = repos.collect { |r| !!assigns(:repo_writable)[r.uuid] }
    assert_includes editables, true, "should have a writable repository"
    assert_includes editables, false, "should have a readonly repository"
  end
end
