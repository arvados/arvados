# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Perform api_token checking very early in the request process.  We want to do
# this in the Rack stack instead of in ApplicationController because
# websockets needs access to authentication but doesn't use any of the rails
# active dispatch infrastructure.
class ArvadosApiToken

  # Create a new ArvadosApiToken handler
  # +app+  The next layer of the Rack stack.
  def initialize(app = nil, options = nil)
    @app = app.respond_to?(:call) ? app : nil
  end

  def call env
    # First, clean up just in case we have a multithreaded server and thread
    # local variables are still set from a prior request.  Also useful for
    # tests that call this code to set up the environment.
    Thread.current[:supplied_token] = nil
    Thread.current[:api_client_ip_address] = nil
    Thread.current[:api_client_authorization] = nil
    Thread.current[:api_client_uuid] = nil
    Thread.current[:api_client] = nil
    Thread.current[:user] = nil

    request = Rack::Request.new(env)
    params = request.params
    remote_ip = env["action_dispatch.remote_ip"]

    Thread.current[:request_starttime] = Time.now
    user = nil
    api_client = nil
    api_client_auth = nil
    supplied_token =
      params["api_token"] ||
      params["oauth_token"] ||
      env["HTTP_AUTHORIZATION"].andand.match(/(OAuth2|Bearer) ([-,a-zA-Z0-9]+)/).andand[2]
    if supplied_token
      case supplied_token[0..2]
      when 'v2,'
        _, uuid, secret = supplied_token.split(',')
        auth = ApiClientAuthorization.
               includes(:api_client, :user).
               where('uuid=? and api_token=? and (expires_at is null or expires_at > CURRENT_TIMESTAMP)',
                     uuid, secret).
               first
      else
        auth = ApiClientAuthorization.
               includes(:api_client, :user).
               where('api_token=? and (expires_at is null or expires_at > CURRENT_TIMESTAMP)',
                     supplied_token).
               first
      end
      if auth.andand.user
        user = auth.user
        api_client = auth.api_client
      else
        # Token is invalid, or belongs to a non-existent (deleted?) user.
        auth = nil
      end
    end
    Thread.current[:supplied_token] = supplied_token
    Thread.current[:api_client_ip_address] = remote_ip
    Thread.current[:api_client_authorization] = auth
    Thread.current[:api_client_uuid] = api_client.andand.uuid
    Thread.current[:api_client] = api_client
    Thread.current[:user] = user
    if auth
      auth.last_used_at = Time.now
      auth.last_used_by_ip_address = remote_ip.to_s
      auth.save validate: false
    end

    @app.call env if @app
  end
end
