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

    # put this pipeline instance in "A Project"
    find('button', text: 'Choose a project...').click
    within('.modal-dialog') do
      find('.selectable', text: 'A Project').click
      find('button', text: 'Move').click
    end

    # Go over to the collections page and select something
    visit '/collections'
    within('tr', text: 'GNU_General_Public_License') do
      find('input[type=checkbox]').click
    end
    find('#persistent-selection-count').click

    # Add this collection to the project
    visit '/projects'
    find('.arv-project-list a,button', text: 'A Project').click
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
      first('span', text: 'foo_tag').click
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

  # Create a pipeline instance from within a project and run
  test 'Create pipeline inside a project and run' do
    add_a_collection_and_pipeline_to_project
  end

  def add_a_collection_and_pipeline_to_project
    visit page_with_token('active_trustedclient')

    # Go over to the collections page and select something
    visit '/collections'
    within('tr', text: 'GNU_General_Public_License') do
      find('input[type=checkbox]').click
    end
    find('#persistent-selection-count').click

    # Add this collection to the project using collections menu from top nav
    visit '/'
    find('.arv-project-list a,button', text: 'A Project').click

    find('li.selection-menu > a').click
    click_button 'Copy selections into this project'

    # create a pipeline instance
    find('.btn', text: 'Run a pipeline').click
    within('.modal-dialog') do
      assert page.has_text? 'Two Part Pipeline Template'
      find('.fa-gear').click
      find('.btn', text: 'Next: choose inputs').click
    end

    assert find('p', text: 'Provide a value')

    find('div.form-group', text: 'Foo/bar pair').
      find('.btn', text: 'Choose').
      click

    within('.modal-dialog') do
      first('span', text: 'foo_tag').click
      find('button', text: 'OK').click
    end

    wait_for_ajax

    # "Run" button present and enabled
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'
    first('a,button', text: 'Run').click

    # Pipeline is running. We have a "Stop" button instead now.
    page.assert_no_selector 'a,button', text: 'Run'
    page.assert_selector 'a,button', text: 'Stop'

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

  # Visit project as anonymous user and verify that pipeline cannot be modified
  test 'visit shared project as anonymous user' do
    add_a_collection_and_pipeline_to_project

    # login as anonymous user and verify that top nav
    visit page_with_token('anonymous')
    
    within('.navbar-fixed-top') do
      assert page.has_text? 'You are viewing public data'
      assert page.has_link? 'Log in'
    end

    assert page.has_text? 'Welcome'
    assert page.has_no_text? 'My projects'
    assert page.has_no_button? 'Add new project'
    assert page.has_text? 'Projects shared with me'
    assert page.has_text? 'A Project'
    assert page.has_text? 'Unrestricted public data'

    find('.arv-project-list a,button', text: 'Unrestricted public data').click
    page.has_text? ('An anonymously accessible project')

    find('a', text: 'Projects').click
    within('.dropdown-menu') do
      page.has_no_text? ('New project')
      page.has_text? ('Projects shared with me')
    end

    # as anonymous user verify the shared project is accessible
    visit page_with_token('anonymous')
    assert page.has_text? 'A Project'
    find('a', text: 'A Project').click
    page.has_text? ('Test project belonging to active user')

    #find('tr[data-kind="arvados#pipelineInstance"]', text: 'New pipeline instance').
    #  find('a', text: 'Show').click

    # as inactive user "A Project" is accessible
    visit page_with_token('inactive')
    assert page.has_text? 'A Project'
    find('.arv-project-list a,button', text: 'Unrestricted public data').click
    page.has_text? ('An anonymously accessible project')
    find('a', text: 'Projects').click
    find('a', text: 'A Project').click
    page.has_text? ('Test project belonging to active user')
    find('a', text: 'Projects').click
    within('.dropdown-menu') do
      page.has_text? ('New project')
      page.has_text? ('Projects shared with me')
    end
  end

end
