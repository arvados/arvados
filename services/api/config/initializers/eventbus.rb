require 'eventbus'

Server::Application.configure do
  config.middleware.insert_before ActionDispatch::Static, RackSocket, {:handler => EventBus}
end
