require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class ApplicationLayoutTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  # test the search box
  def verify_search_box user
    if user && user['is_active']
      # let's search for a valid uuid
      within('.navbar-fixed-top') do
        page.has_field?('search')
        page.find_field('search').set user['uuid']
        page.find('.glyphicon-search').click
      end

      # we should now be in the user's home project as a result of search
      assert_selector "#Data_collections[data-object-uuid='#{user['uuid']}']", "Expected to be in user page after search click"

      # let's search again for an invalid valid uuid
      within('.navbar-fixed-top') do
        search_for = String.new user['uuid']
        search_for[0]='1'
        page.find_field('search').set search_for
        page.find('.glyphicon-search').click
      end

      # we should see 'not found' error page
      assert page.has_text?('Not Found'), 'No text - Not Found'
      assert page.has_link?('Report problem'), 'No text - Report problem'
      click_link 'Report problem'
      within '.modal-content' do
        assert page.has_text?('Report a problem'), 'No text - Report a problem'
        assert page.has_no_text?('Version / debugging info'), 'No text - Version / debugging info'
        assert page.has_text?('API version'), 'No text - API version'
        assert page.has_text?('API startup time'), 'No text - API startup time'
        assert page.has_text?('Found a problem?'), 'No text - Found a problem'
        assert page.has_button?('Send problem report'), 'No button - Send problem report'
        assert page.has_button?('Cancel'), 'No button - Cancel'

        # enter a report text and click on report
        page.find_field('report_issue_text').set 'my test report text'
        click_button 'Send problem report'

        # ajax success updated button texts and added footer message
        assert page.has_no_button?('Send problem report'), 'Found button - Send problem report'
        assert page.has_no_button?('Cancel'), 'Found button - Cancel'
        assert page.has_text?('Report sent'), 'No text - Report sent'
        assert page.has_button?('Close'), 'No text - Close'
        assert page.has_text?('Thanks for reporting this issue'), 'No text - Thanks for reporting this issue'

        click_button 'Close'
      end

      # let's search for the anonymously accessible project
      publicly_accessible_project = api_fixture('groups')['anonymously_accessible_project']

      within('.navbar-fixed-top') do
        # search again for the anonymously accessible project
        page.find_field('search').set publicly_accessible_project['name'][0,10]
        page.find('.glyphicon-search').click
      end

      within '.modal-content' do
        assert page.has_text?('All projects'), 'No text - All projects'
        assert page.has_text?('Search'), 'No text - Search'
        assert page.has_text?('Cancel'), 'No text - Cancel'
        assert_selector('div', text: publicly_accessible_project['name'])
        find(:xpath, '//div[./span[contains(.,publicly_accessible_project["uuid"])]]').click

        click_button 'Show'
      end

      # seeing "Unrestricted public data" now
      assert page.has_text?(publicly_accessible_project['name']), 'No text - publicly accessible project name'
      assert page.has_text?(publicly_accessible_project['description']), 'No text - publicly accessible project description'
    else
      within('.navbar-fixed-top') do
        page.has_no_field?('search')
      end
    end
  end

  [
    [nil, nil],
    ['inactive', api_fixture('users')['inactive']],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited']],
    ['active', api_fixture('users')['active']],
    ['admin', api_fixture('users')['admin']],
  ].each do |token, user|

    test "test search box for user #{token}" do
      visit page_with_token(token)

      verify_search_box user
    end

  end

end
