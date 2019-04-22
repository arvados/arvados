class FillMissingModifiedAt < ActiveRecord::Migration
  def up
    Collection.where('modified_at is null').update_all('modified_at = created_at')
  end
  def down
  end
end
