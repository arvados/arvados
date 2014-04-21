Server::Application.configure do
  config.middleware.insert_before ActionDispatch::Static, RackSocket
end
