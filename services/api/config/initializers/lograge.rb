# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'safe_json'

Server::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
  config.lograge.custom_options = lambda do |event|
    payload = {
      ClusterID: Rails.configuration.ClusterID,
      request_id: event.payload[:request_id],
      client_ipaddr: event.payload[:client_ipaddr],
      client_auth: event.payload[:client_auth],
    }

    # Lograge adds exceptions not being rescued to event.payload, but we're
    # catching all errors on ApplicationController so we look for backtraces
    # elsewhere.
    if !Thread.current[:backtrace].nil?
      payload.merge!(
        {
          exception: Thread.current[:exception],
          exception_backtrace: Thread.current[:backtrace],
        }
      )
      Thread.current[:exception] = nil
      Thread.current[:backtrace] = nil
    end

    exceptions = %w(controller action format id)
    params = event.payload[:params].except(*exceptions)

    # Omit secret_mounts field if supplied in create/update request
    # body.
    [
      ['container', 'secret_mounts'],
      ['container_request', 'secret_mounts'],
    ].each do |resource, field|
      if params[resource].is_a? Hash
        params[resource] = params[resource].except(field)
      end
    end

    # Redact new_user_token param in /arvados/v1/users/merge
    # request. Log the auth UUID instead, if the token exists.
    if params['new_user_token'].is_a? String
      params['new_user_token_uuid'] =
        ApiClientAuthorization.
          where('api_token = ?', params['new_user_token']).
          first.andand.uuid
      params['new_user_token'] = '[...]'
    end

    params_s = SafeJSON.dump(params)
    if params_s.length > Rails.configuration.SystemLogs["MaxRequestLogParamsSize"]
      payload[:params_truncated] = params_s[0..Rails.configuration.SystemLogs["MaxRequestLogParamsSize"]] + "[...]"
    else
      payload[:params] = params
    end
    payload
  end
end
