class EmptyCollection < ActiveRecord::Migration
  include CurrentApiClient

  def up
    empty_collection
  end

  def down
    # do nothing when migrating down (having the empty collection
    # and a permission link for it is harmless)
  end
end
