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

end
