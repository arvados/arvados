require 'integration_helper'
require 'selenium-webdriver'
require 'headless'

class WebsocketTest < ActionDispatch::IntegrationTest

  setup do
    headless = Headless.new
    headless.start
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

  test "test live logging scrolling" do
    visit(page_with_token("admin", "/pipeline_instances/zzzzz-d1hrv-9fm8l10i9z2kqc6"))
    click_link("Log")
    assert_no_text '123 hello'

    api = ArvadosApiClient.new

    text = ""
    (1..1000).each do |i|
      text << "#{i} hello\n"
    end

    Thread.current[:arvados_api_token] = @@API_AUTHS["admin"]['api_token']
    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => text}}})
    assert_text '1000 hello'

    # First test that when we're already at the bottom of the page, it scrolls down
    # when a new line is added.
    old_top = page.evaluate_script("$('#pipeline_event_log_div').scrollTop()")

    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => "1001 hello\n"}}})
    assert_text '1001 hello'

    # Check that new value of scrollTop is greater than the old one
    assert page.evaluate_script("$('#pipeline_event_log_div').scrollTop()") > old_top

    # Now scroll to 30 pixels from the top
    page.execute_script "$('#pipeline_event_log_div').scrollTop(30)"
    assert_equal 30, page.evaluate_script("$('#pipeline_event_log_div').scrollTop()")

    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => "1002 hello\n"}}})
    assert_text '1002 hello'

    # Check that we haven't changed scroll position
    assert_equal 30, page.evaluate_script("$('#pipeline_event_log_div').scrollTop()")

    Thread.current[:arvados_api_token] = nil
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

end
