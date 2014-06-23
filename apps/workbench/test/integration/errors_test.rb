require 'integration_helper'

class ErrorsTest < ActionDispatch::IntegrationTest
  BAD_UUID = "ffffffffffffffffffffffffffffffff+0"

  test "error page renders user navigation" do
    visit(page_with_token("active", "/collections/#{BAD_UUID}"))
    assert(page.has_text?(api_fixture("users")["active"]["email"]),
           "User information missing from error page")
    assert(page.has_no_text?(/log ?in/i),
           "Logged in user prompted to log in on error page")
  end

  test "error page renders without login" do
    visit "/collections/download/#{BAD_UUID}/#{@@API_AUTHS['active']['api_token']}"
    assert(page.has_no_text?(/\b500\b/),
           "Error page without login returned 500")
  end

  test "'object not found' page includes search link" do
    visit(page_with_token("active", "/collections/#{BAD_UUID}"))
    assert(all("a").any? { |a| a[:href] =~ %r{/collections/?(\?|$)} },
           "no search link found on 404 page")
  end
end
