require 'integration_helper'

class AnonymousAccessTest < ActionDispatch::IntegrationTest
  # These tests don't do state-changing API calls. Save some time by
  # skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  setup do
    need_javascript
  end

  def verify_homepage_anonymous_enabled user, is_active, has_profile
    if user
      if user['is_active']
        if has_profile
          assert_text 'Unrestricted public data'
          assert_selector 'a', text: 'Projects'
        else
          assert_text 'All required fields must be completed before you can proceed'
        end
      else
        assert_text 'indicate that you have read and accepted the user agreement'
      end
      within('.navbar-fixed-top') do
        assert_no_text 'You are viewing public data'
        assert_selector 'a', text: "#{user['email']}"
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          assert_selector 'a', text: 'Log out'
        end
      end
    else
      assert_text 'Unrestricted public data'
      within('.navbar-fixed-top') do
        assert_text 'You are viewing public data'
        anonymous_user = api_fixture('users')['anonymous']
        assert_selector 'a', "#{anonymous_user['email']}"
        find('a', text: "#{anonymous_user['email']}").click
        within('.dropdown-menu') do
          assert_selector 'a', text: 'Log in'
          assert_no_selector 'a', text: 'Log out'
        end
      end
    end
  end

  [
    [nil, nil, false, false],
    ['inactive', api_fixture('users')['inactive'], false, false],
    ['active', api_fixture('users')['active'], true, true],
    ['active_no_prefs_profile', api_fixture('users')['active_no_prefs_profile'], true, false],
    ['admin', api_fixture('users')['admin'], true, true],
  ].each do |token, user, is_active, has_profile|
    test "visit public project as user #{token} when anonymous browsing is enabled" do
      Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']

      path = "/projects/#{api_fixture('groups')['anonymously_accessible_project']['uuid']}/?public_data=true"

      if !token
        visit path
      else
        visit page_with_token(token, path)
      end
      verify_homepage_anonymous_enabled user, is_active, has_profile
    end
  end

  [
    [nil, nil],
    ['active', api_fixture('users')['active']],
  ].each do |token, user, is_active|
    test "visit public project as user #{token} when anonymous browsing is not enabled" do
      Rails.configuration.anonymous_user_token = false

      path = "/projects/#{api_fixture('groups')['anonymously_accessible_project']['uuid']}/?public_data=true"
      if !token
        visit path
      else
        visit page_with_token(token, path)
      end

      if user
        assert_text 'Unrestricted public data'
      else
        assert_text 'Please log in'
      end
    end
  end

  test "visit non-public project as anonymous when anonymous browsing is enabled and expect page not found" do
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
    visit "/projects/#{api_fixture('groups')['aproject']['uuid']}/?public_data=true"
    assert_text 'Not Found'
  end
end
