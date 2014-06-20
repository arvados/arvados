require 'integration_helper'

class ErrorsTest < ActionDispatch::IntegrationTest
  BAD_UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

  test "error page renders user navigation" do
    visit(page_with_token("active", "/collections/#{BAD_UUID}"))
    assert(page.has_text?(@@API_AUTHS["active"]["email"]),
           "User information missing from error page")
    assert(page.has_no_text?(/log ?in/i),
           "Logged in user prompted to log in on error page")
  end

  test "error page renders without login" do
    visit "/collections/download/#{BAD_UUID}/#{@@API_AUTHS['active']['api_token']}"
    assert(page.has_no_text?(/\b500\b/),
           "Error page without login returned 500")
  end
end
