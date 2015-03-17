class ChangeCollectionExpiresAtToDatetime < ActiveRecord::Migration
  def up
    change_column :collections, :expires_at, :datetime
  end

  def down
    change_column :collections, :expires_at, :date
  end
end
