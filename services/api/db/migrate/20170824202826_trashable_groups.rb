class TrashableGroups < ActiveRecord::Migration
  def change
    add_column :groups, :trash_at, :datetime
    add_column :groups, :delete_at, :datetime
    add_column :groups, :is_trashed, :boolean, null: false, default: false
  end
end
