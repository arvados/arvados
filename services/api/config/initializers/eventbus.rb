require 'eventbus'

Server::Application.configure do
  config.middleware.insert_after ArvadosApiToken, RackSocket, {:handler => EventBus}
end
