require 'test_helper'

class UsersControllerTest < ActionController::TestCase

  test "valid token works in controller test" do
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

  test "show repositories lists linked as well as owned repositories" do
    get :manage_account, {}, session_for(:active)
    assert_response :success
    repos = assigns(:my_repositories)
    assert repos
    repo_writables = assigns(:repo_writable)
    assert_not_empty repo_writables, "repo_writables should not be empty"
    assert_includes repo_writables, api_fixture('repositories')['repository4']['uuid']  # writable by active
    assert_includes repo_writables, api_fixture('repositories')['repository2']['uuid']  # owned by active
  end

  test "request shell access" do
    user = api_fixture('users')['spectator']

    ActionMailer::Base.deliveries = []

    post :request_shell_access, {
      id: user['uuid'],
      format: 'js'
    }, session_for(:spectator)
    assert_response :success

    full_name = "#{user['first_name']} #{user['last_name']}"
    expected = "Shell account request from #{full_name} (#{user['email']}, #{user['uuid']})"
    found_email = 0
    ActionMailer::Base.deliveries.each do |email|
      if email.subject.include?(expected)
        found_email += 1
        break
      end
    end
    assert_equal 1, found_email, "Expected 1 email after requesting shell access"
  end

  [
    'admin',
    'active',
  ].each do |username|
    test "access users page as #{username} and verify show button is available" do
      admin_user = api_fixture('users','admin')
      active_user = api_fixture('users','active')
      get :index, {}, session_for(username)
      if username == 'admin'
        assert_match /<a href="\/projects\/#{admin_user['uuid']}">Home<\/a.*./, @response.body
        assert_match /<a href="\/projects\/#{active_user['uuid']}">Home<\/a.*./, @response.body
        assert_match /<a.*href="\/users\/#{admin_user['uuid']}".*Show<\/a.*./, @response.body
        assert_match /<a.*href="\/users\/#{active_user['uuid']}".*Show<\/a.*./, @response.body
        assert_includes @response.body, admin_user['email']
        assert_includes @response.body, active_user['email']
      else
        refute_match  /a href=.*Home<.*\/a.*./, @response.body
        refute_match /<a.*href="\/users\/#{admin_user['uuid']}".*Show<\/a.*./, @response.body
        assert_match /<a.*href="\/users\/#{active_user['uuid']}".*Show<\/a.*./, @response.body
        assert_includes @response.body, active_user['email']
      end
    end
  end

  [
    'admin',
    'active',
  ].each do |username|
    test "access settings drop down menu as #{username}" do
      admin_user = api_fixture('users','admin')
      active_user = api_fixture('users','active')
      get :show, {
        id: api_fixture('users')[username]['uuid']
      }, session_for(username)
      if username == 'admin'
        assert_includes @response.body, admin_user['email']
        refute_empty css_select('[id="system-menu"]')
      else
        assert_includes @response.body, active_user['email']
        assert_empty css_select('[id="system-menu"]')
      end
    end
  end
end
