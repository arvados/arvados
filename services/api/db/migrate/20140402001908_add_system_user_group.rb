class AddSystemUserGroup < ActiveRecord::Migration
  include CurrentApiClient

  def up
    # Make sure the system group exists.
    system_group
  end

  def down
    # The system group does no harm if we don't delete it.
  end
end
