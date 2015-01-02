require 'integration_helper'

class ErrorsTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  BAD_UUID = "ffffffffffffffffffffffffffffffff+0"

  test "error page renders user navigation" do
    visit(page_with_token("active", "/collections/#{BAD_UUID}"))
    assert(page.has_text?(api_fixture("users")["active"]["email"]),
           "User information missing from error page")
    assert(page.has_no_text?(/log ?in/i),
           "Logged in user prompted to log in on error page")
  end

  test "no user navigation with expired token" do
    visit(page_with_token("expired", "/collections/#{BAD_UUID}"))
    assert(page.has_no_text?(api_fixture("users")["active"]["email"]),
           "Page visited with expired token included user information")
    assert(page.has_selector?("a", text: /log ?in/i),
           "Login prompt missing on expired token error page")
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

  def now_timestamp
    Time.now.utc.to_i
  end

  def page_has_error_token?(start_stamp)
    matching_stamps = (start_stamp .. now_timestamp).to_a.join("|")
    # Check the page HTML because we really don't care how it's presented.
    # I think it would even be reasonable to put it in a comment.
    page.html =~ /\b(#{matching_stamps})\+[0-9A-Fa-f]{8}\b/
  end

  # We use API tokens with limited scopes as the quickest way to get the API
  # server to return an error.  If Workbench gets smarter about coping when
  # it has a too-limited token, these tests will need to be adjusted.
  test "API error page includes error token" do
    start_stamp = now_timestamp
    visit(page_with_token("active_readonly", "/authorized_keys"))
    click_on "Add a new authorized key"
    assert(page.has_text?(/fiddlesticks/i),
           "Not on an error page after making an SSH key out of scope")
    assert(page_has_error_token?(start_stamp), "no error token on 404 page")
  end

  test "showing a bad UUID returns 404" do
    visit(page_with_token("active", "/pipeline_templates/zzz"))
    assert(page.has_no_text?(/fiddlesticks/i),
           "trying to show a bad UUID rendered a fiddlesticks page, not 404")
  end

  test "404 page includes information about missing object" do
    visit(page_with_token("active", "/groups/zazazaz"))
    assert(page.has_text?(/group with UUID zazazaz/i),
           "name of searched group missing from 404 page")
  end

  test "unrouted 404 page works" do
    visit(page_with_token("active", "/__asdf/ghjk/zxcv"))
    assert(page.has_text?(/not found/i),
           "unrouted page missing 404 text")
    assert(page.has_no_text?(/fiddlesticks/i),
           "unrouted request returned a generic error page, not 404")
  end

  test "API error page has Report problem button" do
    original_arvados_v1_base = Rails.configuration.arvados_v1_base

    begin
      # point to a bad api server url to generate fiddlesticks error
      Rails.configuration.arvados_v1_base = "https://[100::f]:1/"

      visit page_with_token("active")

      assert(page.has_text?(/fiddlesticks/i), 'Expected to be in error page')

      # reset api server base config to let the popup rendering to work
      Rails.configuration.arvados_v1_base = original_arvados_v1_base

      # check the "Report problem" button
      assert page.has_link? 'Report problem', 'Report problem link not found'

      click_link 'Report problem'
      within '.modal-content' do
        assert page.has_text?('Report a problem'), 'Report a problem text not found'
        assert page.has_no_text?('Version / debugging info'), 'Version / debugging info is not expected'
        assert page.has_text?('Describe the problem'), 'Describe the problem text not found'
        assert page.has_text?('Send problem report'), 'Send problem report button text is not found'
        assert page.has_no_button?('Send problem report'), 'Send problem report button is not disabled before entering problem description'
        assert page.has_button?('Cancel'), 'Cancel button not found'

        # enter a report text and click on report
        page.find_field('report_issue_text').set 'my test report text'
        assert page.has_button?('Send problem report'), 'Send problem report button not enabled after entering text'
        click_button 'Send problem report'

        # ajax success updated button texts and added footer message
        assert page.has_no_text?('Send problem report'), 'Found button - Send problem report'
        assert page.has_no_button?('Cancel'), 'Found button - Cancel'
        assert page.has_text?('Report sent'), 'No text - Report sent'
        assert page.has_button?('Close'), 'No button - Close'
        assert page.has_text?('Thanks for reporting this issue'), 'No text - Thanks for reporting this issue'

        click_button 'Close'
      end

      # out of the popup now and should be back in the error page
      assert(page.has_text?(/fiddlesticks/i), 'Expected to be in error page after closing report issue popup')
    ensure
      Rails.configuration.arvados_v1_base = original_arvados_v1_base
    end
  end

end
