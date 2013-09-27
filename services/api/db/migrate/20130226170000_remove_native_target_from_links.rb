class RemoveNativeTargetFromLinks < ActiveRecord::Migration
  def up
    remove_column :links, :native_target_id
    remove_column :links, :native_target_type
  end
  def down
    add_column :links, :native_target_id, :integer
    add_column :links, :native_target_type, :string
  end
end
