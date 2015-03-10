require 'integration_helper'

class AjaxErrorsTest < ActionDispatch::IntegrationTest
  setup do
    # Regrettably...
    need_selenium 'to assert_text in iframe'
  end

  test 'load pane with deleted session' do
    # Simulate loading a page in browser-tab A, hitting "Log out" in
    # browser-tab B, then returning to browser-tab A and choosing a
    # different tab. (Automatic tab refreshes will behave similarly.)
    visit page_with_token('active', '/projects/' + api_fixture('groups')['aproject']['uuid'])
    ActionDispatch::Request::Session.any_instance.stubs(:[]).returns(nil)
    click_link "Subprojects"
    wait_for_ajax
    assert_no_double_layout
    assert_selector 'a,button', text: 'Reload tab'
    assert_selector '.pane-error-display'
    page.driver.browser.switch_to.frame 0
    assert_text 'You are not logged in.'
  end

  test 'load pane with expired token' do
    # Similar to 'deleted session'. Here, the session cookie is still
    # alive, but it contains a token which has expired. This uses a
    # different code path because Workbench cannot detect that
    # anything is amiss until it actually uses the token in an API
    # request.
    visit page_with_token('active', '/projects/' + api_fixture('groups')['aproject']['uuid'])
    use_token :active_trustedclient do
      # Go behind Workbench's back to expire the "active" token.
      token = api_fixture('api_client_authorizations')['active']['api_token']
      auth = ApiClientAuthorization.find(token)
      auth.update_attributes(expires_at: '1999-12-31T23:59:59Z')
    end
    click_link "Subprojects"
    wait_for_ajax
    assert_no_double_layout
    assert_selector 'a,button', text: 'Reload tab'
    assert_selector '.pane-error-display'
    page.driver.browser.switch_to.frame 0
    assert_text 'You are not logged in.'
  end

  protected

  def assert_no_double_layout
    # Check we're not rendering a full page layout within a tab
    # pane. Bootstrap responsive layouts require exactly one
    # div.container-fluid. Checking "body body" would be more generic,
    # but doesn't work when the browser/driver automatically collapses
    # syntatically invalid tags.
    assert_no_selector '.container-fluid .container-fluid'
  end
end
