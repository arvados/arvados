# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'safe_json'

Server::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
  config.lograge.custom_options = lambda do |event|
    payload = {
      request_id: event.payload[:request_id],
      client_ipaddr: event.payload[:client_ipaddr],
      client_auth: event.payload[:client_auth],
    }
    exceptions = %w(controller action format id)
    params = event.payload[:params].except(*exceptions)
    params_s = SafeJSON.dump(params)
    if params_s.length > Rails.configuration.max_request_log_params_size
      payload[:params_truncated] = params_s[0..Rails.configuration.max_request_log_params_size] + "[...]"
    else
      payload[:params] = params
    end
    payload
  end
end
