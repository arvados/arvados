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

  def current_api_base
    Thread.current[:api_url_base]
  end

  def current_default_owner
    # owner_uuid for newly created objects
    ((current_api_client_authorization &&
      current_api_client_authorization.default_owner_uuid) ||
     (current_user && current_user.default_owner_uuid) ||
     (current_user && current_user.uuid) ||
     nil)
  end

  # Where is the client connecting from?
  def current_api_client_ip_address
    Thread.current[:api_client_ip_address]
  end

  # Does the current API client authorization include any of ok_scopes?
  def current_api_client_auth_has_scope(ok_scopes)
    auth_scopes = current_api_client_authorization.andand.scopes || []
    unless auth_scopes.index('all') or (auth_scopes & ok_scopes).any?
      logger.warn "Insufficient auth scope: need #{ok_scopes}, #{current_api_client_authorization.inspect} has #{auth_scopes}"
      return false
    end
    true
  end

  def system_user_uuid
    [Server::Application.config.uuid_prefix,
     User.uuid_prefix,
     '000000000000000'].join('-')
  end

  def system_user
    if not $system_user
      real_current_user = Thread.current[:user]
      Thread.current[:user] = User.new(is_admin: true, is_active: true)
      $system_user = User.where('uuid=?', system_user_uuid).first
      if !$system_user
        $system_user = User.new(uuid: system_user_uuid,
                                is_active: true,
                                is_admin: true,
                                email: 'root',
                                first_name: 'root',
                                last_name: '')
        $system_user.save!
        $system_user.reload
      end
      Thread.current[:user] = real_current_user
    end
    $system_user
  end

  def act_as_system_user
    Thread.current[:user] = system_user
  end
end
