require 'integration_helper'

class PipelineInstancesTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  def parse_browser_timestamp t
    # Timestamps are displayed in the browser's time zone (which can
    # differ from ours) and they come from toLocaleTimeString (which
    # means they don't necessarily tell us which time zone they're
    # using). In order to make sense of them, we need to ask the
    # browser to parse them and generate a timestamp that can be
    # parsed reliably.
    #
    # Note: Even with all this help, phantomjs seem to behave badly
    # when parsing timestamps on the other side of a DST transition.
    # See skipped tests below.
    if /(\d+:\d+ [AP]M) (\d+\/\d+\/\d+)/ =~ t
      # Currently dates.js renders timestamps as
      # '{t.toLocaleTimeString()} {t.toLocaleDateString()}' which even
      # browsers can't make sense of. First we need to flip it around
      # so it looks like what toLocaleString() would have made.
      t = $~[2] + ', ' + $~[1]
    end
    DateTime.parse(page.evaluate_script "new Date('#{t}').toUTCString()").to_time
  end

  if false
    # No need to test (or mention) these all the time. If they start
    # working (without need_selenium) then some real tests might not
    # need_selenium any more.

    test 'phantomjs DST' do
      skip '^^'
      t0s = '3/8/2015, 01:59 AM'
      t1s = '3/8/2015, 03:01 AM'
      t0 = parse_browser_timestamp t0s
      t1 = parse_browser_timestamp t1s
      assert_equal 120, t1-t0, "'#{t0s}' to '#{t1s}' was reported as #{t1-t0} seconds, should be 120"
    end

    test 'phantomjs DST 2' do
      skip '^^'
      t0s = '2015-03-08T10:43:00Z'
      t1s = '2015-03-09T03:43:00Z'
      t0 = parse_browser_timestamp page.evaluate_script("new Date('#{t0s}').toLocaleString()")
      t1 = parse_browser_timestamp page.evaluate_script("new Date('#{t1s}').toLocaleString()")
      assert_equal 17*3600, t1-t0, "'#{t0s}' to '#{t1s}' was reported as #{t1-t0} seconds, should be #{17*3600} (17 hours)"
    end
  end

  test 'Create and run a pipeline' do
    visit page_with_token('active_trustedclient', '/pipeline_templates')
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
    find('.dropdown-menu a,button', text: 'Copy data from another project').click
    within('.modal-dialog') do
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('.btn', text: 'Copy').click
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
      find('.btn', text: 'Choose').click

    within('.modal-dialog') do
      assert(has_text?("Foo/bar pair"),
             "pipeline input picker missing name of input")
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('button', text: 'OK').click
    end

    # For good measure, check one last time that the input, after being specified twice, is still be displayed (#3382)
    assert find('div.form-group', text: 'Foo/bar pair')

    # Ensure that the collection's portable_data_hash, uuid and name
    # are saved in the desired places. (#4015)

    # foo_collection_in_aproject is the collection tagged with foo_tag.
    collection = api_fixture('collections', 'foo_collection_in_aproject')
    click_link 'Advanced'
    click_link 'API response'
    api_response = JSON.parse(find('div#advanced_api_response pre').text)
    input_params = api_response['components']['part-one']['script_parameters']['input']
    assert_equal input_params['value'], collection['portable_data_hash']
    assert_equal input_params['selection_name'], collection['name']
    assert_equal input_params['selection_uuid'], collection['uuid']

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
    visit page_with_token('active_trustedclient', '/projects')

    # Add collection to the project using Add data button
    find("#projects-menu").click
    find('.dropdown-menu a,button', text: 'A Project').click
    find('.btn', text: 'Add data').click
    find('.dropdown-menu a,button', text: 'Copy data from another project').click
    within('.modal-dialog') do
      wait_for_ajax
      first('span', text: 'foo_tag').click
      find('.btn', text: 'Copy').click
    end
    using_wait_time(Capybara.default_wait_time * 3) do
      wait_for_ajax
    end

    create_and_run_pipeline_in_aproject true, 'Two Part Pipeline Template', 'foo_collection_in_aproject', false
  end

  # Create a pipeline instance from outside of a project
  test 'Run a pipeline from dashboard' do
    visit page_with_token('active_trustedclient')
    create_and_run_pipeline_in_aproject false, 'Two Part Pipeline Template', 'foo_collection_in_aproject', false
  end

  test 'view pipeline with job and see graph' do
    visit page_with_token('active_trustedclient', '/pipeline_instances')
    assert page.has_text? 'pipeline_with_job'

    find('a', text: 'pipeline_with_job').click

    # since the pipeline component has a job, expect to see the graph
    assert page.has_text? 'Graph'
    click_link 'Graph'
    page.assert_selector "#provenance_graph"
  end

  test 'pipeline description' do
    visit page_with_token('active_trustedclient', '/pipeline_instances')
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

  def create_pipeline_from(template_name, project_name="Home")
    # Visit the named pipeline template and create a pipeline instance from it.
    # The instance will be created under the named project.
    template_uuid = api_fixture("pipeline_templates", template_name, "uuid")
    visit page_with_token("active", "/pipeline_templates/#{template_uuid}")
    click_on "Run this pipeline"
    within(".modal-dialog") do
      # Set project for the new pipeline instance
      find(".selectable", text: project_name).click
      click_on "Choose"
    end
    assert(has_text?("This pipeline was created from the template"),
           "did not land on pipeline instance page")
  end

  PROJECT_WITH_SEARCH_COLLECTION = "A Subproject"
  def check_parameter_search(proj_name)
    create_pipeline_from("parameter_with_search", proj_name)
    search_text = api_fixture("pipeline_templates", "parameter_with_search",
                              "components", "with-search",
                              "script_parameters", "input", "search_for")
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

  test "enter a float for a number pipeline input" do
    # Poltergeist either does not support the HTML 5 <input
    # type="number">, or interferes with the associated X-Editable
    # validation code.  If the input field has type=number (forcing an
    # integer), this test will yield a false positive under
    # Poltergeist.  --Brett, 2015-02-05
    need_selenium "for strict X-Editable input validation"
    create_pipeline_from("template_with_dataclass_number")
    INPUT_SELECTOR =
      ".editable[data-name='[components][work][script_parameters][input][value]']"
    find(INPUT_SELECTOR).click
    find(".editable-input input").set("12.34")
    find("#editable-submit").click
    assert_no_selector(".editable-popup")
    assert_selector(INPUT_SELECTOR, text: "12.34")
  end

  [
    [true, 'Two Part Pipeline Template', 'foo_collection_in_aproject', false],
    [false, 'Two Part Pipeline Template', 'foo_collection_in_aproject', false],
    [true, 'Two Part Template with dataclass File', 'foo_collection_in_aproject', true],
    [false, 'Two Part Template with dataclass File', 'foo_collection_in_aproject', true],
    [true, 'Two Part Pipeline Template', 'collection_with_no_name_in_aproject', false],
  ].each do |in_aproject, template_name, collection, choose_file|
    test "Run pipeline instance in #{in_aproject} with #{template_name} with #{collection} file #{choose_file}" do
      if in_aproject
        visit page_with_token 'active', \
        '/projects/'+api_fixture('groups')['aproject']['uuid']
      else
        visit page_with_token 'active', '/'
      end

      # need bigger modal size when choosing a file from collection
      if Capybara.current_driver == :selenium
        Capybara.current_session.driver.browser.manage.window.resize_to(1200, 800)
      end

      create_and_run_pipeline_in_aproject in_aproject, template_name, collection, choose_file
      instance_path = current_path

      # Pause the pipeline
      find('a,button', text: 'Pause').click
      assert page.has_text? 'Paused'
      page.assert_no_selector 'a.disabled,button.disabled', text: 'Resume'
      page.assert_selector 'a,button', text: 'Re-run with latest'
      page.assert_selector 'a,button', text: 'Re-run options'

      # Verify that the newly created instance is created in the right project.
      assert page.has_text? 'Home'
      if in_aproject
        assert page.has_text? 'A Project'
      else
        assert page.has_no_text? 'A Project'
      end
    end
  end

  [
    ['active', false, false, false],
    ['active', false, false, true],
    ['active', true, false, false],
    ['active', true, true, false],
    ['active', true, false, true],
    ['active', true, true, true],
    ['project_viewer', false, false, true],
    ['project_viewer', true, true, true],
  ].each do |user, with_options, choose_options, in_aproject|
    test "Rerun pipeline instance as #{user} using options #{with_options} #{choose_options} in #{in_aproject}" do
      if in_aproject
        path = '/pipeline_instances/'+api_fixture('pipeline_instances')['pipeline_owned_by_active_in_aproject']['uuid']
      else
        path = '/pipeline_instances/'+api_fixture('pipeline_instances')['pipeline_owned_by_active_in_home']['uuid']
      end

      visit page_with_token(user, path)

      page.assert_selector 'a,button', text: 'Re-run with latest'
      page.assert_selector 'a,button', text: 'Re-run options'

      if user == 'project_viewer' && in_aproject
        assert page.has_text? 'A Project'
      end

      # Now re-run the pipeline
      if with_options
        assert_triggers_dom_event 'shown.bs.modal' do
          find('a,button', text: 'Re-run options').click
        end
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

      # Verify that the newly created instance is created in the right
      # project. In case of project_viewer user, since the user cannot
      # write to the project, the pipeline should have been created in
      # the user's Home project.
      assert_not_equal path, current_path, 'Rerun instance path expected to be different'
      assert_text 'Home'
      if in_aproject && (user != 'project_viewer')
        assert_text 'A Project'
      else
        assert_no_text 'A Project'
      end
    end
  end

  # Create and run a pipeline for 'Two Part Pipeline Template' in 'A Project'
  def create_and_run_pipeline_in_aproject in_aproject, template_name, collection_fixture, choose_file=false
    # collection in aproject to be used as input
    collection = api_fixture('collections', collection_fixture)

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

      if collection_fixture == 'foo_collection_in_aproject'
        first('span', text: 'foo_tag').click
      elsif collection['name']
        first('span', text: "#{collection['name']}").click
      else
        collection_uuid = collection['uuid']
        find("div[data-object-uuid=#{collection_uuid}]").click
      end

      if choose_file
        wait_for_ajax
        find('.preview-selectable', text: 'foo').click
      end
      find('button', text: 'OK').click
    end

    # The input, after being specified, should still be displayed (#3382)
    assert find('div.form-group', text: 'Foo/bar pair')

    # Ensure that the collection's portable_data_hash, uuid and name
    # are saved in the desired places. (#4015)
    click_link 'Advanced'
    click_link 'API response'

    api_response = JSON.parse(find('div#advanced_api_response pre').text)
    input_params = api_response['components']['part-one']['script_parameters']['input']
    assert_equal(input_params['selection_uuid'], collection['uuid'], "Not found expected input param uuid")
    if choose_file
      assert_equal(input_params['value'], collection['portable_data_hash']+'/foo', "Not found expected input file param value")
      assert_equal(input_params['selection_name'], collection['name']+'/foo', "Not found expected input file param name")
    else
      assert_equal(input_params['value'], collection['portable_data_hash'], "Not found expected input param value")
      assert_equal(input_params['selection_name'], collection['name'], "Not found expected input selection name")
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
    ['user1_with_load', 'zzzzz-d1hrv-10pipelines0001', 0], # run time 0 minutes
    ['user1_with_load', 'zzzzz-d1hrv-10pipelines0010', 17*60*60 + 51*60], # run time 17 hours and 51 minutes
    ['active', 'zzzzz-d1hrv-runningpipeline', nil], # state = running
  ].each do |user, uuid, run_time|
    test "pipeline start and finish time display for #{uuid}" do
      need_selenium 'to parse timestamps correctly across DST boundaries'
      visit page_with_token(user, "/pipeline_instances/#{uuid}")

      assert page.has_text? 'This pipeline started at'
      page_text = page.text

      if run_time
        match = /This pipeline started at (.*)\. It failed after (.*) at (.*)\. Check the Log/.match page_text
      else
        match = /This pipeline started at (.*). It has been active for(.*)/.match page_text
      end
      assert_not_nil(match, 'Did not find text - This pipeline started at . . . ')

      start_at = match[1]
      assert_not_nil(start_at, 'Did not find start_at time')

      start_time = parse_browser_timestamp start_at
      if run_time
        finished_at = match[3]
        assert_not_nil(finished_at, 'Did not find finished_at time')
        finished_time = parse_browser_timestamp finished_at
        assert_equal(run_time, finished_time-start_time,
          "Time difference did not match for start_at #{start_at}, finished_at #{finished_at}, ran_for #{match[2]}")
      else
        match = /\d(.*)/.match match[2]
        assert_not_nil match, 'Did not find expected match for running component'
      end
    end
  end

  [
    ['fuse', nil, 2, 20],                           # has 2 as of 11-07-2014
    ['user1_with_load', '000025pipelines', 25, 25], # owned_by the project zzzzz-j7d0g-000025pipelines, two pages
    ['admin', 'pipeline_20', 1, 1],
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
          page.driver.scroll_to 0, 999000
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

  test 'render job run time when job record is inaccessible' do
    pi = api_fixture('pipeline_instances', 'has_component_with_completed_jobs')
    visit page_with_token 'active', '/pipeline_instances/' + pi['uuid']
    assert_text 'Queued for '
  end

  test "job logs linked for running pipeline" do
    pi = api_fixture("pipeline_instances", "running_pipeline_with_complete_job")
    visit(page_with_token("active", "/pipeline_instances/#{pi['uuid']}"))
    click_on "Log"
    within "#Log" do
      assert_text "Log for previous"
      log_link = find("a", text: "Log for previous")
      assert_includes(log_link[:href],
                      pi["components"]["previous"]["job"]["log"])
      assert_selector "#event_log_div"
    end
  end

  test "job logs linked for complete pipeline" do
    pi = api_fixture("pipeline_instances", "complete_pipeline_with_two_jobs")
    visit(page_with_token("active", "/pipeline_instances/#{pi['uuid']}"))
    click_on "Log"
    within "#Log" do
      assert_text "Log for previous"
      pi["components"].each do |cname, cspec|
        log_link = find("a", text: "Log for #{cname}")
        assert_includes(log_link[:href], cspec["job"]["log"])
      end
      assert_no_selector "#event_log_div"
    end
  end

  test "job logs linked for failed pipeline" do
    pi = api_fixture("pipeline_instances", "failed_pipeline_with_two_jobs")
    visit(page_with_token("active", "/pipeline_instances/#{pi['uuid']}"))
    click_on "Log"
    within "#Log" do
      assert_text "Log for previous"
      pi["components"].each do |cname, cspec|
        log_link = find("a", text: "Log for #{cname}")
        assert_includes(log_link[:href], cspec["job"]["log"])
      end
      assert_no_selector "#event_log_div"
    end
  end
end
