require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class ApplicationLayoutTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium

    @user_profile_form_fields = Rails.configuration.user_profile_form_fields
  end

  teardown do
    Rails.configuration.user_profile_form_fields = @user_profile_form_fields
  end

  def verify_homepage_with_profile user, invited, has_profile
    profile_config = Rails.configuration.user_profile_form_fields

    if !user
      assert page.has_text?('Please log in'), 'Not found text - Please log in'
      assert page.has_text?('The "Log in" button below will show you a Google sign-in page'), 'Not found text - google sign in page'
      assert page.has_no_text?('My projects'), 'Found text - My projects'
      assert page.has_link?("Log in to #{Rails.configuration.site_name}"), 'Not found text - log in to'
    elsif profile_config && !has_profile && user['is_active']
      add_profile user
    elsif user['is_active']
      assert page.has_text?('My projects'), 'Not found text - My projects'
      assert page.has_text?('Projects shared with me'), 'Not found text - Project shared with me'
      assert page.has_no_text?('Save profile'), 'Found text - Save profile'
    elsif invited
      assert page.has_text?('Please check the box below to indicate that you have read and accepted the user agreement'), 'Not found text - Please check the box below . . .'
      assert page.has_no_text?('Save profile'), 'Found text - Save profile'
    else
      assert page.has_text?('Your account is inactive'), 'Not found text - Your account is inactive'
      assert page.has_no_text?('Save profile'), 'Found text - Save profile'
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
        assert page.has_link?('Show version / debugging info'), 'No link - Show version / debugging info'
        assert page.has_link?('Report a problem'), 'No link - Report a problem'

        # check show version info link
        click_link 'Show version / debugging info'
      end
    end

    within '.modal-content' do
      assert page.has_text?('Version / debugging info'), 'No text - Version / debugging info'
      assert page.has_no_text?('Report a problem'), 'Found text - Report a problem'
      assert page.has_text?('Server version'), 'No text - Server version'
      assert page.has_text?('Server restarted at'), 'No text - Server restarted at'
      assert page.has_text?('Workbench version'), 'No text - Workbench version'
      assert page.has_text?('Arvados base'), 'No text - Arvados base'
      assert page.has_text?('Additional info'), 'No text - Additional info'
      assert page.has_no_text?('Found a problem?'), 'Found text - Found a problem'
      assert page.has_button?('Close'), 'No button - Close'
      assert page.has_no_button?('Report issue'), 'Found button - Report issue'
      click_button 'Close'
    end

    # check report issue link
    within('.navbar-fixed-top') do
      page.find("#arv-help").click
      within('.dropdown-menu') do
        click_link 'Report a problem'
      end
    end

    within '.modal-content' do
      assert page.has_text?('Report a problem'), 'No text - Report a problem'
      assert page.has_no_text?('Version / debugging info'), 'Found text - Version / debugging info'
      assert page.has_text?('Server version'), 'No text - Server version'
      assert page.has_text?('Server restarted at'), 'No text - Server restarted at'
      assert page.has_text?('Workbench version'), 'No text - Workbench version'
      assert page.has_text?('Arvados base'), 'No text - Arvados base'
      assert page.has_text?('Additional info'), 'No text - Additional info'
      assert page.has_text?('Found a problem?'), 'No text - Found a problem'
      assert page.has_no_button?('Close'), 'Found button - Close'
      assert page.has_button?('Report issue'), 'No button - Report issue'
      assert page.has_button?('Cancel'), 'No button - Cancel'

      # enter a report text and click on report
      page.find_field('report_issue_text').set 'my test report text'
      click_button 'Report issue'

      # ajax success updated button texts and added footer message
      assert page.has_no_button?('Report issue'), 'Found button - Report issue'
      assert page.has_no_button?('Cancel'), 'Found button - Cancel'
      assert page.has_text?('Report sent'), 'No text - Report sent'
      assert page.has_button?('Close'), 'No text - Close'
      assert page.has_text?('Thanks for reporting this issue'), 'No text - Thanks for reporting this issue'

      click_button 'Close'
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

  # Check manage profile page and add missing profile to the user
  def add_profile user
    assert page.has_no_text?('My projects'), 'Found text - My projects'
    assert page.has_no_text?('Projects shared with me'), 'Found text - Projects shared with me'

    assert page.has_text?('Profile'), 'No text - Profile'
    assert page.has_text?('First name'), 'No text - First name'
    assert page.has_text?('Last name'), 'No text - Last name'
    assert page.has_text?('Identity URL'), 'No text - Identity URL'
    assert page.has_text?('Email'), 'No text - Email'
    assert page.has_text?(user['email']), 'No text - user email'

    # Using the default profile which has message and one required field

    # Save profile without filling in the required field. Expect to be back in this profile page again
    click_button "Save profile"
    assert page.has_text?('Profile'), 'No text - Profile'
    assert page.has_text?('First name'), 'No text - First name'
    assert page.has_text?('Last name'), 'No text - Last name'
    assert page.has_text?('Save profile'), 'No text - Save profile'

    # This time fill in required field and then save. Expect to go to requested page after that.
    profile_message = Rails.configuration.user_profile_form_message
    required_field_title = ''
    required_field_key = ''
    profile_config = Rails.configuration.user_profile_form_fields
    profile_config.andand.each do |entry|
      if entry['required']
        required_field_key = entry['key']
        required_field_title = entry['form_field_title']
      end
    end

    assert page.has_text?(profile_message[0,25]), 'No text - configured profile message'
    assert page.has_text?(required_field_title), 'No text - configured required field title'
    page.find_field('user[prefs][:profile][:'+required_field_key+']').set 'value to fill required field'

    click_button "Save profile"
    # profile saved and in profile page now with success
    assert page.has_text?('Thank you for filling in your profile'), 'No text - Thank you for filling'
    click_button 'Access Arvados Workbench'

    # profile saved and in home page now
    assert page.has_text?('My projects'), 'No text - My projects'
    assert page.has_text?('Projects shared with me'), 'No text - Projects shared with me'
  end

  # test the search box
  def verify_search_box user
    if user && user['is_active']
      # let's search for a valid uuid
      within('.navbar-fixed-top') do
        page.find_field('search').set user['uuid']
        page.find('.glyphicon-search').click
      end

      # we should now be in the user's page as a result of search
      assert page.has_text?(user['first_name']), 'No text - user first name'

      # let's search again for an invalid valid uuid
      within('.navbar-fixed-top') do
        search_for = String.new user['uuid']
        search_for[0]='1'
        page.find_field('search').set search_for
        page.find('.glyphicon-search').click
      end

      # we should see 'not found' error page
      assert page.has_text?('Not Found'), 'No text - Not Found'
      assert page.has_link?('Report problem'), 'No text - Report problem'
      click_link 'Report problem'
      within '.modal-content' do
        assert page.has_text?('Report a problem'), 'No text - Report a problem'
        assert page.has_no_text?('Version / debugging info'), 'No text - Version / debugging info'
        assert page.has_text?('Server version'), 'No text - Server version'
        assert page.has_text?('Server restarted at'), 'No text - Server restarted at'
        assert page.has_text?('Found a problem?'), 'No text - Found a problem'
        assert page.has_button?('Report issue'), 'No button - Report issue'
        assert page.has_button?('Cancel'), 'No button - Cancel'

        # enter a report text and click on report
        page.find_field('report_issue_text').set 'my test report text'
        click_button 'Report issue'

        # ajax success updated button texts and added footer message
        assert page.has_no_button?('Report issue'), 'Found button - Report issue'
        assert page.has_no_button?('Cancel'), 'Found button - Cancel'
        assert page.has_text?('Report sent'), 'No text - Report sent'
        assert page.has_button?('Close'), 'No text - Close'
        assert page.has_text?('Thanks for reporting this issue'), 'No text - Thanks for reporting this issue'

        click_button 'Close'
      end

      # let's search for the anonymously accessible project
      publicly_accessible_project = api_fixture('groups')['anonymously_accessible_project']

      within('.navbar-fixed-top') do
        # search again for the anonymously accessible project
        page.find_field('search').set publicly_accessible_project['name'][0,10]
        page.find('.glyphicon-search').click
      end

      within '.modal-content' do
        assert page.has_text?('All projects'), 'No text - All projects'
        assert page.has_text?('Search'), 'No text - Search'
        assert page.has_text?('Cancel'), 'No text - Cancel'
        assert_selector('div', text: publicly_accessible_project['name'])
        find(:xpath, '//div[./span[contains(.,publicly_accessible_project["uuid"])]]').click

        click_button 'Show'
      end

      # seeing "Unrestricted public data" now
      assert page.has_text?(publicly_accessible_project['name']), 'No text - publicly accessible project name'
      assert page.has_text?(publicly_accessible_project['description']), 'No text - publicly accessible project description'
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

    test "visit home page when profile is configured for user #{token}" do
      # Our test config enabled profile by default. So, no need to update config
      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      verify_homepage_with_profile user, invited, has_profile
    end

    test "visit home page when profile not configured for user #{token}" do
      Rails.configuration.user_profile_form_fields = false

      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      verify_homepage_with_profile user, invited, has_profile
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
    ['active', api_fixture('users')['active'], true, true],
    ['admin', api_fixture('users')['admin'], true, true],
  ].each do |token, user|

    test "test system menu for user #{token}" do
      visit page_with_token(token)
      verify_system_menu user
    end

    test "test manage account for user #{token}" do
      visit page_with_token(token)
      verify_manage_account user
    end

    test "test search for user #{token}" do
      visit page_with_token(token)
      verify_search_box user
    end

  end

end
