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
    visit(page_with_token("active", "/websockets"))
    fill_in("websocket-message-content", :with => "Stuff")
    click_button("Send")
    assert page.has_text? '"status":400'
  end

  test "test live logging" do
    visit(page_with_token("active", "/pipeline_instances/zzzzz-d1hrv-9fm8l10i9z2kqc6"))
    click_link("Log")
    assert page.has_no_text? '123 hello'

    api = ArvadosApiClient.new

    Thread.current[:arvados_api_token] = @@API_AUTHS["active"]['api_token']
    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => "123 hello"}}})
    assert page.has_text? '123 hello'
    Thread.current[:arvados_api_token] = nil
  end

  test "test live logging scrolling" do
    visit(page_with_token("active", "/pipeline_instances/zzzzz-d1hrv-9fm8l10i9z2kqc6"))
    click_link("Log")
    assert page.has_no_text? '123 hello'

    api = ArvadosApiClient.new

    text = ""
    (1..1000).each do |i|
      text << "#{i} hello\n"
    end

    Thread.current[:arvados_api_token] = @@API_AUTHS["active"]['api_token']
    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => text}}})
    assert page.has_text? '1000 hello'

    # First test that when we're already at the bottom of the page, it scrolls down
    # when a new line is added.
    old_top = page.evaluate_script("$('#pipeline_event_log_div').scrollTop()")

    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => "1001 hello\n"}}})
    assert page.has_text? '1001 hello'

    # Check that new value of scrollTop is greater than the old one
    assert page.evaluate_script("$('#pipeline_event_log_div').scrollTop()") > old_top

    # Now scroll to 30 pixels from the top
    page.execute_script "$('#pipeline_event_log_div').scrollTop(30)"
    assert_equal 30, page.evaluate_script("$('#pipeline_event_log_div').scrollTop()")

    api.api("logs", "", {log: {
                object_uuid: "zzzzz-d1hrv-9fm8l10i9z2kqc6",
                event_type: "stderr",
                properties: {"text" => "1002 hello\n"}}})
    assert page.has_text? '1002 hello'

    # Check that we haven't changed scroll position
    assert_equal 30, page.evaluate_script("$('#pipeline_event_log_div').scrollTop()")

    Thread.current[:arvados_api_token] = nil
  end

end
