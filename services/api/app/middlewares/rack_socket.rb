require 'rack'
require 'faye/websocket'
require 'eventmachine'

# A Rack middleware to handle inbound websocket connection requests and hand
# them over to the faye websocket library.
class RackSocket

  DEFAULT_ENDPOINT  = '/websocket'

  # Stop EventMachine on signal, this should give it a chance to to unwind any
  # open connections.
  def die_gracefully_on_signal
    Signal.trap("INT") { EM.stop }
    Signal.trap("TERM") { EM.stop }
  end

  # Create a new RackSocket handler
  # +app+  The next layer of the Rack stack.
  #
  # Accepts options:
  # +:handler+ (Required) A class to handle new connections.  #initialize will
  # call handler.new to create the actual handler instance object.  When a new
  # websocket connection is established, #on_connect on the handler instance
  # object will be called with the new connection.
  #
  # +:mount+ The HTTP request path that will be recognized for websocket
  # connect requests, defaults to '/websocket'.
  #
  # +:websocket_only+  If true, the server will only handle websocket requests,
  # and all other requests will result in an error.  If false, unhandled
  # non-websocket requests will be passed along on to 'app' in the usual Rack
  # way.
  def initialize(app = nil, options = nil)
    @app = app if app.respond_to?(:call)
    @options = [app, options].grep(Hash).first || {}
    @endpoint = @options[:mount] || DEFAULT_ENDPOINT
    @websocket_only = @options[:websocket_only] || false

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

    # Create actual handler instance object from handler class.
    @handler = @options[:handler].new
  end

  # Handle websocket connection request, or pass on to the next middleware
  # supplied in +app+ initialize (unless +:websocket_only+ option is true, in
  # which case return an error response.)
  # +env+ the Rack environment with information about the request.
  def call env
    request = Rack::Request.new(env)
    if request.path_info == @endpoint and Faye::WebSocket.websocket?(env)
      if @handler.overloaded?
        return [503, {"Content-Type" => "text/plain"}, ["Too many connections, try again later."]]
      end

      ws = Faye::WebSocket.new(env, nil, :ping => 30)

      # Notify handler about new connection
      @handler.on_connect ws

      # Return async Rack response
      ws.rack_response
    elsif not @websocket_only
      @app.call env
    else
      [406, {"Content-Type" => "text/plain"}, ["Only websocket connections are permitted on this port."]]
    end
  end

end
