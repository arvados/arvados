require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class CollectionsTest < ActionDispatch::IntegrationTest
  def change_persist oldstate, newstate
    find "div[data-persistent-state='#{oldstate}']"
    page.assert_no_selector "div[data-persistent-state='#{newstate}']"
    find('.btn', text: oldstate.capitalize).click
    find '.btn', text: newstate.capitalize
    page.assert_no_selector '.btn', text: oldstate.capitalize
    find "div[data-persistent-state='#{newstate}']"
    page.assert_no_selector "div[data-persistent-state='#{oldstate}']"
  end

  ['/collections', '/'].each do |path|
    test "Flip persistent switch at #{path}" do
      Capybara.current_driver = Capybara.javascript_driver
      uuid = api_fixture('collections')['foo_file']['uuid']
      visit page_with_token('active', path)
      within "tr[data-object-uuid='#{uuid}']" do
        change_persist 'cache', 'persistent'
      end
      # Refresh page and make sure the change was committed.
      visit current_path
      within "tr[data-object-uuid='#{uuid}']" do
        change_persist 'persistent', 'cache'
      end
    end
  end

  test 'Flip persistent switch on collection#show' do
    Capybara.current_driver = Capybara.javascript_driver
    uuid = api_fixture('collections')['foo_file']['uuid']
    visit page_with_token('active', "/collections/#{uuid}")
    change_persist 'cache', 'persistent'
    visit current_path
    change_persist 'persistent', 'cache'
  end

  test "can download an entire collection with a reader token" do
    uuid = api_fixture('collections')['foo_file']['uuid']
    token = api_fixture('api_client_authorizations')['active']['api_token']
    q_string = URI.encode_www_form('reader_tokens[]' => token)
    visit "/collections/#{uuid}?#{q_string}"
    # It seems that Capybara can't inspect tags outside the body, so this is
    # a very blunt approach.
    assert_no_match(/\bnofollow\b/i, page.html,
                    "wget prohibited from recursing the collection page")
    # TODO: When we can test against a Keep server, actually click the link
    # and check the contents, rather than testing the href directly
    # (this is too closely tied to implementation details).
    link = nil
    assert_nothing_raised("failed to list foo files with reader token") do
      link = find_link('Download')
    end
    assert_match(%r{^/collections/#{Regexp.escape uuid}/foo}, link[:href],
                 "download link doesn't point to foo file")
    assert_match(/\b#{Regexp.escape q_string}\b/, link[:href],
                 "collection file download link did not inherit reader tokens")
  end
end
