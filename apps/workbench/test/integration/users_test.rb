require 'integration_helper'
require "selenium-webdriver"
#require 'headless'

class UsersTest < ActionDispatch::IntegrationTest

  test "login as active user but not admin" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('active_trustedclient')

    assert page.has_no_link? 'Users' 'Found Users link for non-admin user'
  end

  test "login as admin user and verify active user data" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('admin_trustedclient')

    # go to Users list page
    click_link 'Users'

    # check active user attributes in the list page
    page.within(:xpath, '//tr[@data-object-uuid="zzzzz-tpzed-xurymjxw79nv3jz"]') do
      assert (text.include? 'true false'), 'Expected is_active'
    end

    click_link 'zzzzz-tpzed-xurymjxw79nv3jz'
    assert page.has_text? 'Attributes'
    assert page.has_text? 'Metadata'
    assert page.has_text? 'Admin'

    # go to the Attributes tab
    click_link 'Attributes'
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//a[@data-name="is_active"]') do
      assert_equal "true", text, "Expected user's is_active to be true"
    end
    page.within(:xpath, '//a[@data-name="is_admin"]') do
      assert_equal "false", text, "Expected user's is_admin to be false"
    end

  end

  test "create a new user" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('admin_trustedclient')

    click_link 'Users'

    assert page.has_text? 'zzzzz-tpzed-d9tiejq69daie8f'

    click_link 'Add a new user'
    
    # for now just check that we are back in Users -> List page
    assert page.has_text? 'zzzzz-tpzed-d9tiejq69daie8f'
  end

  #@headless
  test "unsetup active user" do
    Capybara.current_driver = Capybara.javascript_driver
    #Capybara.current_driver = :selenium

    visit page_with_token('admin_trustedclient')

    click_link 'Users'

    assert page.has_link? 'zzzzz-tpzed-xurymjxw79nv3jz'

    # click on active user
    click_link 'zzzzz-tpzed-xurymjxw79nv3jz'

    # Verify that is_active is set
    click_link 'Attributes'
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//a[@data-name="is_active"]') do
      assert_equal "true", text, "Expected user's is_active to be true"
    end

    # go to Admin tab
    click_link 'Admin'
    assert page.has_text? 'As an admin, you can deactivate and reset this user'

    # Click on Deactivate button
    click_button 'Deactivate Active User'

    # Click Ok in the confirm dialog
=begin
#use with selenium
    page.driver.browser.switch_to.alert.accept
    sleep(0.1)
    popup = page.driver.browser.window_handles.last
    #popup = page.driver.browser.window_handle
    page.within_window popup do
      assert has_text? 'Are you sure you want to deactivate'
      click_button "OK"
    end

# use with poltergeist driver
popup = page.driver.window_handles.last
page.within_window popup do
  #fill_in "email", :with => "my_email"
  assert has_text? 'Are you sure you want to deactivate'
  click_button "OK"
end
=end

    # Should now be back in the Attributes tab for the user
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//a[@data-name="is_active"]') do
      assert_equal "false", text, "Expected user's is_active to be false after unsetup"
    end
  end

  test "setup the active user" do
    Capybara.current_driver = :selenium
    visit page_with_token('admin_trustedclient')

    click_link 'Users'

    assert page.has_link? 'zzzzz-tpzed-xurymjxw79nv3jz'

    # click on active user
    click_link 'zzzzz-tpzed-xurymjxw79nv3jz'

    # Setup user
    click_link 'Admin'
    assert page.has_text? 'As an admin, you can setup'

    click_link 'Setup Active User'

    sleep(0.1)
    popup = page.driver.browser.window_handles.last
    page.within_window popup do
      assert has_text? 'Virtual Machine'
      fill_in "repo_name", :with => "test_repo"
      click_button "Submit"
    end

    sleep(0.1)
    assert page.has_text? 'modified_by_client_uuid'

    click_link 'Metadata'
    assert page.has_text? '(Repository: test_repo)'
    assert !(page.has_text? '(VirtualMachine:)')

    # Click on Setup button again and this time also choose a VM
    click_link 'Admin'
    click_link 'Setup Active User'

    sleep(0.1)
    popup = page.driver.browser.window_handles.last
    page.within_window popup do
      fill_in "repo_name", :with => "second_test_repo"
      select("testvm.shell", :from => 'vm_uuid')
      click_button "Submit"
    end

    sleep(0.1)
    assert page.has_text? 'modified_by_client_uuid'

    click_link 'Metadata'
    assert page.has_text? '(Repository: second_test_repo)'
    assert page.has_text? '(VirtualMachine: testvm.shell)'

    # unsetup user and verify all the above links are deleted
    click_link 'Admin'
    click_button 'Deactivate Active User'
    page.driver.browser.switch_to.alert.accept
    sleep(0.1)

#    popup = page.driver.browser.window_handles.last

    click_link 'Metadata'
    assert !(page.has_text? '(Repository: test_repo)')
    assert !(page.has_text? '(Repository: second_test_repo)')
    assert !(page.has_text? '(VirtualMachine: testvm.shell)')

    # setup user again and verify links present
    click_link 'Admin'
    click_link 'Setup Active User'

    sleep(0.1)
    popup = page.driver.browser.window_handles.last
    page.within_window popup do
      fill_in "repo_name", :with => "second_test_repo"
      select("testvm.shell", :from => 'vm_uuid')
      click_button "Submit"
    end

    sleep(0.1)
    assert page.has_text? 'modified_by_client_uuid'

    click_link 'Metadata'
    assert page.has_text? '(Repository: second_test_repo)'
    assert page.has_text? '(VirtualMachine: testvm.shell)'
  end

end
