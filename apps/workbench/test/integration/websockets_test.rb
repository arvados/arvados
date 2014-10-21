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

end
