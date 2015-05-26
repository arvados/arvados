require 'integration_helper'

class ApplicationLayoutTest < ActionDispatch::IntegrationTest
  # These tests don't do state-changing API calls. Save some time by
  # skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  setup do
    need_javascript
  end

  def verify_homepage user, invited, has_profile
    profile_config = Rails.configuration.user_profile_form_fields

    if !user
      assert page.has_text?('Please log in'), 'Not found text - Please log in'
      assert page.has_text?('The "Log in" button below will show you a Google sign-in page'), 'Not found text - google sign in page'
      assert page.has_no_text?('My projects'), 'Found text - My projects'
      assert page.has_link?("Log in to #{Rails.configuration.site_name}"), 'Not found text - log in to'
    elsif user['is_active']
      if profile_config && !has_profile
        assert page.has_text?('Save profile'), 'No text - Save profile'
      else
        assert page.has_link?("Projects"), 'Not found link - Projects'
        page.find("#projects-menu").click
        assert_selector 'a', text: 'Add a new project'
        assert_no_selector 'a', text: 'Browse public projects'  # anonymous config is not enabled by default
        assert page.has_text?('Projects shared with me'), 'Not found text - Project shared with me'
      end
    elsif invited
      assert page.has_text?('Please check the box below to indicate that you have read and accepted the user agreement'), 'Not found text - Please check the box below . . .'
    else
      assert page.has_text?('Your account is inactive'), 'Not found text - Your account is inactive'
    end

    within('.navbar-fixed-top') do
      if !user
        assert_text Rails.configuration.site_name.downcase
        assert_no_selector 'a', text: Rails.configuration.site_name.downcase
        assert page.has_link?('Log in'), 'Not found link - Log in'
      else
        # my account menu
        assert_selector 'a', text: Rails.configuration.site_name.downcase
        assert(page.has_link?("notifications-menu"), 'no user menu')
        page.find("#notifications-menu").click
        within('.dropdown-menu') do
          if user['is_active']
            assert page.has_no_link?('Not active'), 'Found link - Not active'
            assert page.has_no_link?('Sign agreements'), 'Found link - Sign agreements'

            assert_selector "a[href=\"/projects/#{user['uuid']}\"]", text: 'Home project'
            assert page.has_link?('Manage account'), 'No link - Manage account'

            if profile_config
              assert page.has_link?('Manage profile'), 'No link - Manage profile'
            else
              assert page.has_no_link?('Manage profile'), 'Found link - Manage profile'
            end
          else
            assert_no_selector 'a', text: 'Home project'
            assert page.has_no_link?('Manage account'), 'Found link - Manage account'
            assert page.has_no_link?('Manage profile'), 'Found link - Manage profile'
          end
          assert page.has_link?('Log out'), 'No link - Log out'
        end
      end
    end
  end

  # test the help menu
  def check_help_menu
    within('.navbar-fixed-top') do
      page.find("#arv-help").click
      within('.dropdown-menu') do
        assert_selector 'a', text:'Getting Started ...'
        assert_selector 'a', text:'Public Pipelines and Data sets'
        assert page.has_link?('Tutorials and User guide'), 'No link - Tutorials and User guide'
        assert page.has_link?('API Reference'), 'No link - API Reference'
        assert page.has_link?('SDK Reference'), 'No link - SDK Reference'
        assert page.has_link?('Show version / debugging info ...'), 'No link - Show version / debugging info'
        assert page.has_link?('Report a problem ...'), 'No link - Report a problem'
        # Version info and Report a problem are tested in "report_issue_test.rb"
      end
    end
  end

  def verify_system_menu user
    if user && user['is_admin']
      assert page.has_link?('system-menu'), 'No link - system menu'
      within('.navbar-fixed-top') do
        page.find("#system-menu").click
        within('.dropdown-menu') do
          assert page.has_text?('Groups'), 'No text - Groups'
          assert page.has_link?('Repositories'), 'No link - Repositories'
          assert page.has_link?('Virtual machines'), 'No link - Virtual machines'
          assert page.has_link?('SSH keys'), 'No link - SSH keys'
          assert page.has_link?('API tokens'), 'No link - API tokens'
          find('a', text: 'Users').click
        end
      end
      assert page.has_text? 'Add a new user'
    else
      assert page.has_no_link?('system-menu'), 'Found link - system menu'
    end
  end

  [
    [nil, nil, false, false],
    ['inactive', api_fixture('users')['inactive'], true, false],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited'], false, false],
    ['active', api_fixture('users')['active'], true, true],
    ['admin', api_fixture('users')['admin'], true, true],
    ['active_no_prefs', api_fixture('users')['active_no_prefs'], true, false],
    ['active_no_prefs_profile_no_getting_started_shown',
        api_fixture('users')['active_no_prefs_profile_no_getting_started_shown'], true, false],
  ].each do |token, user, invited, has_profile|

    test "visit home page for user #{token}" do
      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      verify_homepage user, invited, has_profile
    end

    test "check help for user #{token}" do
      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      check_help_menu
    end

    test "test system menu for user #{token}" do
      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      verify_system_menu user
    end
  end

  test "test getting started help menu item" do
    visit page_with_token('active')
    within '.navbar-fixed-top' do
      find('.help-menu > a').click
      find('.help-menu .dropdown-menu a', text: 'Getting Started ...').click
    end

    within '.modal-content' do
      assert_text 'Getting Started'
      assert_selector 'button:not([disabled])', text: 'Next'
      assert_no_selector 'button:not([disabled])', text: 'Prev'

      # Use Next button to enable Prev button
      click_button 'Next'
      assert_selector 'button:not([disabled])', text: 'Prev'  # Prev button is now enabled
      click_button 'Prev'
      assert_no_selector 'button:not([disabled])', text: 'Prev'  # Prev button is again disabled

      # Click Next until last page is reached and verify that it is disabled
      (0..20).each do |i|   # currently we only have 4 pages, and don't expect to have more than 20 in future
        click_button 'Next'
        begin
          find('button:not([disabled])', text: 'Next')
        rescue => e
          break
        end
      end
      assert_no_selector 'button:not([disabled])', text: 'Next'  # Next button is disabled
      assert_selector 'button:not([disabled])', text: 'Prev'     # Prev button is enabled
      click_button 'Prev'
      assert_selector 'button:not([disabled])', text: 'Next'     # Next button is now enabled

      first('button', text: 'x').click
    end
    assert_text 'Active pipelines' # seeing dashboard now
  end

  test "test arvados_public_data_doc_url config unset" do
    Rails.configuration.arvados_public_data_doc_url = false

    visit page_with_token('active')
    within '.navbar-fixed-top' do
      find('.help-menu > a').click

      assert_no_selector 'a', text:'Public Pipelines and Data sets'

      assert_selector 'a', text:'Getting Started ...'
      assert page.has_link?('Tutorials and User guide'), 'No link - Tutorials and User guide'
      assert page.has_link?('API Reference'), 'No link - API Reference'
      assert page.has_link?('SDK Reference'), 'No link - SDK Reference'
      assert page.has_link?('Show version / debugging info ...'), 'No link - Show version / debugging info'
      assert page.has_link?('Report a problem ...'), 'No link - Report a problem'
    end
  end
end
