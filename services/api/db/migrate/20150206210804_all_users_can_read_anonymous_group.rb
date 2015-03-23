class AllUsersCanReadAnonymousGroup < ActiveRecord::Migration
  include CurrentApiClient

  def up
    anonymous_group_read_permission
  end

  def down
    # Do nothing - it's too dangerous to try to figure out whether or not
    # the permission was created by the migration.
  end
end
