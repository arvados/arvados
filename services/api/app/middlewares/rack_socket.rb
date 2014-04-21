
  require 'faye/websocket'

  class RackSocket

    DEFAULT_ENDPOINT  = '/websocket'

    def initialize(app = nil, options = nil)
      @app = app if app.respond_to?(:call)
      @options = [app, options].grep(Hash).first || {}
      @endpoint = @options[:mount] || DEFAULT_ENDPOINT
    end

    def call env
      request = Rack::Request.new(env)
      if request.path_info == @endpoint and Faye::WebSocket.websocket?(env)
        ws = Faye::WebSocket.new(env)

        ws.on :message do |event|
          puts "got #{event.data}"
          ws.send(event.data)
        end

        ws.on :close do |event|
          p [:close, event.code, event.reason]
          ws = nil
        end

        # Return async Rack response
        ws.rack_response
      else
        @app.call env
      end
    end

  end


