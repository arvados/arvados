# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'

class WebsocketTest < ActionDispatch::IntegrationTest
  setup do
    need_selenium "to make websockets work"
    @dispatch_client = ArvadosApiClient.new
  end

  def dispatch_log(body)
    use_token :dispatch1 do
      @dispatch_client.api('logs', '', log: body)
    end
  end

  test "test page" do
    visit(page_with_token("active", "/websockets"))
    fill_in("websocket-message-content", :with => "Stuff")
    click_button("Send")
    assert_text '"status":400'
  end

  [
   ['pipeline_instances', 'pipeline_in_running_state', api_fixture('jobs')['running']],
   ['jobs', 'running'],
   ['containers', 'running'],
   ['container_requests', 'running', api_fixture('containers')['running']],
  ].each do |controller, view_fixture_name, log_target_fixture|
    view_fixture = api_fixture(controller)[view_fixture_name]
    log_target_fixture ||= view_fixture

    test "test live logging and scrolling for #{controller}" do

      visit(page_with_token("active", "/#{controller}/#{view_fixture['uuid']}\#Log"))
      assert_no_text '123 hello'

      text = ""
      (1..1000).each do |i|
        text << "#{i} hello\n"
      end

      dispatch_log(owner_uuid: log_target_fixture['owner_uuid'],
                   object_uuid: log_target_fixture['uuid'],
                   event_type: "stderr",
                   properties: {"text" => text})
      assert_text '1000 hello'

      # First test that when we're already at the bottom of the page, it scrolls down
      # when a new line is added.
      old_top = page.evaluate_script("$('#event_log_div').scrollTop()")

      dispatch_log(owner_uuid: log_target_fixture['owner_uuid'],
                   object_uuid: log_target_fixture['uuid'],
                   event_type: "dispatch",
                   properties: {"text" => "1001 hello\n"})
      assert_text '1001 hello'

      # Check that new value of scrollTop is greater than the old one
      new_top = page.evaluate_script("$('#event_log_div').scrollTop()")
      assert_operator new_top, :>, old_top

      # Now scroll to 30 pixels from the top
      page.execute_script "$('#event_log_div').scrollTop(30)"
      assert_equal 30, page.evaluate_script("$('#event_log_div').scrollTop()")

      dispatch_log(owner_uuid: log_target_fixture['owner_uuid'],
                   object_uuid: log_target_fixture['uuid'],
                   event_type: "stdout",
                   properties: {"text" => "1002 hello\n"})
      assert_text '1002 hello'

      # Check that we haven't changed scroll position
      assert_equal 30, page.evaluate_script("$('#event_log_div').scrollTop()")
    end
  end

  test "pipeline instance arv-refresh-on-log-event" do
    # Do something and check that the pane reloads.
    p = use_token :active do
      PipelineInstance.create(state: "RunningOnServer",
                              components: {
                                c1: {
                                  script: "test_hash.py",
                                  script_version: "1de84a854e2b440dc53bf42f8548afa4c17da332"
                                }
                              })
    end
    visit(page_with_token("active", "/pipeline_instances/#{p.uuid}"))

    assert_text 'Active'
    assert page.has_link? 'Pause'
    assert_no_text 'Complete'
    assert page.has_no_link? 'Re-run with latest'

    use_token :dispatch1 do
      p.update_attributes!(state: 'Complete')
    end

    assert_no_text 'Active'
    assert page.has_no_link? 'Pause'
    assert_text 'Complete'
    assert page.has_link? 'Re-run with latest'
  end

  test "job arv-refresh-on-log-event" do
    # Do something and check that the pane reloads.
    uuid = api_fixture('jobs')['running_will_be_completed']['uuid']
    visit(page_with_token("active", "/jobs/#{uuid}"))

    assert_no_text 'complete'
    assert_no_text 'Re-run job'

    use_token :dispatch1 do
      Job.find(uuid).update_attributes!(state: 'Complete')
    end

    assert_text 'complete'
    assert_text 'Re-run job'
  end

  test "dashboard arv-refresh-on-log-event" do
    visit(page_with_token("active", "/"))

    assert_no_text 'test dashboard arv-refresh-on-log-event'

    # Do something and check that the pane reloads.
    use_token :active do
      p = PipelineInstance.create({state: "RunningOnServer",
                                    name: "test dashboard arv-refresh-on-log-event",
                                    components: {
                                    }
                                  })
    end

    assert_text 'test dashboard arv-refresh-on-log-event'
  end

  test 'job graph appears when first data point is already in logs table' do
    job_graph_first_datapoint_test
  end

  test 'job graph appears when first data point arrives by websocket' do
    use_token :admin do
      Log.find(api_fixture('logs')['crunchstat_for_running_job']['uuid']).destroy
    end
    job_graph_first_datapoint_test expect_existing_datapoints: false
  end

  def job_graph_first_datapoint_test expect_existing_datapoints: true
    uuid = api_fixture('jobs')['running']['uuid']

    visit page_with_token "active", "/jobs/#{uuid}"
    click_link "Log"

    assert_selector '#event_log_div', visible: true

    if expect_existing_datapoints
      assert_selector '#log_graph_div', visible: true
      # Magic numbers 12.99 etc come from the job log fixture:
      assert_last_datapoint 'T1-cpu', (((12.99+0.99)/10.0002)/8)
    else
      # Until graphable data arrives, we should see the text log but not the graph.
      assert_no_selector '#log_graph_div', visible: true
    end

    text = "2014-11-07_23:33:51 #{uuid} 31708 1 stderr crunchstat: cpu 1970.8200 user 60.2700 sys 8 cpus -- interval 10.0002 seconds 35.3900 user 0.8600 sys"

    assert_triggers_dom_event 'arv-log-event' do
      dispatch_log(owner_uuid: api_fixture('jobs')['running']['owner_uuid'],
                   object_uuid: uuid,
                   event_type: "stderr",
                   properties: {"text" => text})
    end

    # Graph should have appeared (even if it hadn't above). It's
    # important not to wait like matchers usually do: we are
    # confirming the graph is visible _immediately_ after the first
    # data point arrives.
    using_wait_time 0 do
      assert_selector '#log_graph_div', visible: true
    end
    assert_last_datapoint 'T1-cpu', (((35.39+0.86)/10.0002)/8)
  end

  test "live log charting from replayed log" do
    uuid = api_fixture("jobs")['running']['uuid']

    visit page_with_token "active", "/jobs/#{uuid}"
    click_link "Log"

    assert_triggers_dom_event 'arv-log-event' do
      ApiServerForTests.new.run_rake_task("replay_job_log", "test/job_logs/crunchstatshort.log,1.0,#{uuid}")
    end

    assert_last_datapoint 'T1-cpu', (((35.39+0.86)/10.0002)/8)
  end

  def assert_last_datapoint series, value
    datum = page.evaluate_script("jobGraphData[jobGraphData.length-1]['#{series}']")
    assert_in_epsilon value, datum.to_f
  end

  test "test running job with just a few previous log records" do
    job = api_fixture("jobs")['running']

    # Create just one old log record
    dispatch_log(owner_uuid: job['owner_uuid'],
                 object_uuid: job['uuid'],
                 event_type: "stderr",
                 properties: {"text" => "Historic log message"})

    visit page_with_token("active", "/jobs/#{job['uuid']}\#Log")

    # Expect "all" historic log records because we have less than
    # default Rails.configuration.running_job_log_records_to_fetch count
    assert_text 'Historic log message'

    # Create new log record and expect it to show up in log tab
    dispatch_log(owner_uuid: job['owner_uuid'],
                 object_uuid: job['uuid'],
                 event_type: "stderr",
                 properties: {"text" => "Log message after subscription"})
    assert_text 'Log message after subscription'
  end

  test "test running job with too many previous log records" do
    max = 5
    Rails.configuration.running_job_log_records_to_fetch = max
    job = api_fixture("jobs")['running']

    # Create max+1 log records
    (0..max).each do |count|
      dispatch_log(owner_uuid: job['owner_uuid'],
                   object_uuid: job['uuid'],
                   event_type: "stderr",
                   properties: {"text" => "Old log message #{count}"})
    end

    visit page_with_token("active", "/jobs/#{job['uuid']}\#Log")

    # Expect all but the first historic log records,
    # because that was one too many than fetch count.
    (1..max).each do |count|
      assert_text "Old log message #{count}"
    end
    assert_no_text 'Old log message 0'

    # Create one more log record after subscription
    dispatch_log(owner_uuid: job['owner_uuid'],
                 object_uuid: job['uuid'],
                 event_type: "stderr",
                 properties: {"text" => "Life goes on!"})

    # Expect it to show up in log tab
    assert_text 'Life goes on!'
  end
end
