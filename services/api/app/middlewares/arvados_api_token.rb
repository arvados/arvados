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
    request = Rack::Request.new(env)
    params = request.params
    remote_ip = env["action_dispatch.remote_ip"]

    Thread.current[:request_starttime] = Time.now
    Thread.current[:supplied_token] =
      params["api_token"] ||
      params["oauth_token"] ||
      env["HTTP_AUTHORIZATION"].andand.
        match(/(OAuth2|Bearer) ([-\/a-zA-Z0-9]+)/).andand[2]

    auth = ApiClientAuthorization.
           validate(token: Thread.current[:supplied_token], remote: false)
    Thread.current[:api_client_ip_address] = remote_ip
    Thread.current[:api_client_authorization] = auth
    Thread.current[:api_client_uuid] = auth.andand.api_client.andand.uuid
    Thread.current[:api_client] = auth.andand.api_client
    Thread.current[:user] = auth.andand.user

    if auth
      auth.last_used_at = Time.now
      auth.last_used_by_ip_address = remote_ip.to_s
      auth.save validate: false
    end

    @app.call env if @app
  end
end
