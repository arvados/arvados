Server::Application.configure do
  config.middleware.insert_before ActionDispatch::Static, ArvadosApiToken
  config.middleware.delete ActionDispatch::RemoteIp
  config.middleware.insert_before ArvadosApiToken, ActionDispatch::RemoteIp
end
