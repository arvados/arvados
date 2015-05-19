class ReadOnlyOnKeepServices < ActiveRecord::Migration
  def change
    add_column :keep_services, :read_only, :boolean, null: false, default: false
  end
end
