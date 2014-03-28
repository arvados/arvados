require 'integration_helper'

class UsersTest < ActionDispatch::IntegrationTest

  test "create a new user" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('admin_trustedclient')

    click_link 'Users'

    assert page.has_text? 'zzzzz-tpzed-d9tiejq69daie8f'

    click_on 'Add a new user'
    
    # for now just check that we are back in Users -> List page
    assert page.has_text? 'zzzzz-tpzed-d9tiejq69daie8f'
  end

  test "unsetup active user" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('admin_trustedclient')

    click_link 'Users'

    assert page.has_link? 'zzzzz-tpzed-xurymjxw79nv3jz'

    # click on active user
    click_link 'zzzzz-tpzed-xurymjxw79nv3jz'
    assert page.has_text? 'Attributes'
    assert page.has_text? 'Metadata'
    assert page.has_text? 'Admin'

    # go to the Attributes tab
    click_link 'Attributes'
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//a[@data-name="is_active"]') do
      assert_equal text, "true", "Expected user's is_active to be true"
    end

    # go to Admin tab
    click_link 'Admin'
    assert page.has_text? 'As an admin, you can deactivate and reset this user'

    # Click on Deactivate button
    click_button 'Deactivate Active User'

    # Click Ok in the confirm dialog
    sleep(0.1)

    popup = page.driver.window_handles.last
    page.within_window popup do
      assert has_text? 'Are you sure you want to deactivate Active User'
      click_button "Ok"
    end

    # Should now be back in the Attributes tab for the user
    assert page.has_text? 'modified_by_user_uuid'
    page.within(:xpath, '//a[@data-name="is_active"]') do
      assert_equal text, "false", "Expected user's is_active to be false after unsetup"
    end

  end

  test "setup the active user" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('admin_trustedclient')

    click_link 'Users'

    assert page.has_link? 'zzzzz-tpzed-xurymjxw79nv3jz'

    # click on active user
    click_link 'zzzzz-tpzed-xurymjxw79nv3jz'
    assert page.has_text? 'Attributes'
    assert page.has_text? 'Metadata'
    assert page.has_text? 'Admin'

    # go to Admin tab
    click_link 'Admin'
    assert page.has_text? 'As an admin, you can deactivate and reset this user'

=begin
    # Click on Setup button
    click_button 'Setup Active User'

    # Click Ok in the confirm dialog
    sleep(0.1)

    popup = page.driver.window_handles.last
    page.within_window popup do
      assert has_text? 'Are you sure you want to deactivate Active User'
      fill_in "email", :with => "test@example.com"
      click_button "Ok"
    end

    # Should now be back in the Attributes tab for the user
    assert page.has_text? 'modified_by_client_uuid'

    puts "\n\n************* page now = \n#{page.body}"
=end
  end

end
