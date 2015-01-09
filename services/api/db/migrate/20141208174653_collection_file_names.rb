class CollectionFileNames < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :collections, :file_names, :string, :limit => 2**13

    act_as_system_user do
      Collection.all.each do |c|
        if c.manifest_text
          file_names = Collection.manifest_files c.manifest_text
          update_sql "UPDATE collections SET file_names = '#{file_names}' WHERE uuid = '#{c.uuid}'"
        end
      end
    end
  end

  def down
    remove_column :collections, :file_names
  end
end
