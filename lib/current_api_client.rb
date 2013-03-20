module CurrentApiClient
  def current_user
    Thread.current[:user]
  end

  def current_api_client
    Thread.current[:api_client]
  end

  def current_api_client_authorization
    Thread.current[:api_client_authorization]
  end

  def current_default_owner
    # owner uuid for newly created objects
    ((current_api_client_authorization &&
      current_api_client_authorization.default_owner) ||
     (current_user && current_user.default_owner) ||
     (current_user && current_user.uuid) ||
     nil)
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
