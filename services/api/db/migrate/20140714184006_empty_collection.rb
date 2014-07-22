class EmptyCollection < ActiveRecord::Migration
  include CurrentApiClient

  def up
    act_as_system_user do
      empty_collection
    end
  end

  def down
    # do nothing when migrating down (having the empty collection
    # and a permission link for it is harmless)
  end
end
