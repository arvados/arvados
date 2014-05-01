class AddObjectOwnerToLogs < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :logs, :object_owner_uuid, :string
    act_as_system_user do
      Log.find_each do |log|
        if log.properties[:new_attributes]
          log.object_owner_uuid = log.properties[:new_attributes][:owner_uuid]
        elsif log.properties[:old_attributes]
          log.object_owner_uuid = log.properties[:old_attributes][:owner_uuid]
        end
        log.save!
      end
    end
  end

  def down
    remove_column :logs, :object_owner_uuid
  end
end
