class AddContainerAuthUuidIndex < ActiveRecord::Migration
  def change
    add_index :containers, :auth_uuid
  end
end
