# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UsersControllerTest < ActionController::TestCase

  test "valid token works in controller test" do
    get :index, params: {}, session: session_for(:active)
    assert_response :success
  end

  test "ignore previously valid token (for deleted user), don't crash" do
    get :activity, params: {}, session: session_for(:valid_token_deleted_user)
    assert_response :redirect
    assert_match /^#{Rails.configuration.Services.Workbench1.ExternalURL}users\/welcome/, @response.redirect_url
    assert_nil assigns(:my_jobs)
    assert_nil assigns(:my_ssh_keys)
  end

  test "expired token redirects to api server login" do
    assert Rails.configuration.Login.Test.Enable
    get :show, params: {
      id: api_fixture('users')['active']['uuid']
    }, session: session_for(:expired_trustedclient)
    assert_response :redirect
    assert_match /^#{Rails.configuration.Services.Workbench1.ExternalURL}users\/welcome/, @response.redirect_url
    assert_nil assigns(:my_jobs)
    assert_nil assigns(:my_ssh_keys)
  end

  test "show welcome page if no token provided" do
    get :index, params: {}
    assert_response :redirect
    assert_match /\/users\/welcome/, @response.redirect_url
  end

  test "'log in as user' feature uses a v2 token" do
    post :sudo, params: {
      id: api_fixture('users')['active']['uuid']
    }, session: session_for('admin_trustedclient')
    assert_response :redirect
    assert_match /api_token=v2%2F/, @response.redirect_url
  end

  test "request shell access" do
    user = api_fixture('users')['spectator']

    ActionMailer::Base.deliveries = []

    post :request_shell_access, params: {
      id: user['uuid'],
      format: 'js'
    }, session: session_for(:spectator)
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
      get :index, params: {}, session: session_for(username)
      if username == 'admin'
        assert_match /<a href="\/projects\/#{admin_user['uuid']}">Home<\/a>/, @response.body
        assert_match /<a href="\/projects\/#{active_user['uuid']}">Home<\/a>/, @response.body
        assert_match /href="\/users\/#{admin_user['uuid']}"><i class="fa fa-fw fa-user"><\/i> Show<\/a/, @response.body
        assert_match /href="\/users\/#{active_user['uuid']}"><i class="fa fa-fw fa-user"><\/i> Show<\/a/, @response.body
        assert_includes @response.body, admin_user['email']
        assert_includes @response.body, active_user['email']
      else
        refute_match  /Home<\/a>/, @response.body
        refute_match /href="\/users\/#{admin_user['uuid']}"><i class="fa fa-fw fa-user"><\/i> Show<\/a/, @response.body
        assert_match /href="\/users\/#{active_user['uuid']}"><i class="fa fa-fw fa-user"><\/i> Show<\/a/, @response.body
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
      get :show, params: {
        id: api_fixture('users')[username]['uuid']
      }, session: session_for(username)
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
