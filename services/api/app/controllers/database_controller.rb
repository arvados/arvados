# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class DatabaseController < ApplicationController
  skip_before_action :find_object_by_uuid
  skip_before_action :render_404_if_no_object
  before_action :admin_required
  around_action :silence_logs, only: [:reset]

  def reset
    raise ArvadosModel::PermissionDeniedError unless Rails.env == 'test'

    # Sanity check: If someone has actually logged in here, this might
    # not really be a throwaway database. Client test suites should
    # use @example.com email addresses when creating user records, so
    # we can tell they're not valuable.
    user_uuids = User.
      where('email is null or (email not like ? and email not like ?)', '%@example.com', '%.example.com').
      collect(&:uuid)
    fixture_uuids =
      YAML::load_file(File.expand_path('../../../test/fixtures/users.yml',
                                       __FILE__)).
      values.collect { |u| u['uuid'] }
    unexpected_uuids = user_uuids - fixture_uuids
    if unexpected_uuids.any?
      logger.error("Running in test environment, but non-fixture users exist: " +
                   "#{unexpected_uuids}" + "\nMaybe test users without @example.com email addresses were created?")
      raise ArvadosModel::PermissionDeniedError
    end

    require 'active_record/fixtures'

    # What kinds of fixtures do we have?
    fixturesets = Dir.glob(Rails.root.join('test', 'fixtures', '*.yml')).
      collect { |yml| yml.match(/([^\/]*)\.yml$/)[1] }

    # Don't reset keep_services: clients need to discover our
    # integration-testing keepstores, not test fixtures.
    fixturesets -= %w[keep_services]

    table_names = '"' + ActiveRecord::Base.connection.tables.join('","') + '"'

    attempts_left = 20
    begin
      ActiveRecord::Base.transaction do
        # Avoid deadlock by locking all tables before doing anything
        # drastic.
        ActiveRecord::Base.connection.execute \
        "LOCK TABLE #{table_names} IN ACCESS EXCLUSIVE MODE"

        # Delete existing fixtures (and everything else) from fixture
        # tables
        fixturesets.each do |x|
          x.classify.constantize.unscoped.delete_all
        end

        # create_fixtures() is a no-op for cached fixture sets, so
        # uncache them all.
        ActiveRecord::FixtureSet.reset_cache
        ActiveRecord::FixtureSet.
          create_fixtures(Rails.root.join('test', 'fixtures'), fixturesets)

        # Dump cache of permissions etc.
        Rails.cache.clear
        ActiveRecord::Base.connection.clear_query_cache

        # Reload database seeds
        DatabaseSeeds.install
      end
    rescue ActiveRecord::StatementInvalid => e
      if "#{e.inspect}" =~ /deadlock detected/i and (attempts_left -= 1) > 0
        logger.info "Waiting for lock -- #{e.inspect}"
        sleep 0.5
        retry
      end
      raise
    end

    require 'update_permissions'

    refresh_permissions
    refresh_trashed

    # Done.
    send_json success: true
  end

  protected

  def silence_logs
    Rails.logger.info("(logging level temporarily raised to :error, see #{__FILE__})")
    orig = ActiveRecord::Base.logger.level
    ActiveRecord::Base.logger.level = :error
    begin
      yield
    ensure
      ActiveRecord::Base.logger.level = orig
    end
  end
end
