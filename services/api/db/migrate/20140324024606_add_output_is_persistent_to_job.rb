class AddOutputIsPersistentToJob < ActiveRecord::Migration
  def change
    add_column :jobs, :output_is_persistent, :boolean, null: false, default: false
  end
end
