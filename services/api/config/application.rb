# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require_relative "boot"

require "rails"
# Pick the frameworks you want:
require "active_model/railtie"
require "active_job/railtie"
require "active_record/railtie"
# require "active_storage/engine"
require "action_controller/railtie"
require "action_mailer/railtie"
# require "action_mailbox/engine"
# require "action_text/engine"
require "action_view/railtie"
# require "action_cable/engine"
require "sprockets/railtie"
require "rails/test_unit/railtie"

# Require the gems listed in Gemfile, including any gems
# you've limited to :test, :development, or :production.
Bundler.require(*Rails.groups)

if ENV["ARVADOS_RAILS_LOG_TO_STDOUT"]
  Rails.logger = ActiveSupport::TaggedLogging.new(Logger.new(STDOUT))
end

module Server
  class Application < Rails::Application

    require_relative "arvados_config.rb"

    # Initialize configuration defaults for specified Rails version.
    config.load_defaults 5.2

    # Configuration for the application, engines, and railties goes here.
    #
    # These settings can be overridden in specific environments using the files
    # in config/environments, which are processed later.
    #
    # config.time_zone = "Central Time (US & Canada)"
    # config.eager_load_paths << Rails.root.join("extras")

    # We use db/structure.sql instead of db/schema.rb.
    config.active_record.schema_format = :sql

    config.eager_load = true

    config.active_support.test_order = :sorted

    # container_request records can contain arbitrary data structures
    # in mounts.*.content, so rails must not munge them.
    config.action_dispatch.perform_deep_munge = false

    # force_ssl's redirect-to-https feature doesn't work when the
    # client supplies a port number, and prevents arvados-controller
    # from connecting to Rails internally via plain http.
    config.ssl_options = {redirect: false}

    # Before using the filesystem backend for Rails.cache, check
    # whether we own the relevant directory. If we don't, using it is
    # likely to either fail or (if we're root) pollute it and cause
    # other processes to fail later.
    default_cache_path = Rails.root.join('tmp', 'cache')
    if not File.owned?(default_cache_path)
      if File.exist?(default_cache_path)
        why = "owner (uid=#{File::Stat.new(default_cache_path).uid}) " +
          "is not me (uid=#{Process.euid})"
      else
        why = "does not exist"
      end
      STDERR.puts("Defaulting to memory cache, " +
                  "because #{default_cache_path} #{why}")
      config.cache_store = :memory_store
    else
      require Rails.root.join('lib/safer_file_store')
      config.cache_store = ::SaferFileStore.new(default_cache_path)
    end
  end
end
