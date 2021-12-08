# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ArvadosWorkbench::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
  config.lograge.custom_options = lambda do |event|
    payload = {
      ClusterID: Rails.configuration.ClusterID,
      request_id: event.payload[:request_id],
    }
    # Also log params (minus the pseudo-params added by Rails). But if
    # params is huge, don't log the whole thing, just hope we get the
    # most useful bits in truncate(json(params)).
    exceptions = %w(controller action format id)
    params = event.payload[:params].except(*exceptions)
    params_s = Oj.dump(params)
    if params_s.length > 1000
      payload[:params_truncated] = params_s[0..1000] + "[...]"
    else
      payload[:params] = params
    end
    payload
  end
end
