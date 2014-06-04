require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class PipelineInstancesTest < ActionDispatch::IntegrationTest
  setup do
    # Selecting collections requiresLocalStorage
    headless = Headless.new
    headless.start
    Capybara.current_driver = :selenium
  end

  test 'Create and run a pipeline' do
    visit page_with_token('active_trustedclient')

    click_link 'Pipeline templates'
    within('tr', text: 'Two Part Pipeline Template') do
      find('a,button', text: 'Run').click
    end

    instance_page = current_path

    # Go over to the collections page and select something
    click_link 'Collections (data files)'
    within('tr', text: 'GNU_General_Public_License') do
      find('input[type=checkbox]').click
    end
    find('#persistent-selection-count').click

    # Go back to the pipeline instance page to use the new selection
    visit instance_page

    page.assert_selector 'a.disabled,button.disabled', text: 'Run'
    assert find('p', text: 'Provide a value')

    find('div.form-group', text: 'Foo/bar pair').
      find('a,input').
      click
    find('.editable-input select').click
    find('.editable-input').
      first(:option, 'b519d9cb706a29fc7ea24dbea2f05851+249025').click
    wait_for_ajax

    # "Run" button is now enabled
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'

    first('a,button', text: 'Run').click

    # Pipeline is running. We have a "Stop" button instead now.
    page.assert_selector 'a,button', text: 'Stop'
    find('a,button', text: 'Stop').click

    # Pipeline is stopped. We have the option to resume it.
    page.assert_selector 'a,button', text: 'Run'

    # Go over to the graph tab
    click_link 'Graph'
    assert page.has_css? 'div#provenance_graph'
  end
end
