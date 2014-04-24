require 'eventbus'

Server::Application.configure do
  if ENV['ARVADOS_WEBSOCKETS'] == '1'
    config.middleware.insert_after ArvadosApiToken, RackSocket, {:handler => EventBus, :websocket_only => true }
  end
end
