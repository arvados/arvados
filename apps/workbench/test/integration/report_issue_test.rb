require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class ApplicationLayoutTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium

    @user_profile_form_fields = Rails.configuration.user_profile_form_fields
  end

  teardown do
    Rails.configuration.user_profile_form_fields = @user_profile_form_fields
  end

  # test version info and report issue from help menu
  def check_version_info_and_report_issue_from_help_menu
    within('.navbar-fixed-top') do
      page.find("#arv-help").click
      within('.dropdown-menu') do
        assert page.has_link?('Tutorials and User guide'), 'No link - Tutorials and User guide'
        assert page.has_link?('API Reference'), 'No link - API Reference'
        assert page.has_link?('SDK Reference'), 'No link - SDK Reference'
        assert page.has_link?('Show version / debugging info'), 'No link - Show version / debugging info'
        assert page.has_link?('Report a problem'), 'No link - Report a problem'

        # check show version info link
        click_link 'Show version / debugging info'
      end
    end

    within '.modal-content' do
      assert page.has_text?('Version / debugging info'), 'No text - Version / debugging info'
      assert page.has_no_text?('Report a problem'), 'Found text - Report a problem'
      assert page.has_text?('Server version'), 'No text - Server version'
      assert page.has_text?('Server restarted at'), 'No text - Server restarted at'
      assert page.has_text?('Workbench version'), 'No text - Workbench version'
      assert page.has_text?('Arvados base'), 'No text - Arvados base'
      assert page.has_text?('Additional info'), 'No text - Additional info'
      assert page.has_no_text?('Found a problem?'), 'Found text - Found a problem'
      assert page.has_button?('Close'), 'No button - Close'
      assert page.has_no_button?('Report issue'), 'Found button - Report issue'
      click_button 'Close'
    end

    # check report issue link
    within('.navbar-fixed-top') do
      page.find("#arv-help").click
      within('.dropdown-menu') do
        click_link 'Report a problem'
      end
    end

    within '.modal-content' do
      assert page.has_text?('Report a problem'), 'No text - Report a problem'
      assert page.has_no_text?('Version / debugging info'), 'Found text - Version / debugging info'
      assert page.has_text?('Server version'), 'No text - Server version'
      assert page.has_text?('Server restarted at'), 'No text - Server restarted at'
      assert page.has_text?('Workbench version'), 'No text - Workbench version'
      assert page.has_text?('Arvados base'), 'No text - Arvados base'
      assert page.has_text?('Additional info'), 'No text - Additional info'
      assert page.has_text?('Found a problem?'), 'No text - Found a problem'
      assert page.has_no_button?('Close'), 'Found button - Close'
      assert page.has_button?('Report issue'), 'No button - Report issue'
      assert page.has_button?('Cancel'), 'No button - Cancel'

      # enter a report text and click on report
      page.find_field('report_issue_text').set 'my test report text'
      click_button 'Report issue'

      # ajax success updated button texts and added footer message
      assert page.has_no_button?('Report issue'), 'Found button - Report issue'
      assert page.has_no_button?('Cancel'), 'Found button - Cancel'
      assert page.has_text?('Report sent'), 'No text - Report sent'
      assert page.has_button?('Close'), 'No text - Close'
      assert page.has_text?('Thanks for reporting this issue'), 'No text - Thanks for reporting this issue'

      click_button 'Close'
    end
  end

  [
    [nil, nil],
    ['inactive', api_fixture('users')['inactive']],
    ['inactive_uninvited', api_fixture('users')['inactive_uninvited']],
    ['active', api_fixture('users')['active']],
    ['admin', api_fixture('users')['admin']],
    ['active_no_prefs', api_fixture('users')['active_no_prefs']],
    ['active_no_prefs_profile', api_fixture('users')['active_no_prefs_profile']],
  ].each do |token, user|

    test "check help for user #{token}" do
      if !token
        visit ('/')
      else
        visit page_with_token(token)
      end

      check_version_info_and_report_issue_from_help_menu
    end

  end

end
