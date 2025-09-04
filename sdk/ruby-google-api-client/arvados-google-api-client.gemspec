# -*- encoding: utf-8 -*-
require File.join(File.dirname(__FILE__), 'lib/google/api_client', 'version')

Gem::Specification.new do |s|
  s.name = "arvados-google-api-client"
  s.version = Google::APIClient::VERSION::STRING

  s.required_ruby_version = '>= 3.0.0'
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

  # Dependencies below are pinned to a minor version in cases where
  # the currently support Ruby 3.0 but they have previously dropped
  # support for a Ruby version without incrementing the major version.

  # addressable 2.8.0 dropped Ruby 2.1.
  # addressable 2.8.1 updated metadata to require Ruby >= 2.2.
  # addressable 2.8 is the highest minor version we've tested.
  s.add_runtime_dependency 'addressable', '>= 2.3', '< 2.9'
  # signet 0.20.0 dropped Ruby 3.0.
  s.add_runtime_dependency 'signet', '~> 0.19.0'
  # faraday 2.9.0 dropped Ruby 2.7.
  # faraday 2.13.4 is the highest minor version we've tested.
  s.add_runtime_dependency 'faraday', '~> 2.13.0'
  # faraday-multipart 1.0.1 dropped Ruby 2.4, but 1.0.2 added it back.
  s.add_runtime_dependency 'faraday-multipart', '~> 1.0'
  # faraday-gzip 3.0.0 dropped Ruby 2.
  s.add_runtime_dependency 'faraday-gzip', '~> 3.0'
  # googleauth 1.14.0 dropped Ruby 2.
  # googleauth 1.15 is the highest minor version we've tested.
  s.add_runtime_dependency 'googleauth', '~> 1.15.0'
  # multi_json 2.0 will drop Ruby 3.0.
  # https://github.com/sferik/multi_json/pull/16#issue-3237521157
  # multi_json 1 is the highest major version we've tested.
  s.add_runtime_dependency 'multi_json', '~> 1.15'
  # autoparse was archived 2022-07-27 at 0.3.3.
  s.add_runtime_dependency 'autoparse', '~> 0.3'
  # extlib had no release between 2014 and 2025.
  s.add_runtime_dependency 'extlib', '~> 0.9'
  # launchy 3.0.0 dropped Ruby 2.
  # launchy 3.1.0 stopped testing against Ruby 3.0.
  # launchy 3.0 is the higest minor version we've tested.
  s.add_runtime_dependency 'launchy', '~> 3.0.1'
  # retriable 2 is not API compatible with retriable 1.
  s.add_runtime_dependency 'retriable', '~> 1.4'
  # activesupport 7.2.0 dropped Ruby 3.0.
  s.add_dependency('activesupport', '~> 7.1.3', '>= 7.1.3.4')

  # These are indirect dependencies of the above where we force a resolution
  # that supports all our Rubies.
  # google-cloud-env 2.3.0 dropped Ruby 3.0.
  s.add_runtime_dependency 'google-cloud-env', '~> 2.2.0'
  # public_suffix 6.0.0 dropped Ruby 2.7.
  s.add_runtime_dependency 'public_suffix', '~> 6.0'
  # securerandom 0.4.0 dropped Ruby 3.0 (and 2.6 and 2.7) without
  # mentioning anything in the changelog / release notes.
  s.add_runtime_dependency 'securerandom', '~> 0.3.2'

  # rake 12.3.0 dropped Ruby 1.
  # rake 12 is the highest major version we've tested.
  s.add_development_dependency 'rake', '>= 10', '< 13'
  # yard 0.9.37 (2024) metadata still claims to support Ruby 1.
  s.add_development_dependency 'yard', '>= 0.8', '< 0.10'
  # rspec 3.13.1 (2025) metadata still claims to support Ruby 1.
  # rspec 3.0 is the highest minor version we've tested.
  s.add_development_dependency 'rspec', '~> 3.1'
  # kramdown 2.5.0 dropped Ruby 2.4.
  # kramdown 2.5 is the highest minor version we've tested.
  s.add_development_dependency 'kramdown', '>= 1.5', '< 2.6'
  # simplecov 0.19.0 (2020) dropped Ruby 2.4.
  # simplecov 0.21 is the highest minor version we've tested.
  s.add_development_dependency 'simplecov', '>= 0.9.2', '< 0.22.0'
  # coveralls hasn't had a release since 0.8.23 (2019) whose metadata
  # claims to support Ruby 1.8.7.
  # coveralls 0.8 is the highest minor version we've tested.
  s.add_development_dependency 'coveralls', '>= 0.7.11', '< 0.9'
end
