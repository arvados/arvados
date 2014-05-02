class RenameLogInfoToProperties < ActiveRecord::Migration
  def change
    rename_column :logs, :info, :properties
  end
end
