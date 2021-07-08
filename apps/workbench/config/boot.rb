# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'rubygems'

# Set up gems listed in the Gemfile.
ENV['BUNDLE_GEMFILE'] ||= File.expand_path('../../Gemfile', __FILE__)

require 'bundler/setup' if File.exists?(ENV['BUNDLE_GEMFILE'])

# Use ARVADOS_API_TOKEN environment variable (if set) in console
require 'rails'
module ArvadosApiClientConsoleMode
  class Railtie < Rails::Railtie
    console do
      Thread.current[:arvados_api_token] ||= ENV['ARVADOS_API_TOKEN']
    end
  end
end
