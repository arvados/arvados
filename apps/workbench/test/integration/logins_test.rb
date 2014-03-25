require 'test_helper'

class LoginsTest < ActionDispatch::IntegrationTest
  test "login with api_token works after redirect" do
    visit page_with_token('active_trustedclient')
    assert page.has_text? 'Recent jobs'
    assert_no_match(/\bapi_token=/, current_path)
  end

  test "can't use expired token" do
    visit page_with_token('expired_trustedclient')
    assert page.has_text? 'Log in'
  end
end
