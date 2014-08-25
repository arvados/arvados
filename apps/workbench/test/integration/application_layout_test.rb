require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class ApplicationLayoutTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
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
        assert page.has_text?('My projects'), 'Not found text - My projects'
        assert page.has_text?('Projects shared with me'), 'Not found text - Project shared with me'
      end
    elsif invited
      assert page.has_text?('Please check the box below to indicate that you have read and accepted the user agreement'), 'Not found text - Please check the box below . . .'
    else
      assert page.has_text?('Your account is inactive'), 'Not found text - Your account is inactive'
    end

    within('.navbar-fixed-top') do
      if !user
        assert page.has_link?('Log in'), 'Not found link - Log in'
      else
        # my account menu
        assert page.has_link?("#{user['email']}"), 'Not found link - email'
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          if user['is_active']
            assert page.has_no_link?('Not active'), 'Found link - Not active'
            assert page.has_no_link?('Sign agreements'), 'Found link - Sign agreements'

            assert page.has_link?('Manage account'), 'No link - Manage account'

            if profile_config
              assert page.has_link?('Manage profile'), 'No link - Manage profile'
            else
              assert page.has_no_link?('Manage profile'), 'Found link - Manage profile'
            end
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
    if user && user['is_active']
      look_for_add_new = nil
      within('.navbar-fixed-top') do
        page.find("#system-menu").click
        if user['is_admin']
          within('.dropdown-menu') do
            assert page.has_text?('Groups'), 'No text - Groups'
            assert page.has_link?('Repositories'), 'No link - Repositories'
            assert page.has_link?('Virtual machines'), 'No link - Virtual machines'
            assert page.has_link?('SSH keys'), 'No link - SSH keys'
            assert page.has_link?('API tokens'), 'No link - API tokens'
            find('a', text: 'Users').click
            look_for_add_new = 'Add a new user'
          end
        else
          within('.dropdown-menu') do
            assert page.has_no_text?('Users'), 'Found text - Users'
            assert page.has_no_link?('Repositories'), 'Found link - Repositories'
            assert page.has_no_link?('Virtual machines'), 'Found link - Virtual machines'
            assert page.has_no_link?('SSH keys'), 'Found link - SSH keys'
            assert page.has_no_link?('API tokens'), 'Found link - API tokens'

            find('a', text: 'Groups').click
            look_for_add_new = 'Add a new group'
          end
        end
      end
      if look_for_add_new
        assert page.has_text? look_for_add_new
      end
    else
      assert page.has_no_link?('#system-menu'), 'Found link - system menu'
    end
  end

  # test manage_account page
  def verify_manage_account user
    if user && user['is_active']
      within('.navbar-fixed-top') do
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          find('a', text: 'Manage account').click
        end
      end

      # now in manage account page
      assert page.has_text?('Virtual Machines'), 'No text - Virtual Machines'
      assert page.has_text?('Repositories'), 'No text - Repositories'
      assert page.has_text?('SSH Keys'), 'No text - SSH Keys'
      assert page.has_text?('Current Token'), 'No text - Current Token'

      assert page.has_text?('The Arvados API token is a secret key that enables the Arvados SDKs to access Arvados'), 'No text - Arvados API token'

      click_link 'Add new SSH key'

      within '.modal-content' do
        assert page.has_text?('Public Key'), 'No text - Public Key'
        assert page.has_button?('Cancel'), 'No button - Cancel'
        assert page.has_button?('Submit'), 'No button - Submit'

        page.find_field('public_key').set 'first test with an incorrect ssh key value'
        click_button 'Submit'
        assert page.has_text?('Public key does not appear to be a valid ssh-rsa or dsa public key'), 'No text - Public key does not appear to be a valid'

        public_key_str = api_fixture('authorized_keys')['active']['public_key']
        page.find_field('public_key').set public_key_str
        page.find_field('name').set 'added_in_test'
        click_button 'Submit'
        assert page.has_text?('Public key already exists in the database, use a different key.'), 'No text - Public key already exists'

        new_key = SSHKey.generate
        page.find_field('public_key').set new_key.ssh_public_key
        page.find_field('name').set 'added_in_test'
        click_button 'Submit'
      end

      # key must be added. look for it in the refreshed page
      assert page.has_text?('added_in_test'), 'No text - added_in_test'
    end
  end

  [
    [nil, nil, false, false],
    ['inactive', api_fixture('users')['inactive'], true, false],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited'], false, false],
    ['active', api_fixture('users')['active'], true, true],
    ['admin', api_fixture('users')['admin'], true, true],
    ['active_no_prefs', api_fixture('users')['active_no_prefs'], true, false],
    ['active_no_prefs_profile', api_fixture('users')['active_no_prefs_profile'], true, false],
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
  end

  [
    ['active', api_fixture('users')['active']],
    ['admin', api_fixture('users')['admin']],
  ].each do |token, user|

    test "test system menu for user #{token}" do
      visit page_with_token(token)
      verify_system_menu user
    end

    test "test manage account for user #{token}" do
      visit page_with_token(token)
      verify_manage_account user
    end
  end
end
