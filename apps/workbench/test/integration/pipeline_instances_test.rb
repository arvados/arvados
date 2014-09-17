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

    # project chooser
    within('.modal-dialog') do
      find('.selectable', text: 'A Project').click
      find('button', text: 'Choose').click
    end

    # This pipeline needs input. So, Run should be disabled
    page.assert_selector 'a.disabled,button.disabled', text: 'Run'

    instance_page = current_path

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
    within('.modal-dialog') do
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('.btn', text: 'Add').click
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      wait_for_ajax
    end

    click_link 'Jobs and pipelines'
    find('tr[data-kind="arvados#pipelineInstance"]', text: '(none)').
      find('a', text: 'Show').
      click

    assert find('p', text: 'Provide a value')

    find('div.form-group', text: 'Foo/bar pair').
      find('.btn', text: 'Choose').
      click

    within('.modal-dialog') do
      assert(has_text?("Foo/bar pair"),
             "pipeline input picker missing name of input")
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('button', text: 'OK').click
    end
    wait_for_ajax

    # "Run" button is now enabled
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'

    first('a,button', text: 'Run').click

    # Pipeline is running. We have a "Pause" button instead now.
    page.assert_selector 'a,button', text: 'Pause'
    find('a,button', text: 'Pause').click

    # Pipeline is stopped. It should now be in paused state and Runnable again.
    assert page.has_text? 'Paused'
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Resume'
    page.assert_selector 'a,button', text: 'Re-run with latest'

    # Since it is test env, no jobs are created to run. So, graph not visible
    assert_not page.has_text? 'Graph'
  end

  # Create a pipeline instance from within a project and run
  test 'Create pipeline inside a project and run' do
    visit page_with_token('active_trustedclient')

    # Go over to the collections page and select something
    visit '/collections'
    within('tr', text: 'GNU_General_Public_License') do
      find('input[type=checkbox]').click
    end
    find('#persistent-selection-count').click

    # Add this collection to the project using collections menu from top nav
    visit '/projects'
    find('.arv-project-list a,button', text: 'A Project').click
    find('.btn', text: 'Add data').click
    within('.modal-dialog') do
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('.btn', text: 'Add').click
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      wait_for_ajax
    end

    # create a pipeline instance
    find('.btn', text: 'Run a pipeline').click
    within('.modal-dialog') do
      find('.selectable', text: 'Two Part Pipeline Template').click
      find('.btn', text: 'Next: choose inputs').click
    end

    assert find('p', text: 'Provide a value')

    find('div.form-group', text: 'Foo/bar pair').
      find('.btn', text: 'Choose').
      click

    within('.modal-dialog') do
      assert_selector 'button.dropdown-toggle', text: 'A Project'
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('button', text: 'OK').click
    end
    wait_for_ajax

    # "Run" button present and enabled
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'
    first('a,button', text: 'Run').click

    # Pipeline is running. We have a "Pause" button instead now.
    page.assert_no_selector 'a,button', text: 'Run'
    page.assert_selector 'a,button', text: 'Pause'

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

  test "JSON popup available for strange components" do
    uuid = api_fixture("pipeline_instances")["components_is_jobspec"]["uuid"]
    visit page_with_token("active", "/pipeline_instances/#{uuid}")
    click_on "Components"
    assert(page.has_no_text?("script_parameters"),
           "components JSON visible without popup")
    click_on "Show components JSON"
    assert(page.has_text?("script_parameters"),
           "components JSON not found")
  end

  PROJECT_WITH_SEARCH_COLLECTION = "A Subproject"
  def check_parameter_search(proj_name)
    template = api_fixture("pipeline_templates")["parameter_with_search"]
    search_text = template["components"]["with-search"]["script_parameters"]["input"]["search_for"]
    visit page_with_token("active", "/pipeline_templates/#{template['uuid']}")
    click_on "Run this pipeline"
    within(".modal-dialog") do  # Set project for the new pipeline instance
      find(".selectable", text: proj_name).click
      click_on "Choose"
    end
    assert(has_text?("From template"), "did not land on pipeline instance page")
    first("a.btn,button", text: "Choose").click
    within(".modal-body") do
      if (proj_name != PROJECT_WITH_SEARCH_COLLECTION)
        # Switch finder modal to Subproject to find the Collection.
        click_on proj_name
        click_on PROJECT_WITH_SEARCH_COLLECTION
      end
      assert_equal(search_text, first("input").value,
                   "parameter search not preseeded")
      assert(has_text?(api_fixture("collections")["baz_collection_name_in_asubproject"]["name"]),
             "baz Collection not in preseeded search results")
    end
  end

  test "Workbench respects search_for parameter in templates" do
    check_parameter_search(PROJECT_WITH_SEARCH_COLLECTION)
  end

  test "Workbench preserves search_for parameter after project switch" do
    check_parameter_search("A Project")
  end
end
