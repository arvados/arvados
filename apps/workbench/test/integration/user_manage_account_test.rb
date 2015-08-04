require 'integration_helper'

class UserManageAccountTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  # test manage_account page
  def verify_manage_account user
    if user['is_active']
      within('.navbar-fixed-top') do
        page.find("#notifications-menu").click
        within('.dropdown-menu') do
          assert_selector 'a', text: 'Virtual machines'
          assert_selector 'a', text: 'Repositories'
          assert_selector 'a', text: 'Current token'
          assert_selector 'a', text: 'SSH keys'
          find('a', text: 'SSH keys').click
        end
      end

      # now in SSH Keys page
      assert page.has_text?('Add new SSH key'), 'No text - Add SSH key'
      add_and_verify_ssh_key
    else  # inactive user
      within('.navbar-fixed-top') do
        page.find("#notifications-menu").click
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
        assert_text 'Public key does not appear to be a valid ssh-rsa or dsa public key'

        public_key_str = api_fixture('authorized_keys')['active']['public_key']
        page.find_field('public_key').set public_key_str
        page.find_field('name').set 'added_in_test'
        click_button 'Submit'
        assert_text 'Public key already exists in the database, use a different key.'

        new_key = SSHKey.generate
        page.find_field('public_key').set new_key.ssh_public_key
        page.find_field('name').set 'added_in_test'
        click_button 'Submit'
      end

      # key must be added. look for it in the refreshed page
      assert_text 'added_in_test'
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

  test "pipeline notification shown even though public pipelines exist" do
    skip "created_by doesn't work that way"
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
    visit page_with_token 'job_reader'
    click_link 'notifications-menu'
    assert_selector 'a', text: 'Click here to learn how to run an Arvados Crunch pipeline'
  end

  [
    ['job_reader', :ssh, :pipeline],
    ['active'],
  ].each do |user, *expect|
    test "manage account for #{user} with notifications #{expect.inspect}" do
      Rails.configuration.anonymous_user_token = false
      visit page_with_token(user)
      click_link 'notifications-menu'
      if expect.include? :ssh
        assert_selector('a', text: 'Click here to set up an SSH public key for use with Arvados')
        click_link('Click here to set up an SSH public key for use with Arvados')
        assert_selector('a', text: 'Add new SSH key')

        add_and_verify_ssh_key

        # No more SSH notification
        click_link 'notifications-menu'
        assert_no_selector('a', text: 'Click here to set up an SSH public key for use with Arvados')
      else
        assert_no_selector('a', text: 'Click here to set up an SSH public key for use with Arvados')
        assert_no_selector('a', text: 'Click here to learn how to run an Arvados Crunch pipeline')
      end

      if expect.include? :pipeline
        assert_selector('a', text: 'Click here to learn how to run an Arvados Crunch pipeline')
      end
    end
  end

  test "verify repositories for active user" do
    user = api_fixture('users')['active']
    visit page_with_token('active',"/users/#{api_fixture('users')['active']['uuid']}/repositories")
    repos = [[api_fixture('repositories')['foo'], true, true],
             [api_fixture('repositories')['repository3'], false, false],
             [api_fixture('repositories')['repository4'], true, false]]

    repos.each do |(repo, writable, sharable)|
      within('tr', text: repo['name']+'.git') do
        if sharable
          assert_selector 'a', text:'Share'
          assert_text 'writable'
        else
          assert_text repo['name']
          assert_no_selector 'a', text:'Share'
          if writable
            assert_text 'writable'
          else
            assert_text 'read-only'
          end
        end
      end
    end
  end

  test "request shell access" do
    user = api_fixture('users')['spectator']
    ActionMailer::Base.deliveries = []
    visit page_with_token('spectator', "/users/#{api_fixture('users')['spectator']['uuid']}/virtual_machines")
    assert_text 'You do not have access to any virtual machines'
    click_link 'Send request for shell access'

    # Button text changes to "sending...", then back to normal. In the
    # test suite we can't depend on confirming the "sending..." state
    # before it goes back to normal, though.
    ## assert_selector 'a', text: 'Sending request...'
    assert_selector 'a', text: 'Send request for shell access'
    assert_text 'A request for shell access was sent'

    # verify that the email was sent
    full_name = "#{user['first_name']} #{user['last_name']}"
    expected = "Shell account request from #{full_name} (#{user['email']}, #{user['uuid']})"
    found_email = 0
    ActionMailer::Base.deliveries.each do |email|
      if email.subject.include?(expected)
        found_email += 1
      end
    end
    assert_equal 1, found_email, "Expected email after requesting shell access"

    # Revisit the page and verify the request sent message along with
    # the request button.
    within('.navbar-fixed-top') do
      page.find("#notifications-menu").click
      within('.dropdown-menu') do
        find('a', text: 'Virtual machines').click
      end
    end
    assert_text 'You do not have access to any virtual machines.'
    assert_text 'A request for shell access was sent on '
    assert_selector 'a', text: 'Send request for shell access'
  end

  test "create new repository" do
    visit page_with_token("active_trustedclient")
    within('.navbar-fixed-top') do
      page.find("#notifications-menu").click
      within('.dropdown-menu') do
        assert_selector 'a', text: 'Repositories'
        find('a', text: 'Repositories').click
      end
    end
    click_on "Add new repository"
    within ".modal-dialog" do
      fill_in "Name", with: "workbenchtest"
      click_on "Create"
    end
    assert_text ":active/workbenchtest.git"
    assert_match /git@git.*:active\/workbenchtest.git/, page.text
    assert_match /https:\/\/git.*\/active\/workbenchtest.git/, page.text
  end

  [
    ['virtual_machines', nil, 'Host name', 'testvm2.shell'],
    ['repositories', 'Add new repository', 'It may take a minute or two before you can clone your new repository.', 'active/foo'],
    ['/current_token', nil, 'HISTIGNORE=$HISTIGNORE', 'ARVADOS_API_TOKEN=3kg6k6lzmp9kj5'],
    ['ssh_keys', 'Add new SSH key', 'Click here to learn about SSH keys in Arvados.', 'active'],
  ].each do |page_name, button_name, look_for, content|
    test "test user-settings menu for page #{page_name}" do
      if page_name == '/current_token'
        visit page_with_token('active', page_name)
      else
        visit page_with_token('active', "/users/#{api_fixture('users')['active']['uuid']}/#{page_name}")
      end

      assert page.has_text? content
      if button_name
        assert_selector 'a', text: button_name
        find('a', text: button_name).click
      end

      assert page.has_text? look_for
    end
  end

  [
    ['virtual_machines', 'You do not have access to any virtual machines.'],
    ['repositories', 'You do not seem to have access to any repositories.'],
    ['/current_token', 'HISTIGNORE=$HISTIGNORE'],
    ['ssh_keys', 'You have not yet set up an SSH public key for use with Arvados.'],
  ].each do |page_name, look_for|
    test "test user-settings menu for page #{page_name} when page is empty" do
      if page_name == '/current_token'
        visit page_with_token('user1_with_load', page_name)
      else
        visit page_with_token('admin', "/users/#{api_fixture('users')['user1_with_load']['uuid']}/#{page_name}")
      end

     assert page.has_text? look_for
    end
  end
end
