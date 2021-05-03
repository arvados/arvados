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

    remote = false
    reader_tokens = nil
    if params["remote"] && request.get? && (
         request.path.start_with?('/arvados/v1/groups') ||
         request.path.start_with?('/arvados/v1/api_client_authorizations/current') ||
         request.path.start_with?('/arvados/v1/users/current'))
      # Request from a remote API server, asking to validate a salted
      # token.
      remote = params["remote"]
    elsif request.get? || params["_method"] == 'GET'
      reader_tokens = params["reader_tokens"]
      if reader_tokens.is_a? String
        reader_tokens = SafeJSON.load(reader_tokens)
      end
    end

    # Set current_user etc. based on the primary session token if a
    # valid one is present. Otherwise, use the first valid token in
    # reader_tokens.
    accepted = false
    auth = nil
    [params["api_token"],
     params["oauth_token"],
     env["HTTP_AUTHORIZATION"].andand.match(/(OAuth2|Bearer) ([!-~]+)/).andand[2],
     *reader_tokens,
    ].each do |supplied|
      next if !supplied
      try_auth = ApiClientAuthorization.
                 validate(token: supplied, remote: remote)
      if try_auth.andand.user
        auth = try_auth
        accepted = supplied
        break
      end
    end

    Thread.current[:api_client_ip_address] = remote_ip
    Thread.current[:api_client_authorization] = auth
    Thread.current[:api_client_uuid] = auth.andand.api_client.andand.uuid
    Thread.current[:api_client] = auth.andand.api_client
    Thread.current[:token] = accepted
    Thread.current[:user] = auth.andand.user

    @app.call env if @app
  end
end
