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

    # Add this collection to the project
    visit '/projects'
    find("#projects-menu").click
    find('.dropdown-menu a,button', text: 'A Project').click
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

    # The input, after being specified, should still be displayed (#3382)
    assert find('div.form-group', text: 'Foo/bar pair')

    # The input, after being specified, should still be editable (#3382)
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

    # For good measure, check one last time that the input, after being specified twice, is still be displayed (#3382)
    assert find('div.form-group', text: 'Foo/bar pair')

    # Ensure that the collection's portable_data_hash, uuid and name
    # are saved in the desired places. (#4015)

    # foo_collection_in_aproject is the collection tagged with foo_tag.
    col = api_fixture('collections', 'foo_collection_in_aproject')
    click_link 'Advanced'
    click_link 'API response'
    api_response = JSON.parse(find('div#advanced_api_response pre').text)
    input_params = api_response['components']['part-one']['script_parameters']['input']
    assert_equal input_params['value'], col['portable_data_hash']
    assert_equal input_params['selection_name'], col['name']
    assert_equal input_params['selection_uuid'], col['uuid']

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
    page.assert_selector 'a,button', text: 'Re-run options'

    # Since it is test env, no jobs are created to run. So, graph not visible
    assert_not page.has_text? 'Graph'
  end

  # Create a pipeline instance from within a project and run
  test 'Create pipeline inside a project and run' do
    visit page_with_token('active_trustedclient')

    # Add this collection to the project using collections menu from top nav
    visit '/projects'
    find("#projects-menu").click
    find('.dropdown-menu a,button', text: 'A Project').click
    find('.btn', text: 'Add data').click
    within('.modal-dialog') do
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('.btn', text: 'Add').click
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      wait_for_ajax
    end

    create_and_run_pipeline_in_aproject true, 'Two Part Pipeline Template', false
  end

  # Create a pipeline instance from outside of a project
  test 'Run a pipeline from dashboard' do
    visit page_with_token('active_trustedclient')
    create_and_run_pipeline_in_aproject false, 'Two Part Pipeline Template', false
  end

  test 'view pipeline with job and see graph' do
    visit page_with_token('active_trustedclient')

    visit '/pipeline_instances'
    assert page.has_text? 'pipeline_with_job'

    find('a', text: 'pipeline_with_job').click

    # since the pipeline component has a job, expect to see the graph
    assert page.has_text? 'Graph'
    click_link 'Graph'
    page.assert_selector "#provenance_graph"
  end

  test 'pipeline description' do
    visit page_with_token('active_trustedclient')

    visit '/pipeline_instances'
    assert page.has_text? 'pipeline_with_job'

    find('a', text: 'pipeline_with_job').click

    within('.arv-description-as-subtitle') do
      find('.fa-pencil').click
      find('.editable-input textarea').set('*Textile description for pipeline instance*')
      find('.editable-submit').click
    end
    wait_for_ajax

    # verify description
    assert page.has_no_text? '*Textile description for pipeline instance*'
    assert page.has_text? 'Textile description for pipeline instance'
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
    assert(has_text?("This pipeline was created from the template"), "did not land on pipeline instance page")
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

  [
    ['active', false, false, false, 'Two Part Pipeline Template', false],
    ['active', false, false, true, 'Two Part Pipeline Template', false],
    ['active', true, false, false, 'Two Part Pipeline Template', false],
    ['active', true, true, false, 'Two Part Pipeline Template', false],
    ['active', true, false, true, 'Two Part Pipeline Template', false],
    ['active', true, true, true, 'Two Part Pipeline Template', false],
    ['project_viewer', false, false, true, 'Two Part Pipeline Template', false],
    ['project_viewer', true, false, true, 'Two Part Pipeline Template', false],
    ['project_viewer', true, true, true, 'Two Part Pipeline Template', false],
    ['active', false, false, false, 'Two Part Template with dataclass File', true],
    ['active', false, false, true, 'Two Part Template with dataclass File', true],
  ].each do |user, with_options, choose_options, in_aproject, template_name, choose_file|
    test "Rerun pipeline instance as #{user} using options #{with_options} #{choose_options}
          in #{in_aproject} with #{template_name} with file #{choose_file}" do
      visit page_with_token('active')

      # need bigger modal size when choosing a file from collection
      Capybara.current_session.driver.browser.manage.window.resize_to(1024, 768)

      if in_aproject
        find("#projects-menu").click
        find('.dropdown-menu a,button', text: 'A Project').click
      end

      create_and_run_pipeline_in_aproject in_aproject, template_name, choose_file
      instance_path = current_path

      # Pause the pipeline
      find('a,button', text: 'Pause').click
      assert page.has_text? 'Paused'
      page.assert_no_selector 'a.disabled,button.disabled', text: 'Resume'
      page.assert_selector 'a,button', text: 'Re-run with latest'
      page.assert_selector 'a,button', text: 'Re-run options'

      # Pipeline can be re-run now. Access it as the specified user, and re-run
      if user == 'project_viewer'
        visit page_with_token(user, instance_path)
        assert page.has_text? 'A Project'
        page.assert_no_selector 'a.disabled,button.disabled', text: 'Resume'
        page.assert_selector 'a,button', text: 'Re-run with latest'
        page.assert_selector 'a,button', text: 'Re-run options'
      end

      # Now re-run the pipeline
      if with_options
        find('a,button', text: 'Re-run options').click
        within('.modal-dialog') do
          page.assert_selector 'a,button', text: 'Copy and edit inputs'
          page.assert_selector 'a,button', text: 'Run now'
          if choose_options
            find('button', text: 'Copy and edit inputs').click
          else
            find('button', text: 'Run now').click
          end
        end
      else
        find('a,button', text: 'Re-run with latest').click
      end

      # Verify that the newly created instance is created in the right project.
      # In case of project_viewer user, since the use cannot write to the project,
      # the pipeline should have been created in the user's Home project.
      rerun_instance_path = current_path
      assert_not_equal instance_path, rerun_instance_path, 'Rerun instance path expected to be different'
      assert page.has_text? 'Home'
      if in_aproject && (user != 'project_viewer')
        assert page.has_text? 'A Project'
      else
        assert page.has_no_text? 'A Project'
      end
    end
  end

  # Create and run a pipeline for 'Two Part Pipeline Template' in 'A Project'
  def create_and_run_pipeline_in_aproject in_aproject, template_name, choose_file
    # create a pipeline instance
    find('.btn', text: 'Run a pipeline').click
    within('.modal-dialog') do
      find('.selectable', text: template_name).click
      find('.btn', text: 'Next: choose inputs').click
    end

    assert find('p', text: 'Provide a value')

    find('div.form-group', text: 'Foo/bar pair').
      find('.btn', text: 'Choose').
      click

    within('.modal-dialog') do
      if in_aproject
        assert_selector 'button.dropdown-toggle', text: 'A Project'
        wait_for_ajax
      else
        assert_selector 'button.dropdown-toggle', text: 'Home'
        wait_for_ajax
        click_button "Home"
        click_link "A Project"
        wait_for_ajax
      end
      first('span', text: 'foo_tag').click
      if choose_file
        wait_for_ajax
        find('.preview-selectable', text: 'foo').click
      end
      find('button', text: 'OK').click
    end
    wait_for_ajax

    # The input, after being specified, should still be displayed (#3382)
    assert find('div.form-group', text: 'Foo/bar pair')

    # Ensure that the collection's portable_data_hash, uuid and name
    # are saved in the desired places. (#4015)

    # foo_collection_in_aproject is the collection tagged with foo_tag.
    col = api_fixture('collections', 'foo_collection_in_aproject')
    click_link 'Advanced'
    click_link 'API response'
    api_response = JSON.parse(find('div#advanced_api_response pre').text)
    input_params = api_response['components']['part-one']['script_parameters']['input']
    assert_equal(input_params['selection_uuid'], col['uuid'], "Not found expected input param uuid")
    if choose_file
      assert_equal(input_params['value'], col['portable_data_hash']+'/foo', "Not found expected input file param value")
      assert_equal(input_params['selection_name'], col['name']+'/foo', "Not found expected input file param name")
    else
      assert_equal(input_params['value'], col['portable_data_hash'], "Not found expected input param value")
      assert_equal(input_params['selection_name'], col['name'], "Not found expected input param name")
    end

    # "Run" button present and enabled
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Run'
    first('a,button', text: 'Run').click

    # Pipeline is running. We have a "Pause" button instead now.
    page.assert_no_selector 'a,button', text: 'Run'
    page.assert_no_selector 'a.disabled,button.disabled', text: 'Resume'
    page.assert_selector 'a,button', text: 'Pause'

    # Since it is test env, no jobs are created to run. So, graph not visible
    assert_not page.has_text? 'Graph'
  end

  [
    [1, 0], # run time 0 minutes
    [10, 17*60*60 + 51*60], # run time 17 hours and 51 minutes
  ].each do |index, run_time|
    test "pipeline start and finish time display #{index}" do
      visit page_with_token("user1_with_load", "/pipeline_instances/zzzzz-d1hrv-10pipelines0#{index.to_s.rjust(3, '0')}")

      assert page.has_text? 'This pipeline started at'
      page_text = page.text

      match = /This pipeline started at (.*)\. It failed after (.*) seconds at (.*)\. Check the Log/.match page_text
      assert_not_nil(match, 'Did not find text - This pipeline started at . . . ')

      start_at = match[1]
      finished_at = match[3]
      assert_not_nil(start_at, 'Did not find start_at time')
      assert_not_nil(finished_at, 'Did not find finished_at time')

      # start and finished time display is of the format '2:20 PM 10/20/2014'
      start_time = DateTime.strptime(start_at, '%H:%M %p %m/%d/%Y').to_time
      finished_time = DateTime.strptime(finished_at, '%H:%M %p %m/%d/%Y').to_time
      assert_equal(run_time, finished_time-start_time,
        "Time difference did not match for start_at #{start_at}, finished_at #{finished_at}, ran_for #{match[2]}")
    end
  end

  [
    ['fuse', nil, 2, 20],                           # has 2 as of 11-07-2014
    ['fuse', 'FUSE project', 1, 1],                 # 1 with this name
    ['user1_with_load', nil, 30, 100],              # has 37 as of 11-07-2014
    ['user1_with_load', 'pipeline_10', 2, 2],       # 2 with this name
    ['user1_with_load', '000010pipelines', 10, 10], # owned_by the project zzzzz-j7d0g-000010pipelines
    ['user1_with_load', '000025pipelines', 25, 25], # owned_by the project zzzzz-j7d0g-000025pipelines, two pages
    ['admin', nil, 40, 200],
    ['admin', 'FUSE project', 1, 1],
    ['admin', 'pipeline_10', 2, 2],
    ['active', 'containing at least two', 2, 100],  # component description
    ['admin', 'containing at least two', 2, 100],
    ['active', nil, 10, 100],
    ['active', 'no such match', 0, 0],
  ].each do |user, search_filter, expected_min, expected_max|
    test "scroll pipeline instances page for #{user} with search filter #{search_filter}
          and expect #{expected_min} <= found_items <= #{expected_max}" do
      visit page_with_token(user, "/pipeline_instances")

      if search_filter
        find('.recent-pipeline-instances-filterable-control').set(search_filter)
        # Wait for 250ms debounce timer (see filterable.js)
        sleep 0.350
        wait_for_ajax
      end

      page_scrolls = expected_max/20 + 2    # scroll num_pages+2 times to test scrolling is disabled when it should be
      within('.arv-recent-pipeline-instances') do
        (0..page_scrolls).each do |i|
          page.execute_script "window.scrollBy(0,999000)"
          begin
            wait_for_ajax
          rescue
          end
        end
      end

      # Verify that expected number of pipeline instances are found
      found_items = page.all('tr[data-kind="arvados#pipelineInstance"]')
      found_count = found_items.count
      if expected_min == expected_max
        assert_equal(true, found_count == expected_min,
          "Not found expected number of items. Expected #{expected_min} and found #{found_count}")
        assert page.has_no_text? 'request failed'
      else
        assert_equal(true, found_count>=expected_min,
          "Found too few items. Expected at least #{expected_min} and found #{found_count}")
        assert_equal(true, found_count<=expected_max,
          "Found too many items. Expected at most #{expected_max} and found #{found_count}")
      end
    end
  end

end
