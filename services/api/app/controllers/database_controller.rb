class DatabaseController < ApplicationController
  skip_before_filter :find_object_by_uuid
  skip_before_filter :render_404_if_no_object
  before_filter :admin_required
  def reset
    raise ArvadosModel::PermissionDeniedError unless ENV['RAILS_ENV'] == 'test'

    require 'active_record/fixtures'

    # What kinds of fixtures do we have?
    fixturesets = Dir.glob(Rails.root.join('test', 'fixtures', '*.yml')).
      collect { |yml| yml.match(/([^\/]*)\.yml$/)[1] }

    # Delete existing fixtures (and everything else) from fixture
    # tables
    fixturesets.each do |x|
      x.classify.constantize.unscoped.delete_all
    end

    # create_fixtures() is a no-op for cached fixture sets, so uncache
    # them all.
    ActiveRecord::Fixtures.reset_cache
    ActiveRecord::Fixtures.
      create_fixtures(Rails.root.join('test', 'fixtures'), fixturesets)

    # Dump cache of permissions etc.
    Rails.cache.clear
    ActiveRecord::Base.connection.clear_query_cache

    # Reload database seeds
    DatabaseSeeds.install

    # Done.
    render json: {success: true}
  end
end
