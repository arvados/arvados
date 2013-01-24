module CurrentApiClient
  def current_user
    Thread.current[:user]
  end

  def current_api_client
    Thread.current[:api_client]
  end

  # Where is the client connecting from?
  def current_api_client_ip_address
    Thread.current[:api_client_ip_address]
  end

  # Is the current client permitted to perform ALL actions on behalf
  # of the authenticated user?
  def current_api_client_trusted
    Thread.current[:api_client_trusted]
  end
end
