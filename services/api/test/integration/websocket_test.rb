require 'test_helper'
require 'websocket_runner'
require 'oj'

class WebsocketTest < ActionDispatch::IntegrationTest

  def ws_helper (token = nil)
    EM.run {
      if token
        ws = Faye::WebSocket::Client.new("ws://localhost:3002/websocket?api_token=#{api_client_authorizations(token).api_token}")
      else
        ws = Faye::WebSocket::Client.new("ws://localhost:3002/websocket")
      end

      ws.on :close do |event|
        EM.stop_event_loop
      end

      EM::Timer.new 3 do
        puts "\nTest took too long"
        EM.stop_event_loop
      end

      yield ws
    }
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


  # test "connect, subscribe, get event" do
  #   opened = false
  #   state = 1
  #   spec_uuid = nil
  #   ev_uuid = nil

  #   puts "user #{Thread.current[:user]}"
  #   authorize_with :admin
  #   puts "user #{Thread.current[:user]}"

  #   ws_helper :admin do |ws|
  #     ws.on :open do |event|
  #       puts "XXX"
  #       opened = true
  #       ws.send ({method: 'subscribe'}.to_json)
  #     end

  #     ws.on :message do |event|
  #       d = Oj.load event.data
  #       puts d
  #       case state
  #       when 1
  #         assert_equal 200, d["status"]
  #         spec_uuid = Specimen.create.save.uuid
  #         state = 2
  #       when 2
  #         ev_uuid = d["uuid"]
  #         ws.close
  #       end
  #     end

  #   end

  #   assert opened, "Should have opened web socket"
  #   assert_not spec_uuid.nil?
  #   assert_equal spec_uuid, ev_uuid
  # end

end
