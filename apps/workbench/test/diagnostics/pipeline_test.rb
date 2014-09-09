require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class PipelineTest < ActionDispatch::IntegrationTest
  setup do
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  test 'Run tutorial pipeline' do
    visit page_with_token(Rails.configuration.diagnostics_testing_active_user_token)

    # Search for tutorial template
    within('.navbar-fixed-top') do
      page.find_field('search').set 'Diagnostic testing - Tutorial pipeline template'
      page.find('.glyphicon-search').click
    end

    within '.modal-content' do
      find('.selectable', text: 'Diagnostic testing - Tutorial pipeline template').click
      click_button 'Show'
    end

    # Tun the pipeline
    find('a,button', text: 'Run').click

    # Choose project
    within('.modal-dialog') do
      find('.selectable', text: 'Home').click
      find('button', text: 'Choose').click
    end

    # This pipeline needs input. So, Run should be disabled
    page.assert_selector 'a.disabled,button.disabled', text: 'Run'

    instance_page = current_path

    # Choose input for the pipeline
    find('.btn', text: 'Choose').click
    within('.modal-dialog') do
      find('.selectable', text: 'Diagnostic testing - Tutorial pipeline input').click
      find('button', text: 'OK').click
    end
    wait_for_ajax

    # Run this pipeline instance
    find('a,button', text: 'Run').click

    # Pipeline is running. We have a "Stop" button instead now.
    page.assert_selector 'a,button', text: 'Stop'
  end

end
