require 'integration_helper'

class SystemMenuTest < ActionDispatch::IntegrationTest
  # These tests don't do state-changing API calls. Save some time by
  # skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  setup do
    need_javascript
  end

  [
    #['Repositories','repository','Attributes'], Fails due to #6652
    ['Virtual machines','virtual machine','current_user_logins'],
    ['SSH keys','authorized key','public_key'],
    ['Links','link','link_class'],
    ['Groups','group','group_class'],
    ['Compute nodes','node','info[ping_secret'],
    ['Keep services','keep service','service_ssl_flag'],
    ['Keep disks', 'keep disk','bytes_free'],
  ].each do |page_name, add_button_text, look_for|

    test "test system menu #{page_name} link" do
      visit page_with_token('admin')
      within('.navbar-fixed-top') do
        page.find("#system-menu").click
        within('.dropdown-menu') do
          assert_selector 'a', text: page_name
          find('a', text: page_name).click
        end
      end

      # click the add button
      assert_selector 'button', text: "Add a new #{add_button_text}"
      find('button', text: "Add a new #{add_button_text}").click

      # look for unique property in the created object page
      assert page.has_text? look_for
    end
  end
end
