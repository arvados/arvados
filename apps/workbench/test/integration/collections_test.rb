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

end
