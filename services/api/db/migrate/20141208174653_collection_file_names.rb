class CollectionFileNames < ActiveRecord::Migration
  include CurrentApiClient

  def up
    add_column :collections, :file_names, :string, :limit => 2**12

    act_as_system_user do
      Collection.all.each do |c|
        if c.manifest_text
          file_names = []
          c.manifest_text.split.each do |part|
            file_name = part.rpartition(':')[-1]
            file_names << file_name if file_name != '.'
          end

          c.file_names = file_names.uniq.join(" ")[0,2**12]
          c.save!
        end
      end
    end
  end

  def down
    remove_column :collections, :file_names
  end
end
