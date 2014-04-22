require 'rack'
require 'faye/websocket'
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

    @handler = @options[:handler].new
  end

  def call env
    request = Rack::Request.new(env)
    if request.path_info == @endpoint and Faye::WebSocket.websocket?(env)
      ws = Faye::WebSocket.new(env)

      @handler.on_connect ws

      # Return async Rack response
      ws.rack_response
    else
      @app.call env
    end
  end

end
