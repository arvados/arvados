require 'rubygems'

# Set up gems listed in the Gemfile.
ENV['BUNDLE_GEMFILE'] ||= File.expand_path('../../Gemfile', __FILE__)

require 'bundler/setup' if File.exists?(ENV['BUNDLE_GEMFILE'])

# Use ORVOS_API_TOKEN environment variable (if set) in console
require 'rails'
module OrvosApiClientConsoleMode
  class Railtie < Rails::Railtie
    console do
      Thread.current[:orvos_api_token] ||= ENV['ORVOS_API_TOKEN']
    end
  end
end
