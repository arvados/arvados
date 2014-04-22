require 'eventmachine'
require 'oj'
require 'faye/websocket'

class EventBus
  def initialize
    @channel = EventMachine::Channel.new
    @bgthread = nil
  end

  def on_connect ws
      sub = @channel.subscribe do |msg|
        puts "sending #{msg}"
        ws.send({:message => "log"}.to_json)
      end

      ws.on :message do |event|
        puts "got #{event.data}"
        ws.send(event.data)
      end

      ws.on :close do |event|
        p [:close, event.code, event.reason]
        @channel.unsubscribe sub
        ws = nil
      end

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
                  puts "Received a NOTIFY on channel #{channel}"
                  puts "from PG backend #{pid}"
                  puts "saying #{payload}"
                  @channel.push true
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
