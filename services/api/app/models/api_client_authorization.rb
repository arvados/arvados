# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApiClientAuthorization < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  extend CurrentApiClient
  extend DbCurrentTime

  belongs_to :api_client
  belongs_to :user
  after_initialize :assign_random_api_token
  serialize :scopes, Array

  before_validation :clamp_token_expiration

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

  def self.check_anonymous_user_token token
    case token[0..2]
    when 'v2/'
      _, token_uuid, secret, optional = token.split('/')
      unless token_uuid.andand.length == 27 && secret.andand.length.andand > 0 &&
             token_uuid == Rails.configuration.ClusterID+"-gj3su-anonymouspublic"
        # invalid v2 token, or v2 token for another user
        return nil
      end
    else
      # v1 token
      secret = token
    end

    # The anonymous token content and minimum length is verified in lib/config
    if secret.length >= 0 && secret == Rails.configuration.Users.AnonymousUserToken
      return ApiClientAuthorization.new(user: User.find_by_uuid(anonymous_user_uuid),
                                        uuid: Rails.configuration.ClusterID+"-gj3su-anonymouspublic",
                                        api_token: token,
                                        api_client: anonymous_user_token_api_client,
                                        scopes: ['GET /'])
    else
      return nil
    end
  end

  def self.check_system_root_token token
    if token == Rails.configuration.SystemRootToken
      return ApiClientAuthorization.new(user: User.find_by_uuid(system_user_uuid),
                                        uuid: Rails.configuration.ClusterID+"-gj3su-000000000000000",
                                        api_token: token,
                                        api_client: system_root_token_api_client)
    else
      return nil
    end
  end

  def self.validate(token:, remote: nil)
    return nil if token.nil? or token.empty?
    remote ||= Rails.configuration.ClusterID

    auth = self.check_anonymous_user_token(token)
    if !auth.nil?
      return auth
    end

    auth = self.check_system_root_token(token)
    if !auth.nil?
      return auth
    end

    token_uuid = ''
    secret = token
    stored_secret = nil         # ...if different from secret
    optional = nil

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
        if token_uuid[0..4] != Rails.configuration.ClusterID
          Rails.logger.debug "found cached remote token #{token_uuid} with secret #{secret} in local db"
        end
        return auth
      end

      upstream_cluster_id = token_uuid[0..4]
      if upstream_cluster_id == Rails.configuration.ClusterID
        # Token is supposedly issued by local cluster, but if the
        # token were valid, we would have been found in the database
        # in the above query.
        return nil
      elsif upstream_cluster_id.length != 5
        # malformed
        return nil
      end

    else
      # token is not a 'v2' token. It could be just the secret part
      # ("v1 token") -- or it could be an OpenIDConnect access token,
      # in which case either (a) the controller will have inserted a
      # row with api_token = hmac(systemroottoken,oidctoken) before
      # forwarding it, or (b) we'll have done that ourselves, or (c)
      # we'll need to ask LoginCluster to validate it for us below,
      # and then insert a local row for a faster lookup next time.
      hmac = OpenSSL::HMAC.hexdigest('sha256', Rails.configuration.SystemRootToken, token)
      auth = ApiClientAuthorization.
               includes(:user, :api_client).
               where('api_token in (?, ?) and (expires_at is null or expires_at > CURRENT_TIMESTAMP)', token, hmac).
               first
      if auth && auth.user
        return auth
      elsif !Rails.configuration.Login.LoginCluster.blank? && Rails.configuration.Login.LoginCluster != Rails.configuration.ClusterID
        # An unrecognized non-v2 token might be an OIDC Access Token
        # that can be verified by our login cluster in the code
        # below. If so, we'll stuff the database with hmac instead of
        # the real OIDC token.
        upstream_cluster_id = Rails.configuration.Login.LoginCluster
        stored_secret = hmac
      else
        return nil
      end
    end

    # Invariant: upstream_cluster_id != Rails.configuration.ClusterID
    #
    # In other words the remaining code in this method decides
    # whether to accept a token that was issued by a remote cluster
    # when the token is absent or expired in our database.  To
    # begin, we need to ask the cluster that issued the token to
    # [re]validate it.
    clnt = ApiClientAuthorization.make_http_client(uuid_prefix: upstream_cluster_id)

    host = remote_host(uuid_prefix: upstream_cluster_id)
    if !host
      Rails.logger.warn "remote authentication rejected: no host for #{upstream_cluster_id.inspect}"
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

    # Get token scope, and make sure we use the same UUID as the
    # remote when caching the token.
    remote_token = nil
    begin
      remote_token = SafeJSON.load(
        clnt.get_content('https://' + host + '/arvados/v1/api_client_authorizations/current',
                         {'remote' => Rails.configuration.ClusterID},
                         {'Authorization' => 'Bearer ' + token}))
      Rails.logger.debug "retrieved remote token #{remote_token.inspect}"
      token_uuid = remote_token['uuid']
      if !token_uuid.match(HasUuid::UUID_REGEX) || token_uuid[0..4] != upstream_cluster_id
        raise "remote cluster #{upstream_cluster_id} returned invalid token uuid #{token_uuid.inspect}"
      end
    rescue HTTPClient::BadResponseError => e
      if e.res.status != 401
        raise
      end
      rev = SafeJSON.load(clnt.get_content('https://' + host + '/discovery/v1/apis/arvados/v1/rest'))['revision']
      if rev >= '20010101' && rev < '20210503'
        Rails.logger.warn "remote cluster #{upstream_cluster_id} at #{host} with api rev #{rev} does not provide token expiry and scopes; using scopes=['all']"
      else
        # remote server is new enough that it should have accepted
        # this request if the token was valid
        raise
      end
    rescue => e
      Rails.logger.warn "error getting remote token details for #{token.inspect}: #{e}"
      return nil
    end

    # Clusters can only authenticate for their own users.
    if remote_user_prefix != upstream_cluster_id
      Rails.logger.warn "remote authentication rejected: claimed remote user #{remote_user_prefix} but token was issued by #{upstream_cluster_id}"
      return nil
    end

    # Invariant:    remote_user_prefix == upstream_cluster_id
    # therefore:    remote_user_prefix != Rails.configuration.ClusterID

    # Add or update user and token in local database so we can
    # validate subsequent requests faster.

    if remote_user['uuid'][-22..-1] == '-tpzed-anonymouspublic'
      # Special case: map the remote anonymous user to local anonymous user
      remote_user['uuid'] = anonymous_user_uuid
    end

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
    act_as_system_user do
      %w[first_name last_name email prefs].each do |attr|
        user.send(attr+'=', remote_user[attr])
      end

      if remote_user['uuid'][-22..-1] == '-tpzed-000000000000000'
        user.first_name = "root"
        user.last_name = "from cluster #{remote_user_prefix}"
      end

      begin
        user.save!
      rescue ActiveRecord::RecordInvalid, ActiveRecord::RecordNotUnique
        Rails.logger.debug("remote user #{remote_user['uuid']} already exists, retrying...")
        # Some other request won the race: retry fetching the user record.
        user = User.find_by_uuid(remote_user['uuid'])
        if !user
          Rails.logger.warn("cannot find or create remote user #{remote_user['uuid']}")
          return nil
        end
      end

      if user.is_invited && !remote_user['is_invited']
        # Remote user is not "invited" state, they should be unsetup, which
        # also makes them inactive.
        user.unsetup
      else
        if !user.is_invited && remote_user['is_invited'] and
          (remote_user_prefix == Rails.configuration.Login.LoginCluster or
           Rails.configuration.Users.AutoSetupNewUsers or
           Rails.configuration.Users.NewUsersAreActive or
           Rails.configuration.RemoteClusters[remote_user_prefix].andand["ActivateUsers"])
          user.setup
        end

        if !user.is_active && remote_user['is_active'] && user.is_invited and
          (remote_user_prefix == Rails.configuration.Login.LoginCluster or
           Rails.configuration.Users.NewUsersAreActive or
           Rails.configuration.RemoteClusters[remote_user_prefix].andand["ActivateUsers"])
          user.update_attributes!(is_active: true)
        elsif user.is_active && !remote_user['is_active']
          user.update_attributes!(is_active: false)
        end

        if remote_user_prefix == Rails.configuration.Login.LoginCluster and
          user.is_active and
          user.is_admin != remote_user['is_admin']
          # Remote cluster controls our user database, including the
          # admin flag.
          user.update_attributes!(is_admin: remote_user['is_admin'])
        end
      end

      # If stored_secret is set, we save stored_secret in the database
      # but return the real secret to the caller. This way, if we end
      # up returning the auth record to the client, they see the same
      # secret they supplied, instead of the HMAC we saved in the
      # database.
      stored_secret = stored_secret || secret

      # We will accept this token (and avoid reloading the user
      # record) for 'RemoteTokenRefresh' (default 5 minutes).
      exp = [db_current_time + Rails.configuration.Login.RemoteTokenRefresh,
             remote_token.andand['expires_at']].compact.min
      scopes = remote_token.andand['scopes'] || ['all']
      begin
        retries ||= 0
        auth = ApiClientAuthorization.find_or_create_by(uuid: token_uuid) do |auth|
          auth.user = user
          auth.api_token = stored_secret
          auth.api_client_id = 0
          auth.scopes = scopes
          auth.expires_at = exp
        end
      rescue ActiveRecord::RecordNotUnique
        Rails.logger.debug("cached remote token #{token_uuid} already exists, retrying...")
        # Some other request won the race: retry just once before erroring out
        if (retries += 1) <= 1
          retry
        else
          Rails.logger.warn("cannot find or create cached remote token #{token_uuid}")
          return nil
        end
      end
      auth.update_attributes!(user: user,
                              api_token: stored_secret,
                              api_client_id: 0,
                              scopes: scopes,
                              expires_at: exp)
      Rails.logger.debug "cached remote token #{token_uuid} with secret #{stored_secret} and scopes #{scopes} in local db"
      auth.api_token = secret
      return auth
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

  def clamp_token_expiration
    if Rails.configuration.API.MaxTokenLifetime > 0
      max_token_expiration = db_current_time + Rails.configuration.API.MaxTokenLifetime
      if (self.new_record? || self.expires_at_changed?) && (self.expires_at.nil? || (self.expires_at > max_token_expiration && !current_user.andand.is_admin))
        self.expires_at = max_token_expiration
      end
    end
  end

  def permission_to_create
    current_user.andand.is_admin or (current_user.andand.id == self.user_id)
  end

  def permission_to_update
    permission_to_create && !uuid_changed? &&
      (current_user.andand.is_admin || !user_id_changed?)
  end

  def log_update
    super unless (saved_changes.keys - UNLOGGED_CHANGES).empty?
  end
end
