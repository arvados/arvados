require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class CollectionsTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = Capybara.javascript_driver
  end

  test "Collection page renders name" do
    uuid = api_fixture('collections')['foo_file']['uuid']
    coll_name = api_fixture('collections')['foo_file']['name']
    visit page_with_token('active', "/collections/#{uuid}")
    assert(page.has_text?(coll_name), "Collection page did not include name")
    # Now check that the page is otherwise normal, and the collection name
    # isn't only showing up in an error message.
    assert(page.has_link?('foo'), "Collection page did not include file link")
  end

  test "can download an entire collection with a reader token" do
    Capybara.current_driver = :rack_test

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

  test "combine selected collections into new collection" do
    foo_collection = api_fixture('collections')['foo_file']
    bar_collection = api_fixture('collections')['bar_file']

    visit page_with_token('active', "/collections")

    assert(page.has_text?(foo_collection['uuid']), "Collection page did not include foo file")
    assert(page.has_text?(bar_collection['uuid']), "Collection page did not include bar file")

    within('tr', text: foo_collection['uuid']) do
      find('input[type=checkbox]').click
    end

    within('tr', text: bar_collection['uuid']) do
      find('input[type=checkbox]').click
    end

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Combine selections into a new collection'
    end

    # now in the newly created collection page
    assert(page.has_text?('Copy to project'), "Copy to project text not found in new collection page")
    assert(page.has_no_text?(foo_collection['name']), "Collection page did not include foo file")
    assert(page.has_text?('foo'), "Collection page did not include foo file")
    assert(page.has_no_text?(bar_collection['name']), "Collection page did not include foo file")
    assert(page.has_text?('bar'), "Collection page did not include bar file")
  end

  test "combine selected collection contents into new collection" do
    foo_collection = api_fixture('collections')['foo_file']

    visit page_with_token('active', "/collections")

    # choose file from foo collection
    within('tr', text: foo_collection['uuid']) do
      click_link 'Show'
    end

    # now in collection page
    find('input[type=checkbox]').click

    click_button 'Selection...'
    within('.selection-action-container') do
      click_link 'Combine selections into a new collection'
    end

    # now in the newly created collection page
    assert(page.has_text?('Copy to project'), "Copy to project text not found in new collection page")
    assert(page.has_no_text?(foo_collection['name']), "Collection page did not include foo file")
    assert(page.has_text?('foo'), "Collection page did not include foo file")
  end
end
