require 'integration_helper'

class ApiClientAuthorizationsTest < ActionDispatch::IntegrationTest
  test "try loading Manage API tokens page" do
    Capybara.current_driver = Capybara.javascript_driver
    visit page_with_token('admin_trustedclient')
    click_link 'user-menu'
    click_link 'Manage API tokens'
    assert_equal 200, status_code
  end
end
