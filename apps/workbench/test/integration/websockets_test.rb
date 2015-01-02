require 'integration_helper'

class WebsocketTest < ActionDispatch::IntegrationTest
  setup do
    Capybara.current_driver = :selenium
  end

  test "test page" do
    visit(page_with_token("admin", "/websockets"))
    fill_in("websocket-message-content", :with => "Stuff")
    click_button("Send")
    assert_text '"status":400'
  end

  test "test live logging" do
    visit(page_with_token("admin", "/pipeline_instances/zzzzz-d1hrv-9fm8l10i9z2kqc6"))
    click_link("Log")
    assert_no_text '123 hello'

    api = ArvadosApiClient.new

    Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']
    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => "123 hello"}}})
    assert_text '123 hello'
    Thread.current[:arvados_api_token] = nil
  end


  [["pipeline_instances", api_fixture("pipeline_instances")['pipeline_with_newer_template']['uuid']],
   ["jobs", api_fixture("jobs")['running']['uuid']]].each do |c|
    test "test live logging scrolling #{c[0]}" do

      controller = c[0]
      uuid = c[1]

      visit(page_with_token("admin", "/#{controller}/#{uuid}"))
      click_link("Log")
      assert_no_text '123 hello'

      api = ArvadosApiClient.new

      text = ""
      (1..1000).each do |i|
        text << "#{i} hello\n"
      end

      Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']
      api.api("logs", "", {log: {
                  object_uuid: uuid,
                  event_type: "stderr",
                  properties: {"text" => text}}})
      assert_text '1000 hello'

      # First test that when we're already at the bottom of the page, it scrolls down
      # when a new line is added.
      old_top = page.evaluate_script("$('#event_log_div').scrollTop()")

      api.api("logs", "", {log: {
                  object_uuid: uuid,
                  event_type: "stderr",
                  properties: {"text" => "1001 hello\n"}}})
      assert_text '1001 hello'

      # Check that new value of scrollTop is greater than the old one
      assert page.evaluate_script("$('#event_log_div').scrollTop()") > old_top

      # Now scroll to 30 pixels from the top
      page.execute_script "$('#event_log_div').scrollTop(30)"
      assert_equal 30, page.evaluate_script("$('#event_log_div').scrollTop()")

      api.api("logs", "", {log: {
                  object_uuid: uuid,
                  event_type: "stderr",
                  properties: {"text" => "1002 hello\n"}}})
      assert_text '1002 hello'

      # Check that we haven't changed scroll position
      assert_equal 30, page.evaluate_script("$('#event_log_div').scrollTop()")

      Thread.current[:arvados_api_token] = nil
    end
  end

  test "pipeline instance arv-refresh-on-log-event" do
    Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']
    # Do something and check that the pane reloads.
    p = PipelineInstance.create({state: "RunningOnServer",
                                  components: {
                                    c1: {
                                      script: "test_hash.py",
                                      script_version: "1de84a854e2b440dc53bf42f8548afa4c17da332"
                                    }
                                  }
                                })

    visit(page_with_token("admin", "/pipeline_instances/#{p.uuid}"))

    assert_text 'Active'
    assert page.has_link? 'Pause'
    assert_no_text 'Complete'
    assert page.has_no_link? 'Re-run with latest'

    p.state = "Complete"
    p.save!

    assert_no_text 'Active'
    assert page.has_no_link? 'Pause'
    assert_text 'Complete'
    assert page.has_link? 'Re-run with latest'

    Thread.current[:arvados_api_token] = nil
  end

  test "job arv-refresh-on-log-event" do
    Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']
    # Do something and check that the pane reloads.
    p = Job.where(uuid: api_fixture('jobs')['running_will_be_completed']['uuid']).results.first

    visit(page_with_token("admin", "/jobs/#{p.uuid}"))

    assert_no_text 'complete'
    assert_no_text 'Re-run same version'

    p.state = "Complete"
    p.save!

    assert_text 'complete'
    assert_text 'Re-run same version'

    Thread.current[:arvados_api_token] = nil
  end

  test "dashboard arv-refresh-on-log-event" do
    Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']

    visit(page_with_token("admin", "/"))

    assert_no_text 'test dashboard arv-refresh-on-log-event'

    # Do something and check that the pane reloads.
    p = PipelineInstance.create({state: "RunningOnServer",
                                  name: "test dashboard arv-refresh-on-log-event",
                                  components: {
                                  }
                                })

    assert_text 'test dashboard arv-refresh-on-log-event'

    Thread.current[:arvados_api_token] = nil
  end

  test "live log charting" do
    uuid = api_fixture("jobs")['running']['uuid']

    visit page_with_token "admin", "/jobs/#{uuid}"
    click_link "Log"

    api = ArvadosApiClient.new

    # should give 45.3% or (((36.39+0.86)/10.0002)/8)*100 rounded to 1 decimal place
    text = "2014-11-07_23:33:51 #{uuid} 31708 1 stderr crunchstat: cpu 1970.8200 user 60.2700 sys 8 cpus -- interval 10.0002 seconds 35.3900 user 0.8600 sys"

    Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']
    api.api("logs", "", {log: {
                object_uuid: uuid,
                event_type: "stderr",
                properties: {"text" => text}}})
    wait_for_ajax

    # using datapoint 1 instead of datapoint 0 because there will be a "dummy" datapoint with no actual stats 10 minutes previous to the one we're looking for, for the sake of making the x-axis of the graph show a full 10 minutes of time even though there is only a single real datapoint
    cpu_stat = page.evaluate_script("jobGraphData[1]['T1-cpu']")

    assert_equal 45.3, (cpu_stat.to_f*100).round(1)

    Thread.current[:arvados_api_token] = nil
  end

  test "live log charting from replayed log" do
    uuid = api_fixture("jobs")['running']['uuid']

    visit page_with_token "admin", "/jobs/#{uuid}"
    click_link "Log"

    ApiServerForTests.new.run_rake_task("replay_job_log", "test/job_logs/crunchstatshort.log,1.0,#{uuid}")
    wait_for_ajax

    # see above comment as to why we use datapoint 1 rather than 0
    cpu_stat = page.evaluate_script("jobGraphData[1]['T1-cpu']")

    assert_equal 45.3, (cpu_stat.to_f*100).round(1)
  end

end
