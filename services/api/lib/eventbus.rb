require 'eventmachine'
require 'oj'
require 'faye/websocket'

module Faye
  class WebSocket
    attr_accessor :user
    attr_accessor :last_log_id
  end
end

class EventBus
  include CurrentApiClient

  def initialize
    @channel = EventMachine::Channel.new
    @mtx = Mutex.new
    @bgthread = false
  end

  def on_connect ws
    if not current_user
      ws.send '{"error":"Not logged in"}'
      ws.close
      return
    end

    ws.user = current_user

    sub = @channel.subscribe do |msg|
      Log.where(id: msg.to_i).each do |l|
        if rsc = ArvadosModel::resource_class_for_uuid(l.object_uuid)
          rsc.readable_by(ws.user).where(uuid: l.object_uuid).each do
            ws.send(l.as_api_response.to_json)
          end
        end
        ws.last_log_id = msg.to_i
      end
    end

    ws.on :message do |event|
      #puts "got #{event.data}"
    end

    ws.on :close do |event|
      @channel.unsubscribe sub
      ws = nil
    end

    @mtx.synchronize do
      unless @bgthread
        @bgthread = true
        Thread.new do
          # from http://stackoverflow.com/questions/16405520/postgres-listen-notify-rails
          ActiveRecord::Base.connection_pool.with_connection do |connection|
            conn = connection.instance_variable_get(:@connection)
            begin
              conn.async_exec "LISTEN logs"
              while true
                conn.wait_for_notify do |channel, pid, payload|
                  @channel.push payload
                end
              end
            ensure
              # Don't want the connection to still be listening once we return
              # it to the pool - could result in weird behavior for the next
              # thread to check it out.
              conn.async_exec "UNLISTEN *"
            end
          end
        end
      end
    end
  end
end
