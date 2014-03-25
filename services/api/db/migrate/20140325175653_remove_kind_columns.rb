class RemoveKindColumns < ActiveRecord::Migration
  def up
    remove_column :links, :head_kind
    remove_column :links, :tail_kind
    remove_column :logs, :object_kind
  end

  def down
    add_column :links, :head_kind, :string
    add_column :links, :tail_kind, :string
    add_column :logs, :object_kind, :string
  end
end
