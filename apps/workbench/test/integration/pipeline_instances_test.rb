# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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

    # In some locales (e.g., en_CA.UTF-8) Firefox can't parse what its
    # own toLocaleString() puts out.
    t.sub!(/(\d\d\d\d)-(\d\d)-(\d\d)/, '\2/\3/\1')

    if /(\d+:\d+ [AP]M) (\d+\/\d+\/\d+)/ =~ t
      # Currently dates.js renders timestamps as
      # '{t.toLocaleTimeString()} {t.toLocaleDateString()}' which even
      # en_US browsers can't make sense of. First we need to flip it
      # around so it looks like what toLocaleString() would have made.
      t = $~[2] + ', ' + $~[1]
    end

    utc = page.evaluate_script("new Date('#{t}').toUTCString()")
    DateTime.parse(utc).to_time
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
    within(".modal-dialog") do # FIXME: source of 3 test errors
      # Set project for the new pipeline instance
      find(".selectable", text: project_name).click
      click_on "Choose"
    end
    assert(has_text?("This pipeline was created from the template"),
           "did not land on pipeline instance page")
  end

  [
    ['user1_with_load', 'zzzzz-d1hrv-10pipelines0001', 0], # run time 0 minutes
    ['user1_with_load', 'zzzzz-d1hrv-10pipelines0010', 17*60*60 + 51*60], # run time 17 hours and 51 minutes
    ['active', 'zzzzz-d1hrv-runningpipeline', nil], # state = running
  ].each do |user, uuid, run_time|
    test "pipeline start and finish time display for #{uuid}" do
      need_selenium 'to parse timestamps correctly across DST boundaries'
      visit page_with_token(user, "/pipeline_instances/#{uuid}")

      regexp = "This pipeline started at (.+?)\\. "
      if run_time
        regexp += "It failed after (.+?) at (.+?)\\. Check the Log"
      else
        regexp += "It has been active for \\d"
      end
      assert_match /#{regexp}/, page.text

      return if !run_time

      # match again to capture (.*)
      _, started, duration, finished = *(/#{regexp}/.match(page.text))
      assert_equal(
        run_time,
        parse_browser_timestamp(finished) - parse_browser_timestamp(started),
        "expected: #{run_time}, got: started #{started}, finished #{finished}, duration #{duration}")
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
    find(:xpath, "//a[@href='#Log']").click
    within "#Log" do
      assert_text "Log for previous"
      log_link = find("a", text: "Log for previous")
      assert_includes(log_link[:href],
                      "/jobs/#{pi["components"]["previous"]["job"]["uuid"]}#Log")
      assert_selector "#event_log_div"
    end
  end

  test "job logs linked for complete pipeline" do
    pi = api_fixture("pipeline_instances", "complete_pipeline_with_two_jobs")
    visit(page_with_token("active", "/pipeline_instances/#{pi['uuid']}"))
    find(:xpath, "//a[@href='#Log']").click
    within "#Log" do
      assert_text "Log for previous"
      pi["components"].each do |cname, cspec|
        log_link = find("a", text: "Log for #{cname}")
        assert_includes(log_link[:href], "/jobs/#{cspec["job"]["uuid"]}#Log")
      end
      assert_no_selector "#event_log_div"
    end
  end

  test "job logs linked for failed pipeline" do
    pi = api_fixture("pipeline_instances", "failed_pipeline_with_two_jobs")
    visit(page_with_token("active", "/pipeline_instances/#{pi['uuid']}"))
    find(:xpath, "//a[@href='#Log']").click
    within "#Log" do
      assert_text "Log for previous"
      pi["components"].each do |cname, cspec|
        log_link = find("a", text: "Log for #{cname}")
        assert_includes(log_link[:href], "/jobs/#{cspec["job"]["uuid"]}#Log")
      end
      assert_no_selector "#event_log_div"
    end
  end
end
