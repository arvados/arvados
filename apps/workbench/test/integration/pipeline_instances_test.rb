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

    visit '/pipeline_templates'
    within('tr', text: 'Two Part Pipeline Template') do
      find('a,button', text: 'Run').click
    end

    # This pipeline needs input. So, Run should be disabled
    page.assert_selector 'a.disabled,button.disabled', text: 'Run'

    instance_page = current_path

    find('button', text: 'Choose a folder...').click
    within('.modal-dialog') do
      find('.selectable', text: 'A Folder').click
      find('button', text: 'Move').click
    end

    # Go over to the collections page and select something
    visit '/collections'
    within('tr', text: 'GNU_General_Public_License') do
      find('input[type=checkbox]').click
    end
    find('#persistent-selection-count').click

    # Add this collection to the folder
    visit '/folders'
    find('.arv-folder-list a,button', text: 'A Folder').click
    find('.btn', text: 'Add data').click
    find('span', text: 'foo_tag').click
    within('.modal-dialog') do
      find('.btn', text: 'Add').click
    end
   
    find('tr[data-kind="arvados#pipelineInstance"]', text: 'New pipeline instance').
      find('a', text: 'Show').
      click

    assert find('p', text: 'Provide a value')

    find('div.form-group', text: 'Foo/bar pair').
      find('.btn', text: 'Choose').
      click

    within('.modal-dialog') do
      find('span', text: 'foo_tag').click
      find('button', text: 'OK').click
    end

    wait_for_ajax

    # "Run" button is now enabled
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'

    first('a,button', text: 'Run').click

    # Pipeline is running. We have a "Stop" button instead now.
    page.assert_selector 'a,button', text: 'Stop'
    find('a,button', text: 'Stop').click

    # Pipeline is stopped. It should now be in paused state and Runnable again.
    assert page.has_text? 'Paused'
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Resume'
    page.assert_selector 'a,button', text: 'Clone and edit'

    # Since it is test env, no jobs are created to run. So, graph not visible
    assert_not page.has_text? 'Graph'
  end

  test 'view pipeline with job and see graph' do
    visit page_with_token('active_trustedclient')

    visit '/pipeline_instances'
    assert page.has_text? 'pipeline_with_job'

    find('a', text: 'pipeline_with_job').click

    # since the pipeline component has a job, expect to see the graph
    assert page.has_text? 'Graph'
    click_link 'Graph'
    assert page.has_text? 'script_version'
  end

end
