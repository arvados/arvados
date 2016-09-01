Server::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
  config.lograge.custom_options = lambda do |event|
    exceptions = %w(controller action format id)
    params = event.payload[:params].except(*exceptions)
    params_s = Oj.dump(params)
    if params_s.length > Rails.configuration.max_request_log_params_size
      { params_truncated: params_s[0..Rails.configuration.max_request_log_params_size] + "[...]" }
    else
      { params: params }
    end
  end
end
