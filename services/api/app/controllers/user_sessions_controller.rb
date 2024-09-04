# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class UserSessionsController < ApplicationController
  before_action :require_auth_scope, :only => [ :destroy ]

  skip_before_action :set_cors_headers
  skip_before_action :find_object_by_uuid
  skip_before_action :render_404_if_no_object

  respond_to :html

  def login
    return send_error "Legacy code path no longer supported", status: 404
  end

  def logout
    return send_error "Legacy code path no longer supported", status: 404
  end

  # create a new session
  def create
    remote, return_to_url = params[:return_to].split(',', 2)
    if params[:provider] != 'controller' ||
       return_to_url != 'https://controller.api.client.invalid'
      return send_error "Legacy code path no longer supported", status: 404
    end
    if request.headers['Authorization'] != 'Bearer ' + Rails.configuration.SystemRootToken
      return send_error('Invalid authorization header', status: 401)
    end
    if remote == ''
      remote = nil
    elsif remote !~ /^[0-9a-z]{5}$/
      return send_error 'Invalid remote cluster id', status: 400
    end
    # arvados-controller verified the user and is passing auth_info
    # in request params.
    authinfo = SafeJSON.load(params[:auth_info])
    max_expires_at = authinfo["expires_at"]

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

    # prevent ArvadosModel#before_create and _update from throwing
    # "unauthorized":
    Thread.current[:user] = user

    user.save or raise Exception.new(user.errors.messages)

    return send_api_token_to(return_to_url, user, remote, max_expires_at)
  end

  # Omniauth failure callback
  def failure
    flash[:notice] = params[:message]
  end

  def send_api_token_to(callback_url, user, remote=nil, token_expiration=nil)
    # Give the API client a token for making API calls on behalf of
    # the authenticated user

    if Rails.configuration.Login.TokenLifetime > 0
      if token_expiration == nil
        token_expiration = db_current_time + Rails.configuration.Login.TokenLifetime
      else
        token_expiration = [token_expiration, db_current_time + Rails.configuration.Login.TokenLifetime].min
      end
    end

    @api_client_auth = ApiClientAuthorization.
      new(user: user,
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
    redirect_to callback_url, allow_other_host: true
  end

  def cross_origin_forbidden
    send_error 'Forbidden', status: 403
  end
end
