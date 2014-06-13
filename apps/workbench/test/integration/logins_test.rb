require 'integration_helper'

class LoginsTest < ActionDispatch::IntegrationTest
  test "login with api_token works after redirect" do
    visit page_with_token('active_trustedclient')
    assert page.has_text?('Recent jobs'), "Missing 'Recent jobs' from page"
    assert_no_match(/\bapi_token=/, current_path)
  end

  test "can't use expired token" do
    visit page_with_token('expired_trustedclient')
    assert page.has_text? 'Log in'
  end

  test "expired token yields login page, not error page" do
    visit page_with_token('expired_trustedclient')
    # Even the error page has a "Log in" link. We should look for
    # something that only appears the real login page.
    assert page.has_text? ' Log in Oh... fiddlesticks. Sorry, I had some trouble handling your request'
  end
end
