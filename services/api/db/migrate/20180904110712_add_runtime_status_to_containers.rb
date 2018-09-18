class AddRuntimeStatusToContainers < ActiveRecord::Migration
  def change
    add_column :containers, :runtime_status, :jsonb, default: {}
    add_index :containers, :runtime_status, using: :gin
  end
end
