require 'test_helper'
require 'websocket_runner'
require 'oj'
require 'database_cleaner'

DatabaseCleaner.strategy = :deletion

class WebsocketTest < ActionDispatch::IntegrationTest
  self.use_transactional_fixtures = false

  setup do
    DatabaseCleaner.start
  end

  teardown do
    DatabaseCleaner.clean
  end

  def ws_helper (token = nil)
    close_status = nil

    EM.run {
      if token
        ws = Faye::WebSocket::Client.new("ws://localhost:3002/websocket?api_token=#{api_client_authorizations(token).api_token}")
      else
        ws = Faye::WebSocket::Client.new("ws://localhost:3002/websocket")
      end

      ws.on :close do |event|
        close_status = [:close, event.code, event.reason]
        EM.stop_event_loop
      end

      EM::Timer.new 3 do
        EM.stop_event_loop
      end

      yield ws
    }

    assert_not_nil close_status, "Test took too long"
    assert_equal 1000, close_status[1], "Server closed the connection unexpectedly (check server log for errors)"
  end

  test "connect with no token" do
    opened = false
    status = nil

    ws_helper do |ws|
      ws.on :open do |event|
        opened = true
      end

      ws.on :message do |event|
        d = Oj.load event.data
        status = d["status"]
        ws.close
      end
    end

    assert opened, "Should have opened web socket"
    assert_equal 401, status
  end


  test "connect, subscribe and get response" do
    opened = false
    status = nil

    ws_helper :admin do |ws|
      ws.on :open do |event|
        opened = true
        ws.send ({method: 'subscribe'}.to_json)
      end

      ws.on :message do |event|
        d = Oj.load event.data
        status = d["status"]
        ws.close
      end
    end

    assert opened, "Should have opened web socket"
    assert_equal 200, status
  end

  test "connect, subscribe, get event" do
    opened = false
    state = 1
    spec_uuid = nil
    ev_uuid = nil

    authorize_with :admin

    ws_helper :admin do |ws|
      ws.on :open do |event|
        opened = true
        ws.send ({method: 'subscribe'}.to_json)
      end

      ws.on :message do |event|
        d = Oj.load event.data
        case state
        when 1
          assert_equal 200, d["status"]
          spec = Specimen.create
          spec.save
          spec_uuid = spec.uuid
          state = 2
        when 2
          ev_uuid = d["object_uuid"]
          ws.close
        end
      end

    end

    assert opened, "Should have opened web socket"
    assert_not_nil spec_uuid
    assert_equal spec_uuid, ev_uuid
  end

end
