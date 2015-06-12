require 'integration_helper'

class UsersTest < ActionDispatch::IntegrationTest

  test "login as active user but not admin" do
    need_javascript
    visit page_with_token('active_trustedclient')

    assert page.has_no_link? 'Users' 'Found Users link for non-admin user'
  end

  test "login as admin user and verify active user data" do
    need_javascript
    visit page_with_token('admin_trustedclient')

    # go to Users list page
    find('#system-menu').click
    click_link 'Users'

    # check active user attributes in the list page
    page.within(:xpath, '//tr[@data-object-uuid="zzzzz-tpzed-xurymjxw79nv3jz"]') do
      assert (text.include? 'true false'), 'Expected is_active'
    end

    find('tr', text: 'zzzzz-tpzed-xurymjxw79nv3jz').
      find('a', text: 'Show').
      click
    assert page.has_text? 'Attributes'
    assert page.has_text? 'Advanced'
    assert page.has_text? 'Admin'

    # go to the Attributes tab
    click_link 'Attributes'
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//span[@data-name="is_active"]') do
      assert_equal "true", text, "Expected user's is_active to be true"
    end
    page.within(:xpath, '//span[@data-name="is_admin"]') do
      assert_equal "false", text, "Expected user's is_admin to be false"
    end

  end

  test "create a new user" do
    need_javascript

    visit page_with_token('admin_trustedclient')

    find('#system-menu').click
    click_link 'Users'

    assert page.has_text? 'zzzzz-tpzed-d9tiejq69daie8f'

    click_link 'Add a new user'

    within '.modal-content' do
      find 'label', text: 'Virtual Machine'
      fill_in "email", :with => "foo@example.com"
      fill_in "repo_name", :with => "newtestrepo"
      click_button "Submit"
      wait_for_ajax
    end

    visit '/users'

    # verify that the new user showed up in the users page and find
    # the new user's UUID
    new_user_uuid =
      find('tr[data-object-uuid]', text: 'foo@example.com')['data-object-uuid']
    assert new_user_uuid, "Expected new user uuid not found"

    # go to the new user's page
    find('tr', text: new_user_uuid).
      find('a', text: 'Show').
      click

    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//span[@data-name="is_active"]') do
      assert_equal "false", text, "Expected new user's is_active to be false"
    end

    click_link 'Advanced'
    click_link 'Metadata'
    assert page.has_text? 'Repository: foo/newtestrepo'
    assert !(page.has_text? 'VirtualMachine:')
  end

  test "setup the active user" do
    need_javascript
    visit page_with_token('admin_trustedclient')

    find('#system-menu').click
    click_link 'Users'

    # click on active user
    find('tr', text: 'zzzzz-tpzed-xurymjxw79nv3jz').
      find('a', text: 'Show').
      click
    user_url = page.current_url

    # Setup user
    click_link 'Admin'
    assert page.has_text? 'As an admin, you can setup'

    click_link 'Setup Active User'

    within '.modal-content' do
      find 'label', text: 'Virtual Machine'
      fill_in "repo_name", :with => "activetestrepo"
      click_button "Submit"
    end

    visit user_url
    assert page.has_text? 'modified_by_client_uuid'

    click_link 'Advanced'
    click_link 'Metadata'
    assert page.has_text? 'Repository: active/activetestrepo'
    assert !(page.has_text? 'VirtualMachine:')

    # Click on Setup button again and this time also choose a VM
    click_link 'Admin'
    click_link 'Setup Active User'

    within '.modal-content' do
      fill_in "repo_name", :with => "activetestrepo2"
      select("testvm.shell", :from => 'vm_uuid')
      fill_in "groups", :with => "test group one, test-group-two"
      click_button "Submit"
    end

    visit user_url
    find '#Attributes', text: 'modified_by_client_uuid'

    click_link 'Advanced'
    click_link 'Metadata'
    assert page.has_text? 'Repository: active/activetestrepo2'
    assert page.has_text? 'VirtualMachine: testvm.shell'
    assert page.has_text? '["test group one", "test-group-two"]'
  end

  test "unsetup active user" do
    need_javascript

    visit page_with_token('admin_trustedclient')

    find('#system-menu').click
    click_link 'Users'

    # click on active user
    find('tr', text: 'zzzzz-tpzed-xurymjxw79nv3jz').
      find('a', text: 'Show').
      click
    user_url = page.current_url

    # Verify that is_active is set
    find('a,button', text: 'Attributes').click
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//span[@data-name="is_active"]') do
      assert_equal "true", text, "Expected user's is_active to be true"
    end

    # go to Admin tab
    click_link 'Admin'
    assert page.has_text? 'As an admin, you can deactivate and reset this user'

    # unsetup user and verify all the above links are deleted
    click_link 'Admin'
    click_button 'Deactivate Active User'

    if Capybara.current_driver == :selenium
      sleep(0.1)
      page.driver.browser.switch_to.alert.accept
    else
      # poltergeist returns true for confirm(), so we don't need to accept.
    end

    # Should now be back in the Attributes tab for the user
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//span[@data-name="is_active"]') do
      assert_equal "false", text, "Expected user's is_active to be false after unsetup"
    end

    click_link 'Advanced'
    click_link 'Metadata'
    assert page.has_no_text? 'Repository: active/'
    assert page.has_no_text? 'VirtualMachine: testvm.shell'

    # setup user again and verify links present
    click_link 'Admin'
    click_link 'Setup Active User'

    within '.modal-content' do
      fill_in "repo_name", :with => "activetestrepo"
      select("testvm.shell", :from => 'vm_uuid')
      click_button "Submit"
    end

    visit user_url
    assert page.has_text? 'modified_by_client_uuid'

    click_link 'Advanced'
    click_link 'Metadata'
    assert page.has_text? 'Repository: active/activetestrepo'
    assert page.has_text? 'VirtualMachine: testvm.shell'
  end

  [
    ['admin', false],
    ['active', true],
  ].each do |username, expect_show_button|
    test "login as #{username} and access show button #{expect_show_button}" do
      need_javascript

      user = api_fixture('users', username)

      visit page_with_token(username, '/users')

      if expect_show_button
        within('tr', text: user['uuid']) do
          assert_text user['email']
          assert_selector 'a', text: 'Show'
          find('a', text: 'Show').click
        end
        assert_selector 'a', 'Data collections'
      else
        # no 'Show' button in the admin user's own row
        within('tr', text: user['uuid']) do
          assert_text user['email']
          assert_no_selector 'a', text: 'Show'
        end

        # but the admin user can access 'Show' button for other users
        active_user = api_fixture('users', 'active')
        within('tr', text: active_user['uuid']) do
          assert_text active_user['email']
          assert_selector 'a', text: 'Show'
          find('a', text: 'Show').click
          assert_selector 'a', 'Attributes'
        end
      end
    end
  end
end
