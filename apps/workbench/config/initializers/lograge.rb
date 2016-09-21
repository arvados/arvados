ArvadosWorkbench::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
  config.lograge.custom_options = lambda do |event|
    exceptions = %w(controller action format id)
    params = {current_request_id: Thread.current[:current_request_id]}.
             merge(event.payload[:params].except(*exceptions))
    params_s = Oj.dump(params)
    Thread.current[:current_request_id] = nil # Clear for next request
    if params_s.length > 1000
      { params_truncated: params_s[0..1000] + "[...]" }
    else
      { params: params }
    end
  end
end
