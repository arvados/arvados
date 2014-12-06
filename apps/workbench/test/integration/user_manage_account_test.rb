require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class UserManageAccountTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  # test manage_account page
  def verify_manage_account user
    if user['is_active']
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
      add_and_verify_ssh_key
    else  # inactive user
      within('.navbar-fixed-top') do
        find('a', text: "#{user['email']}").click
        within('.dropdown-menu') do
          assert page.has_no_link?('Manage profile'), 'Found link - Manage profile'
        end
      end
    end
  end

  def add_and_verify_ssh_key
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

  [
    ['inactive', api_fixture('users')['inactive']],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited']],
    ['active', api_fixture('users')['active']],
    ['admin', api_fixture('users')['admin']],
  ].each do |token, user|
    test "test manage account for user #{token}" do
      visit page_with_token(token)
      verify_manage_account user
    end
  end

  [
    ['inactive_but_signed_user_agreement', true],
    ['active', false],
  ].each do |user, notifications|
    test "test manage account for #{user} with notifications #{notifications}" do
      visit page_with_token(user)
      click_link 'notifications-menu'
      if notifications
        assert_selector('a', text: 'Click here to set up an SSH public key for use with Arvados')
        assert_selector('a', text: 'Click here to learn how to run an Arvados Crunch pipeline')
        click_link('Click here to set up an SSH public key for use with Arvados')
        assert_selector('a', text: 'Add new SSH key')

        add_and_verify_ssh_key

        # No more SSH notification
        click_link 'notifications-menu'
        assert_no_selector('a', text: 'Click here to set up an SSH public key for use with Arvados')
        assert_selector('a', text: 'Click here to learn how to run an Arvados Crunch pipeline')
      else
        assert_no_selector('a', text: 'Click here to set up an SSH public key for use with Arvados')
        assert_no_selector('a', text: 'Click here to learn how to run an Arvados Crunch pipeline')
      end
    end
  end
end
