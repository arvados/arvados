# -*- encoding: utf-8 -*-
require File.join(File.dirname(__FILE__), 'lib/google/api_client', 'version')

Gem::Specification.new do |s|
  s.name = "arvados-google-api-client"
  s.version = Google::APIClient::VERSION::STRING

  s.required_ruby_version = '>= 2.7.0'
  s.required_rubygems_version = ">= 1.3.5"
  s.require_paths = ["lib"]
  s.authors = ["Bob Aman", "Steven Bazyl"]
  s.license = "Apache-2.0"
  s.description = "Fork of google-api-client used by Ruby-based Arvados components."
  s.email = "dev@arvados.org"
  s.extra_rdoc_files = ["README.md"]
  s.files = %w(arvados-google-api-client.gemspec Rakefile LICENSE CHANGELOG.md README.md Gemfile)
  s.files += Dir.glob("lib/**/*.rb")
  s.files += Dir.glob("lib/cacerts.pem")
  s.files += Dir.glob("spec/**/*.{rb,opts}")
  s.files += Dir.glob("vendor/**/*.rb")
  s.files += Dir.glob("tasks/**/*")
  s.files += Dir.glob("website/**/*")
  s.homepage = "https://github.com/arvados/arvados/tree/main/sdk/ruby-google-api-client"
  s.rdoc_options = ["--main", "README.md"]
  s.summary = "Fork of google-api-client used by Ruby-based Arvados components."

  s.add_runtime_dependency 'addressable', '~> 2.3'
  s.add_runtime_dependency 'signet', '~> 0.16.0'
  # faraday requires Ruby 3.0 starting with 2.9.0. If you install this gem
  # on Ruby 2.7, the dependency resolver asks you to resolve the conflict
  # manually. Instead of teaching all our tooling to do that, we prefer to
  # require the latest version that supports Ruby 2.7 here. This requirement
  # can be relaxed to '~> 2.0' when we drop support for Ruby 2.7.
  s.add_runtime_dependency 'faraday', '~> 2.8.0'
  s.add_runtime_dependency 'faraday-multipart', '~> 1.0'
  s.add_runtime_dependency 'faraday-gzip', '~> 2.0'
  s.add_runtime_dependency 'googleauth', '~> 1.0'
  s.add_runtime_dependency 'multi_json', '~> 1.10'
  s.add_runtime_dependency 'autoparse', '~> 0.3'
  s.add_runtime_dependency 'extlib', '~> 0.9'
  s.add_runtime_dependency 'launchy', '~> 2.4'
  s.add_runtime_dependency 'retriable', '~> 1.4'
  s.add_runtime_dependency 'activesupport', '>= 3.2', '< 8.0'

  s.add_development_dependency 'rake', '~> 10.0'
  s.add_development_dependency 'yard', '~> 0.8'
  s.add_development_dependency 'rspec', '~> 3.1'
  s.add_development_dependency 'kramdown', '~> 1.5'
  s.add_development_dependency 'simplecov', '~> 0.9.2'
  s.add_development_dependency 'coveralls', '~> 0.7.11'
end
