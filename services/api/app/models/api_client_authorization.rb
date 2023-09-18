# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApiClientAuthorization < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include Rails.application.routes.url_helpers
  extend CurrentApiClient
  extend DbCurrentTime

  belongs_to :api_client, optional: true
  belongs_to :user, optional: true
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
    begin
      self.api_token ||= rand(2**256).to_s(36)
    rescue ActiveModel::MissingAttributeError
      # Ignore the case where self.api_token doesn't exist, which happens when
      # the select=[...] is used.
    end
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
    if method == 'GET' and request.path == url_for(controller: 'arvados/v1/api_client_authorizations', action: 'current', only_path: true)
      true
    elsif method == 'HEAD'
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

  def self.check_anonymous_user_token(token:, remote:)
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

    # Usually, the secret is salted
    salted_secret = OpenSSL::HMAC.hexdigest('sha1', Rails.configuration.Users.AnonymousUserToken, remote)

    # The anonymous token could be specified as a full v2 token in the config,
    # but the config loader strips it down to the secret part.
    # The anonymous token content and minimum length is verified in lib/config
    if secret.length >= 0 && (secret == Rails.configuration.Users.AnonymousUserToken || secret == salted_secret)
      return ApiClientAuthorization.new(user: User.find_by_uuid(anonymous_user_uuid),
                                        uuid: Rails.configuration.ClusterID+"-gj3su-anonymouspublic",
                                        api_token: secret,
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

    auth = self.check_anonymous_user_token(token: token, remote: remote)
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
    remote_url = URI::parse("https://#{host}/")
    remote_query = {"remote" => Rails.configuration.ClusterID}
    remote_headers = {"Authorization" => "Bearer #{token}"}

    # First get the current token. This query is not limited by token scopes,
    # and tells us the user's UUID via owner_uuid, so this gives us enough
    # information to load a local user record from the database if one exists.
    remote_token = nil
    begin
      remote_token = SafeJSON.load(
        clnt.get_content(
          remote_url.merge("arvados/v1/api_client_authorizations/current"),
          remote_query, remote_headers,
        ))
      Rails.logger.debug "retrieved remote token #{remote_token.inspect}"
      token_uuid = remote_token['uuid']
      if !token_uuid.match(HasUuid::UUID_REGEX) || token_uuid[0..4] != upstream_cluster_id
        raise "remote cluster #{upstream_cluster_id} returned invalid token uuid #{token_uuid.inspect}"
      end
    rescue HTTPClient::BadResponseError => e
      # CurrentApiToken#call and ApplicationController#render_error will
      # propagate the status code from the #http_status method, so define
      # that here.
      def e.http_status
        self.res.status_code
      end
      raise
    # TODO #20927: Catch network exceptions and assign a 5xx status to them so
    # the client knows they're a temporary problem.
    rescue => e
      Rails.logger.warn "error getting remote token details for #{token.inspect}: #{e}"
      return nil
    end

    # Next, load the token's user record from the database (might be nil).
    remote_user_prefix, remote_user_suffix = remote_token['owner_uuid'].split('-', 2)
    if anonymous_user_uuid.end_with?(remote_user_suffix)
      # Special case: map the remote anonymous user to local anonymous user
      remote_user_uuid = anonymous_user_uuid
    else
      remote_user_uuid = remote_token['owner_uuid']
    end
    user = User.find_by_uuid(remote_user_uuid)

    # Next, try to load the remote user. If this succeeds, we'll use this
    # information to update/create the local database record as necessary.
    # If this fails for any reason, but we successfully loaded a user record
    # from the database, we'll just rely on that information.
    remote_user = nil
    begin
      remote_user = SafeJSON.load(
        clnt.get_content(
          remote_url.merge("arvados/v1/users/current"),
          remote_query, remote_headers,
        ))
    rescue HTTPClient::BadResponseError => e
      # If user is defined, we will use that alone for auth, see below.
      if user.nil?
        # See rationale in the previous BadResponseError rescue.
        def e.http_status
          self.res.status_code
        end
        raise
      end
    # TODO #20927: Catch network exceptions and assign a 5xx status to them so
    # the client knows they're a temporary problem.
    rescue => e
      Rails.logger.warn "getting remote user with token #{token.inspect} failed: #{e}"
    else
      # Check the response is well formed.
      if !remote_user.is_a?(Hash) || !remote_user['uuid'].is_a?(String)
        Rails.logger.warn "malformed remote user=#{remote_user.inspect}"
        remote_user = nil
      # Clusters can only authenticate for their own users.
      elsif remote_user_prefix != upstream_cluster_id
        Rails.logger.warn "remote user rejected: claimed remote user #{remote_user_prefix} but token was issued by #{upstream_cluster_id}"
        remote_user = nil
      # Force our local copy of a remote root to have a static name
      elsif system_user_uuid.end_with?(remote_user_suffix)
        remote_user.update(
          "first_name" => "root",
          "last_name" => "from cluster #{remote_user_prefix}",
        )
      end
    end

    if user.nil? and remote_user.nil?
      Rails.logger.warn "remote token #{token.inspect} rejected: cannot get owner #{remote_user_uuid} from database or remote cluster"
      return nil
    # Invariant:    remote_user_prefix == upstream_cluster_id
    # therefore:    remote_user_prefix != Rails.configuration.ClusterID
    # Add or update user and token in local database so we can
    # validate subsequent requests faster.
    elsif user.nil?
      # Create a new record for this user.
      user = User.new(uuid: remote_user['uuid'],
                      is_active: false,
                      is_admin: false,
                      email: remote_user['email'],
                      owner_uuid: system_user_uuid)
      user.set_initial_username(requested: remote_user['username'])
    end

    # Sync user record if we loaded a remote user.
    act_as_system_user do
      if remote_user
        %w[first_name last_name email prefs].each do |attr|
          user.send(attr+'=', remote_user[attr])
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
            user.update!(is_active: true)
          elsif user.is_active && !remote_user['is_active']
            user.update!(is_active: false)
          end

          if remote_user_prefix == Rails.configuration.Login.LoginCluster and
            user.is_active and
            user.is_admin != remote_user['is_admin']
            # Remote cluster controls our user database, including the
            # admin flag.
            user.update!(is_admin: remote_user['is_admin'])
          end
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
      auth.update!(user: user,
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
