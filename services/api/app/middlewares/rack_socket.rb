require 'rack'
require 'faye/websocket'
require 'oj'
require 'eventmachine'

class RackSocket

  DEFAULT_ENDPOINT  = '/websocket'

  def die_gracefully_on_signal
    Signal.trap("INT") { EM.stop }
    Signal.trap("TERM") { EM.stop }
  end

  def initialize(app = nil, options = nil)
    @app = app if app.respond_to?(:call)
    @options = [app, options].grep(Hash).first || {}
    @endpoint = @options[:mount] || DEFAULT_ENDPOINT

    # from https://gist.github.com/eatenbyagrue/1338545#file-eventmachine-rb
    if defined?(PhusionPassenger)
      PhusionPassenger.on_event(:starting_worker_process) do |forked|
        # for passenger, we need to avoid orphaned threads
        if forked && EM.reactor_running?
          EM.stop
        end
        Thread.new {
          EM.run
        }
        die_gracefully_on_signal
      end
    else
      # faciliates debugging
      Thread.abort_on_exception = true
      # just spawn a thread and start it up
      Thread.new {
        EM.run
      }
    end

    @channel = EventMachine::Channel.new
    @bgthread = nil
  end

  def call env
    request = Rack::Request.new(env)
    if request.path_info == @endpoint and Faye::WebSocket.websocket?(env)
      ws = Faye::WebSocket.new(env)

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

      # Return async Rack response
      ws.rack_response
    else
      @app.call env
    end
  end

end


