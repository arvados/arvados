class RenameJobForeignUuidAttributes < ActiveRecord::Migration
  def change
    rename_column :jobs, :cancelled_by_client, :cancelled_by_client_uuid
    rename_column :jobs, :cancelled_by_user, :cancelled_by_user_uuid
  end
end
