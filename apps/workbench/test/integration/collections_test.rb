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

  ['/collections'].each do |path|
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

  test "Collection page renders default name links" do
    uuid = api_fixture('collections')['foo_file']['uuid']
    coll_name = api_fixture('links')['foo_collection_name_in_aproject']['name']
    name_uuid = api_fixture('links')['foo_collection_name_in_aproject']['uuid']
    visit page_with_token('active', "/collections/#{name_uuid}")
    assert(page.has_text?(coll_name), "Collection page did not include name")
    # Now check that the page is otherwise normal, and the collection name
    # isn't only showing up in an error message.
    assert(page.has_link?('foo'), "Collection page did not include file link")
  end

  test "can download an entire collection with a reader token" do
    uuid = api_fixture('collections')['foo_file']['uuid']
    token = api_fixture('api_client_authorizations')['active_all_collections']['api_token']
    url_head = "/collections/download/#{uuid}/#{token}/"
    visit url_head
    # It seems that Capybara can't inspect tags outside the body, so this is
    # a very blunt approach.
    assert_no_match(/<\s*meta[^>]+\bnofollow\b/i, page.html,
                    "wget prohibited from recursing the collection page")
    # TODO: When we can test against a Keep server, actually follow links
    # and check their contents, rather than testing the href directly
    # (this is too closely tied to implementation details).
    hrefs = page.all('a').map do |anchor|
      link = anchor[:href] || ''
      if link.start_with? url_head
        link[url_head.size .. -1]
      elsif link.start_with? '/'
        nil
      else
        link
      end
    end
    assert_equal(['foo'], hrefs.compact.sort,
                 "download page did provide strictly file links")
  end

  test "can view empty collection" do
    uuid = 'd41d8cd98f00b204e9800998ecf8427e+0'
    visit page_with_token('active', "/collections/#{uuid}")
    assert page.has_text?('This collection is empty')
  end
end
