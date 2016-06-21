require 'eventbus'

# See application.yml for details about configuring the websocket service.

Server::Application.configure do
  # Enables websockets if ARVADOS_WEBSOCKETS is defined with any value.  If
  # ARVADOS_WEBSOCKETS=ws-only, server will only accept websocket connections
  # and return an error response for all other requests.
  if ENV['ARVADOS_WEBSOCKETS']
    config.middleware.insert_after ArvadosApiToken, RackSocket, {
      :handler => EventBus,
      :mount => "/websocket",
      :websocket_only => (ENV['ARVADOS_WEBSOCKETS'] == "ws-only")
    }
    Rails.logger.info "Websockets #{ENV['ARVADOS_WEBSOCKETS']}, running at /websocket"
  else
    Rails.logger.info "Websockets disabled"
  end
end
