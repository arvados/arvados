class AddModifiedAtToLogs < ActiveRecord::Migration
  def change
    add_column :logs, :modified_at, :datetime
  end
end
