class AddGroupClassToGroups < ActiveRecord::Migration
  def change
    add_column :groups, :group_class, :string
    add_index :groups, :group_class
  end
end
