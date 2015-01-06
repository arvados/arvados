class CollectionFileNames < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :collections, :file_names, :string, :limit => 2**16

    act_as_system_user do
      Collection.all.each do |c|
        if c.manifest_text
          c.file_names = c.manifest_text[0, 2**16]
          c.save!
        end
      end
    end
  end

  def down
    remove_column :collections, :file_names
  end
end
