class DatabaseController < ApplicationController
  skip_before_filter :find_object_by_uuid
  skip_before_filter :render_404_if_no_object
  before_filter :admin_required
  def reset
    raise ArvadosModel::PermissionDeniedError unless Rails.env == 'test'

    # Sanity check: If someone has actually logged in here, this might
    # not really be a throwaway database. Client test suites should
    # use @example.com email addresses when creating user records, so
    # we can tell they're not valuable.
    user_uuids = User.
      where('email is null or email not like ?', '%@example.com').
      collect &:uuid
    fixture_uuids =
      YAML::load_file(File.expand_path('../../../test/fixtures/users.yml',
                                       __FILE__)).
      values.collect { |u| u['uuid'] }
    unexpected_uuids = user_uuids - fixture_uuids
    if unexpected_uuids.any?
      logger.error("Running in test environment, but non-fixture users exist: " +
                   "#{unexpected_uuids}")
      raise ArvadosModel::PermissionDeniedError
    end

    require 'active_record/fixtures'

    # What kinds of fixtures do we have?
    fixturesets = Dir.glob(Rails.root.join('test', 'fixtures', '*.yml')).
      collect { |yml| yml.match(/([^\/]*)\.yml$/)[1] }

    ActiveRecord::Base.transaction do
      # Avoid deadlock by locking all tables before doing anything
      # drastic.
      table_names = '"' + ActiveRecord::Base.connection.tables.join('","') + '"'
      ActiveRecord::Base.connection.execute \
      "LOCK TABLE #{table_names} IN ACCESS EXCLUSIVE MODE"

      # Delete existing fixtures (and everything else) from fixture
      # tables
      fixturesets.each do |x|
        x.classify.constantize.unscoped.delete_all
      end

      # create_fixtures() is a no-op for cached fixture sets, so
      # uncache them all.
      ActiveRecord::Fixtures.reset_cache
      ActiveRecord::Fixtures.
        create_fixtures(Rails.root.join('test', 'fixtures'), fixturesets)

      # Dump cache of permissions etc.
      Rails.cache.clear
      ActiveRecord::Base.connection.clear_query_cache

      # Reload database seeds
      DatabaseSeeds.install
    end

    # Done.
    render json: {success: true}
  end
end
