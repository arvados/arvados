class CollectionFileNames < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :collections, :file_names, :string, :limit => 2**13

    act_as_system_user do
      Collection.find_each(batch_size: 20) do |c|
        file_names = c.manifest_files
        ActiveRecord::Base.connection.execute "UPDATE collections
                    SET file_names = #{ActiveRecord::Base.connection.quote(file_names)}
                    WHERE uuid = '#{c.uuid}'"
      end
    end
  end

  def down
    remove_column :collections, :file_names
  end
end
