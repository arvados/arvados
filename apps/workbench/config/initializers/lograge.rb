ArvadosWorkbench::Application.configure do
  config.lograge.enabled = true
  config.lograge.formatter = Lograge::Formatters::Logstash.new
end
