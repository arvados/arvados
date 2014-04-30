require 'eventbus'

Server::Application.configure do
  # Enables websockets if ARVADOS_WEBSOCKETS is defined with any value.  If
  # ARVADOS_WEBSOCKETS=ws-only, server will only accept websocket connections
  # and return an error response for all other requests.
  if ENV['ARVADOS_WEBSOCKETS']
    config.middleware.insert_after ArvadosApiToken, RackSocket, {
      :handler => EventBus,
      :mount => "/websockets",
      :websocket_only => (ENV['ARVADOS_WEBSOCKETS'] == "ws-only")
    }
  end

  # Define websocket_address configuration option, can be overridden in config files.
  # See application.yml.example for details.
  config.websocket_address = nil
end
