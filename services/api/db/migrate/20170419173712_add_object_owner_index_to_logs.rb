class AddObjectOwnerIndexToLogs < ActiveRecord::Migration
  def change
    add_index :logs, :object_owner_uuid
  end
end
