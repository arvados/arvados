require 'integration_helper'

class LoginsTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  test "login with api_token works after redirect" do
    visit page_with_token('active_trustedclient')
    assert page.has_text?('Active pipelines'), "Missing 'Active pipelines' from page"
    assert_no_match(/\bapi_token=/, current_path)
  end

  test "trying to use expired token redirects to login page" do
    visit page_with_token('expired_trustedclient')
    buttons = all("a.btn", text: /Log in/)
    assert_equal(1, buttons.size, "Failed to find one login button")
    login_link = buttons.first[:href]
    assert_match(%r{//[^/]+/login}, login_link)
    assert_no_match(/\bapi_token=/, login_link)
  end
end
