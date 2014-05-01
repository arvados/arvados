Server::Application.configure do
  config.middleware.delete ActionDispatch::RemoteIp
  config.middleware.insert 0, ActionDispatch::RemoteIp
  config.middleware.insert 1, ArvadosApiToken
end
