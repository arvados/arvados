# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class UserSessionsController < ApplicationController
  before_action :require_auth_scope, :only => [ :destroy ]

  skip_before_action :set_cors_headers
  skip_before_action :find_object_by_uuid
  skip_before_action :render_404_if_no_object

  respond_to :html

  # create a new session
  def create
    if !Rails.configuration.Login.LoginCluster.empty? and Rails.configuration.Login.LoginCluster != Rails.configuration.ClusterID
      raise "Local login disabled when LoginCluster is set"
    end

    max_expires_at = nil
    if params[:provider] == 'controller'
      if request.headers['Authorization'] != 'Bearer ' + Rails.configuration.SystemRootToken
        return send_error('Invalid authorization header', status: 401)
      end
      # arvados-controller verified the user and is passing auth_info
      # in request params.
      authinfo = SafeJSON.load(params[:auth_info])
      max_expires_at = authinfo["expires_at"]
    else
      return send_error "Legacy code path no longer supported", status: 404
    end

    if !authinfo['user_uuid'].blank?
      user = User.find_by_uuid(authinfo['user_uuid'])
      if !user
        Rails.logger.warn "Nonexistent user_uuid in authinfo #{authinfo.inspect}"
        return redirect_to login_failure_url
      end
    else
      begin
        user = User.register(authinfo)
      rescue => e
        Rails.logger.warn "User.register error #{e}"
        Rails.logger.warn "authinfo was #{authinfo.inspect}"
        return redirect_to login_failure_url
      end
    end

    # For the benefit of functional and integration tests:
    @user = user

    if user.uuid[0..4] != Rails.configuration.ClusterID
      # Actually a remote user
      # Send them to their home cluster's login
      rh = Rails.configuration.RemoteClusters[user.uuid[0..4]]
      remote, return_to_url = params[:return_to].split(',', 2)
      @remotehomeurl = "#{rh.Scheme || "https"}://#{rh.Host}/login?remote=#{Rails.configuration.ClusterID}&return_to=#{return_to_url}"
      render
      return
    end

    # prevent ArvadosModel#before_create and _update from throwing
    # "unauthorized":
    Thread.current[:user] = user

    user.save or raise Exception.new(user.errors.messages)

    # Give the authenticated user a cookie for direct API access
    session[:user_id] = user.id
    session[:api_client_uuid] = nil
    session[:api_client_trusted] = true # full permission to see user's secrets

    @redirect_to = root_path
    if params.has_key?(:return_to)
      # return_to param's format is 'remote,return_to_url'. This comes from login()
      # encoding the remote=zbbbb parameter passed by a client asking for a salted
      # token.
      remote, return_to_url = params[:return_to].split(',', 2)
      if remote !~ /^[0-9a-z]{5}$/ && remote != ""
        return send_error 'Invalid remote cluster id', status: 400
      end
      remote = nil if remote == ''
      return send_api_token_to(return_to_url, user, remote, max_expires_at)
    end
    redirect_to @redirect_to
  end

  # Omniauth failure callback
  def failure
    flash[:notice] = params[:message]
  end

  # logout - this gets intercepted by controller, so this is probably
  # mostly dead code at this point.
  def logout
    session[:user_id] = nil

    flash[:notice] = 'You have logged off'
    return_to = params[:return_to] || root_url
    redirect_to return_to
  end

  # login.  Redirect to LoginCluster.
  def login
    if params[:remote] !~ /^[0-9a-z]{5}$/ && !params[:remote].nil?
      return send_error 'Invalid remote cluster id', status: 400
    end
    if current_user and params[:return_to]
      # Already logged in; just need to send a token to the requesting
      # API client.
      #
      # FIXME: if current_user has never authorized this app before,
      # ask for confirmation here!

      return send_api_token_to(params[:return_to], current_user, params[:remote])
    end
    p = []
    p << "auth_provider=#{CGI.escape(params[:auth_provider])}" if params[:auth_provider]

    if !Rails.configuration.Login.LoginCluster.empty? and Rails.configuration.Login.LoginCluster != Rails.configuration.ClusterID
      host = ApiClientAuthorization.remote_host(uuid_prefix: Rails.configuration.Login.LoginCluster)
      if not host
        raise "LoginCluster #{Rails.configuration.Login.LoginCluster} missing from RemoteClusters"
      end
      scheme = "https"
      cluster = Rails.configuration.RemoteClusters[Rails.configuration.Login.LoginCluster]
      if cluster and cluster['Scheme'] and !cluster['Scheme'].empty?
        scheme = cluster['Scheme']
      end
      login_cluster = "#{scheme}://#{host}"
      p << "remote=#{CGI.escape(params[:remote])}" if params[:remote]
      p << "return_to=#{CGI.escape(params[:return_to])}" if params[:return_to]
      redirect_to "#{login_cluster}/login?#{p.join('&')}"
    else
      return send_error "Legacy code path no longer supported", status: 404
    end
  end

  def send_api_token_to(callback_url, user, remote=nil, token_expiration=nil)
    # Give the API client a token for making API calls on behalf of
    # the authenticated user

    # Stub: automatically register all new API clients
    api_client_url_prefix = callback_url.match(%r{^.*?://[^/]+})[0] + '/'
    act_as_system_user do
      @api_client = ApiClient.
        find_or_create_by(url_prefix: api_client_url_prefix)
    end
    if Rails.configuration.Login.TokenLifetime > 0
      if token_expiration == nil
        token_expiration = db_current_time + Rails.configuration.Login.TokenLifetime
      else
        token_expiration = [token_expiration, db_current_time + Rails.configuration.Login.TokenLifetime].min
      end
    end

    @api_client_auth = ApiClientAuthorization.
      new(user: user,
          api_client: @api_client,
          created_by_ip_address: remote_ip,
          expires_at: token_expiration,
          scopes: ["all"])
    @api_client_auth.save!

    if callback_url.index('?')
      callback_url += '&'
    else
      callback_url += '?'
    end
    if remote.nil?
      token = @api_client_auth.token
    else
      token = @api_client_auth.salted_token(remote: remote)
    end
    callback_url += 'api_token=' + token
    redirect_to callback_url
  end

  def cross_origin_forbidden
    send_error 'Forbidden', status: 403
  end
end
