# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require_relative 'boot'

require "rails"
# Pick only the frameworks we need:
require "active_model/railtie"
require "active_job/railtie"
require "active_record/railtie"
# Skip ActiveStorage (new in Rails 5.1)
# require "active_storage/engine"
require "action_controller/railtie"
require "action_mailer/railtie"
require "action_view/railtie"
# Skip ActionCable (new in Rails 5.0) as it adds '/cable' routes that we're not using
# require "action_cable/engine"
require "sprockets/railtie"
require "rails/test_unit/railtie"

Bundler.require(:default, Rails.env)

if ENV["ARVADOS_RAILS_LOG_TO_STDOUT"]
  Rails.logger = ActiveSupport::TaggedLogging.new(Logger.new(STDOUT))
end

module ArvadosWorkbench
  class Application < Rails::Application
    # The following is to avoid SafeYAML's warning message
    SafeYAML::OPTIONS[:default_mode] = :safe

    require_relative "arvados_config.rb"

    # Initialize configuration defaults for originally generated Rails version.
    config.load_defaults 5.1

    # Settings in config/environments/* take precedence over those specified here.
    # Application configuration should go into files in config/initializers
    # -- all .rb files in that directory are automatically loaded.

    # Custom directories with classes and modules you want to be autoloadable.
    # Autoload paths shouldn't be used anymore since Rails 5.0
    # See #15258 and https://github.com/rails/rails/issues/13142#issuecomment-74586224
    # config.autoload_paths += %W(#{config.root}/extras)

    # Only load the plugins named here, in the order given (default is alphabetical).
    # :all can be used as a placeholder for all plugins not explicitly named.
    # config.plugins = [ :exception_notification, :ssl_requirement, :all ]

    # Activate observers that should always be running.
    # config.active_record.observers = :cacher, :garbage_collector, :forum_observer

    # Set Time.zone default to the specified zone and make Active Record auto-convert to this zone.
    # Run "rake -D time" for a list of tasks for finding time zone names. Default is UTC.
    # config.time_zone = 'Central Time (US & Canada)'

    # The default locale is :en and all translations from config/locales/*.rb,yml are auto loaded.
    # config.i18n.load_path += Dir[Rails.root.join('my', 'locales', '*.{rb,yml}').to_s]
    # config.i18n.default_locale = :de

    # Configure the default encoding used in templates for Ruby 1.9.
    config.encoding = "utf-8"

    # Configure sensitive parameters which will be filtered from the log file.
    config.filter_parameters += [:password]

    # Enable escaping HTML in JSON.
    config.active_support.escape_html_entities_in_json = true

    # Use SQL instead of Active Record's schema dumper when creating the database.
    # This is necessary if your schema can't be completely dumped by the schema dumper,
    # like if you have constraints or database-specific column types
    # config.active_record.schema_format = :sql

    # Enable the asset pipeline
    config.assets.enabled = true

    # Version of your assets, change this if you want to expire all your assets
    config.assets.version = '1.0'

    # npm-rails loads top-level modules like window.Mithril, but we
    # also pull in some code from node_modules in application.js, like
    # mithril/stream/stream.
    config.assets.paths << Rails.root.join('node_modules')
  end
end
