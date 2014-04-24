require 'test_helper'
require 'websocket_runner'

class WebsocketTest < ActionDispatch::IntegrationTest

  test "just connect" do
    opened = false
    EM.run {
      ws = Faye::WebSocket::Client.new('ws://localhost:3002/websocket')

      ws.on :open do |event|
        opened = true
        ws.close
      end

      ws.on :close do |event|
        p [:close, event.code, event.reason]
        EM.stop_event_loop
      end
    }

    assert opened, "Should have opened web socket"
  end

end
