if not Rails.env.production? and ENV['ENABLE_PROFILING']
  require 'rack-mini-profiler'
  require 'flamegraph'
  Rack::MiniProfilerRails.initialize! Rails.application
end
