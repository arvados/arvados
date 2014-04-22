require 'eventbus'

Server::Application.configure do
  if ENV['ARVADOS_WEBSOCKETS']
    config.middleware.insert_after ArvadosApiToken, RackSocket, {:handler => EventBus, :websocket_only => true }
  end
end
