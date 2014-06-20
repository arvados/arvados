require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class AnonymousUserTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium

    @anonymous_token = Rails.configuration.anonymous_user_token
  end

  teardown do
    Rails.configuration.anonymous_user_token = @anonymous_token
  end

  def verify_homepage_anonymous_login_configured user, invited
    within('.navbar-fixed-top') do
     if user && user['is_active']
        assert page.has_no_text? 'You are viewing public data'
        assert page.has_no_link? "Log in"
        assert page.has_link? "#{user['email']}"
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          page.has_link? ('Logout')
          page.has_no_link? ('Inactive')
          page.has_link? ('Manage ssh keys')
          page.has_link? ('Manage API tokens')
        end
      else
        assert page.has_text? 'You are viewing public data'
        if !user
          assert page.has_link? "Log in"
        else
          assert page.has_no_link? 'Log in'
          assert page.has_link? "#{user['email']}"
          find('a', text: "#{user['email']}").click
          within('.dropdown-menu') do
            page.has_link? ('Logout')
            page.has_link? ('Inactive')
            page.has_no_link? ('Manage ssh keys')
            page.has_no_link? ('Manage API tokens')
          end
        end
      end
    end

    assert page.has_text? 'Projects shared with me'
    assert page.has_text? 'A Project'
    assert page.has_text? 'Unrestricted public data'

    if user && user['is_active']
      assert page.has_no_text? 'After you assure Google that you want to log in here with your Google account'
      assert page.has_no_text? 'Please indicate that you have read and accepted the user agreements'
      assert page.has_no_text? 'You account is inactive'
      assert page.has_no_text? 'Welcome'
      assert page.has_text? 'My projects'
      assert page.has_button? 'Add new project'
    else
      assert page.has_text? 'Welcome'
      assert page.has_no_text? 'My projects'
      assert page.has_no_button? 'Add new project'
      if !user
        assert page.has_text? 'After you assure Google that you want to log in here with your Google account'
      elsif invited
        assert page.has_text? 'Please indicate that you have read and accepted the user agreements'
      else
        assert page.has_text? 'Your account is inactive'
      end
    end

    find('.arv-project-list a,button', text: 'Unrestricted public data').click
    page.has_text? ('An anonymously accessible project')

    find('a', text: 'Projects').click
    within('.dropdown-menu') do
      if user && user['is_active']
        page.has_text? ('New project')
      else
        page.has_no_text? ('New project')
      end
      page.has_text? ('Projects shared with me')
    end

    assert page.has_text? 'A Project'
    find('a', text: 'A Project').click
    page.has_text? ('Test project belonging to active user')

    #find('tr[data-kind="arvados#pipelineInstance"]', text: 'New pipeline instance').
    #  find('a', text: 'Show').click
  end

  def verify_homepage_anonymous_login_not_configured user, invited
    within('.navbar-fixed-top') do
      assert page.has_no_text? 'You are viewing public data'
      if !user
        assert page.has_link? 'Log in'
      else
        assert page.has_link? "#{user['email']}"
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          page.has_link? ('Logout')
          page.has_no_link? ('Inactive')
        end
      end
    end

    if !user
      assert page.has_text? 'Please log in'
      assert page.has_text? 'The "Log in" button below will show you a Google sign-in page'
      assert page.has_no_text? 'My projects'
      assert page.has_link? "Log in to #{Rails.configuration.site_name}"
    elsif user['is_active']
      assert page.has_text? 'My projects'
      assert page.has_text? 'Projects shared with me'
    elsif invited
      assert page.has_text? 'Please check the box below to indicate that you have read and accepted the user agreement'
    else
      assert page.has_text? 'Your account is inactive'
    end
  end

  [
    [nil, nil, false],
    ['anonymous', nil, false],
    ['inactive', api_fixture('users')['inactive'], true],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited'], false],
    ['active', api_fixture('users')['active'], true]
  ].each do |token, user, invited|
    test "visit home page when anonymous login configured for user #{token}" do
      Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']

      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end
      verify_homepage_anonymous_login_configured user, invited
    end
  end

  [
    [nil, nil, false],
    ['inactive', api_fixture('users')['inactive'], true],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited'], false],
    ['active', api_fixture('users')['active'], true]
  ].each do |token, user, invited|
    test "visit home page when anonymous login configured with bogus token for user #{token}" do
      Rails.configuration.anonymous_user_token = false

      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end
      verify_homepage_anonymous_login_not_configured user, invited
    end
  end

  [
    [nil, nil, false],
    ['inactive', api_fixture('users')['inactive'], true],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited'], false],
    ['active', api_fixture('users')['active'], true]
  ].each do |token, user, invited|
    test "visit home page when anonymous login not configured for user #{token}" do
      Rails.configuration.anonymous_user_token = false

      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end
      verify_homepage_anonymous_login_not_configured user, invited
    end
  end

end
