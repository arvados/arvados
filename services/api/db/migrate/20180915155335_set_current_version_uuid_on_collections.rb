class SetCurrentVersionUuidOnCollections < ActiveRecord::Migration
  def up
    # Set the current version uuid as itself
    Collection.where(current_version_uuid: nil).update_all("current_version_uuid=uuid")
  end

  def down
  end
end
