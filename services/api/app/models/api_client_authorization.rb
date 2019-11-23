# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApiClientAuthorization < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  extend CurrentApiClient

  belongs_to :api_client
  belongs_to :user
  after_initialize :assign_random_api_token
  serialize :scopes, Array

  api_accessible :user, extend: :common do |t|
    t.add :owner_uuid
    t.add :user_id
    t.add :api_client_id
    # NB the "api_token" db column is a misnomer in that it's only the
    # "secret" part of a token: a v1 token is just the secret, but a
    # v2 token is "v2/uuid/secret".
    t.add :api_token
    t.add :created_by_ip_address
    t.add :default_owner_uuid
    t.add :expires_at
    t.add :last_used_at
    t.add :last_used_by_ip_address
    t.add :scopes
  end

  UNLOGGED_CHANGES = ['last_used_at', 'last_used_by_ip_address', 'updated_at']

  def assign_random_api_token
    self.api_token ||= rand(2**256).to_s(36)
  end

  def owner_uuid
    self.user.andand.uuid
  end
  def owner_uuid_was
    self.user_id_changed? ? User.where(id: self.user_id_was).first.andand.uuid : self.user.andand.uuid
  end
  def owner_uuid_changed?
    self.user_id_changed?
  end

  def modified_by_client_uuid
    nil
  end
  def modified_by_client_uuid=(x) end

  def modified_by_user_uuid
    nil
  end
  def modified_by_user_uuid=(x) end

  def modified_at
    nil
  end
  def modified_at=(x) end

  def scopes_allow?(req_s)
    scopes.each do |scope|
      return true if (scope == 'all') or (scope == req_s) or
        ((scope.end_with? '/') and (req_s.start_with? scope))
    end
    false
  end

  def scopes_allow_request?(request)
    method = request.request_method
    if method == 'HEAD'
      (scopes_allow?(['HEAD', request.path].join(' ')) ||
       scopes_allow?(['GET', request.path].join(' ')))
    else
      scopes_allow?([method, request.path].join(' '))
    end
  end

  def logged_attributes
    super.except 'api_token'
  end

  def self.default_orders
    ["#{table_name}.id desc"]
  end

  def self.remote_host(uuid_prefix:)
    (Rails.configuration.RemoteClusters[uuid_prefix].andand["Host"]) ||
      (Rails.configuration.RemoteClusters["*"]["Proxy"] &&
       uuid_prefix+".arvadosapi.com")
  end

  def self.make_http_client(uuid_prefix:)
    clnt = HTTPClient.new

    if uuid_prefix && (Rails.configuration.RemoteClusters[uuid_prefix].andand.Insecure ||
                       Rails.configuration.RemoteClusters['*'].andand.Insecure)
      clnt.ssl_config.verify_mode = OpenSSL::SSL::VERIFY_NONE
    else
      # Use system CA certificates
      ["/etc/ssl/certs/ca-certificates.crt",
       "/etc/pki/tls/certs/ca-bundle.crt"]
        .select { |ca_path| File.readable?(ca_path) }
        .each { |ca_path| clnt.ssl_config.add_trust_ca(ca_path) }
    end
    clnt
  end

  def self.check_system_root_token token
    if token == Rails.configuration.SystemRootToken
      return ApiClientAuthorization.new(user: User.find_by_uuid(system_user_uuid),
                                        api_token: token,
                                        api_client: ApiClient.new(is_trusted: true, url_prefix: ""))
    else
      return nil
    end
  end

  def self.validate(token:, remote: nil)
    return nil if token.nil? or token.empty?
    remote ||= Rails.configuration.ClusterID

    auth = self.check_system_root_token(token)
    if !auth.nil?
      return auth
    end

    case token[0..2]
    when 'v2/'
      _, token_uuid, secret, optional = token.split('/')
      unless token_uuid.andand.length == 27 && secret.andand.length.andand > 0
        return nil
      end

      if !optional.nil?
        # if "optional" is a container uuid, check that it
        # matches expections.
        c = Container.where(uuid: optional).first
        if !c.nil?
          if !c.auth_uuid.nil? and c.auth_uuid != token_uuid
            # token doesn't match the container's token
            return nil
          end
          if !c.runtime_token.nil? and "v2/#{token_uuid}/#{secret}" != c.runtime_token
            # token doesn't match the container's token
            return nil
          end
          if ![Container::Locked, Container::Running].include?(c.state)
            # container isn't locked or running, token shouldn't be used
            return nil
          end
        end
      end

      # fast path: look up the token in the local database
      auth = ApiClientAuthorization.
             includes(:user, :api_client).
             where('uuid=? and (expires_at is null or expires_at > CURRENT_TIMESTAMP)', token_uuid).
             first
      if auth && auth.user &&
         (secret == auth.api_token ||
          secret == OpenSSL::HMAC.hexdigest('sha1', auth.api_token, remote))
        # found it
        return auth
      end

      token_uuid_prefix = token_uuid[0..4]
      if token_uuid_prefix == Rails.configuration.ClusterID
        # Token is supposedly issued by local cluster, but if the
        # token were valid, we would have been found in the database
        # in the above query.
        return nil
      elsif token_uuid_prefix.length != 5
        # malformed
        return nil
      end

      # Invariant: token_uuid_prefix != Rails.configuration.ClusterID
      #
      # In other words the remaing code in this method below is the
      # case that determines whether to accept a token that was issued
      # by a remote cluster when the token absent or expired in our
      # database.  To begin, we need to ask the cluster that issued
      # the token to [re]validate it.
      clnt = ApiClientAuthorization.make_http_client(uuid_prefix: token_uuid_prefix)

      host = remote_host(uuid_prefix: token_uuid_prefix)
      if !host
        Rails.logger.warn "remote authentication rejected: no host for #{token_uuid_prefix.inspect}"
        return nil
      end

      begin
        remote_user = SafeJSON.load(
          clnt.get_content('https://' + host + '/arvados/v1/users/current',
                           {'remote' => Rails.configuration.ClusterID},
                           {'Authorization' => 'Bearer ' + token}))
      rescue => e
        Rails.logger.warn "remote authentication with token #{token.inspect} failed: #{e}"
        return nil
      end

      # Check the response is well formed.
      if !remote_user.is_a?(Hash) || !remote_user['uuid'].is_a?(String)
        Rails.logger.warn "remote authentication rejected: remote_user=#{remote_user.inspect}"
        return nil
      end

      remote_user_prefix = remote_user['uuid'][0..4]

      # Clusters can only authenticate for their own users.
      if remote_user_prefix != token_uuid_prefix
        Rails.logger.warn "remote authentication rejected: claimed remote user #{remote_user_prefix} but token was issued by #{token_uuid_prefix}"
        return nil
      end

      # Invariant:    remote_user_prefix == token_uuid_prefix
      # therefore:    remote_user_prefix != Rails.configuration.ClusterID

      # Add or update user and token in local database so we can
      # validate subsequent requests faster.

      user = User.find_by_uuid(remote_user['uuid'])

      if !user
        # Create a new record for this user.
        user = User.new(uuid: remote_user['uuid'],
                        is_active: false,
                        is_admin: false,
                        email: remote_user['email'],
                        owner_uuid: system_user_uuid)
        user.set_initial_username(requested: remote_user['username'])
      end

      # Sync user record.
      if remote_user_prefix == Rails.configuration.Login.LoginCluster
        # Remote cluster controls our user database, copy both
        # 'is_active' and 'is_admin'
        user.is_active = remote_user['is_active']
        user.is_admin = remote_user['is_admin']
      else
        if Rails.configuration.Users.NewUsersAreActive ||
           Rails.configuration.RemoteClusters[remote_user_prefix].andand["ActivateUsers"]
          # Default policy is to activate users, so match activate
          # with the remote record.
          user.is_active = remote_user['is_active']
        elsif !remote_user['is_active']
          # Deactivate user if the remote is inactive, otherwise don't
          # change 'is_active'.
          user.is_active = false
        end
      end

      %w[first_name last_name email prefs].each do |attr|
        user.send(attr+'=', remote_user[attr])
      end

      act_as_system_user do
        user.save!

        # We will accept this token (and avoid reloading the user
        # record) for 'RemoteTokenRefresh' (default 5 minutes).
        # Possible todo:
        # Request the actual api_client_auth record from the remote
        # server in case it wants the token to expire sooner.
        auth = ApiClientAuthorization.find_or_create_by(uuid: token_uuid) do |auth|
          auth.user = user
          auth.api_client_id = 0
        end
        auth.update_attributes!(user: user,
                                api_token: secret,
                                api_client_id: 0,
                                expires_at: Time.now + Rails.configuration.Login.RemoteTokenRefresh)
      end
      return auth
    else
      # token is not a 'v2' token
      auth = ApiClientAuthorization.
               includes(:user, :api_client).
               where('api_token=? and (expires_at is null or expires_at > CURRENT_TIMESTAMP)', token).
               first
      if auth && auth.user
        return auth
      end
    end

    return nil
  end

  def token
    v2token
  end

  def v1token
    api_token
  end

  def v2token
    'v2/' + uuid + '/' + api_token
  end

  def salted_token(remote:)
    if remote.nil?
      token
    end
    'v2/' + uuid + '/' + OpenSSL::HMAC.hexdigest('sha1', api_token, remote)
  end

  protected

  def permission_to_create
    current_user.andand.is_admin or (current_user.andand.id == self.user_id)
  end

  def permission_to_update
    permission_to_create && !uuid_changed? &&
      (current_user.andand.is_admin || !user_id_changed?)
  end

  def log_update
    super unless (changed - UNLOGGED_CHANGES).empty?
  end
end
