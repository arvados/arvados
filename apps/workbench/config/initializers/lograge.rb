ArvadosWorkbench::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
  config.lograge.custom_options = lambda do |event|
    exceptions = %w(controller action format id)
    params = event.payload[:params].except(*exceptions)
    params_s = Oj.dump(params)
    if params_s.length > 1000
      { params_truncated: params_s[0..1000] + "[...]" }
    else
      { params: params }
    end
  end
end
