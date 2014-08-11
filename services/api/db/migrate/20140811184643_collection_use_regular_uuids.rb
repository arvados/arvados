class CollectionUseRegularUuids < ActiveRecord::Migration
  def up
    add_column :collections, :name, :string
    add_column :collections, :description, :string
    add_column :collections, :properties, :string

    ActiveRecord::Base.connection.execute("update collections set portable_data_hash=uuid, uuid=null;")

    data = ActiveRecord::Base.connection.select_all("select head_uuid, tail_uuid, links.name, manifest_text from links inner join collections on head_uuid=collections.portable_data_hash where link_class='name'")
    created_at = Time.now
    data.each do |d|
      c = Collection.generate_uuid
      s = "insert into collections (uuid, portable_data_hash, owner_uuid, name, manifest_text, created_at, updated_at) values (#{ActiveRecord::Base.connection.quote c}, #{ActiveRecord::Base.connection.quote d['head_uuid']}, #{ActiveRecord::Base.connection.quote d['tail_uuid']}, #{ActiveRecord::Base.connection.quote d['name']}, #{ActiveRecord::Base.connection.quote d['manifest_text']}, #{ActiveRecord::Base.connection.quote created_at}, #{ActiveRecord::Base.connection.quote created_at})"
      ActiveRecord::Base.connection.execute(s)
    end

    data = ActiveRecord::Base.connection.select_all("select head_uuid, tail_uuid, manifest_text from links inner join collections on head_uuid=collections.portable_data_hash where link_class='permission' and links.name='can_read'")
    created_at = Time.now
    data.each do |d|
      c = Collection.generate_uuid
      s = "insert into collections (uuid, portable_data_hash, owner_uuid, name, manifest_text, created_at, updated_at) values (#{ActiveRecord::Base.connection.quote c}, #{ActiveRecord::Base.connection.quote d['head_uuid']}, #{ActiveRecord::Base.connection.quote d['tail_uuid']}, #{ActiveRecord::Base.connection.quote 'something something'}, #{ActiveRecord::Base.connection.quote d['manifest_text']}, #{ActiveRecord::Base.connection.quote created_at}, #{ActiveRecord::Base.connection.quote created_at})"
      ActiveRecord::Base.connection.execute(s)
    end


  end

  def down
    #remove_column :collections, :name
    #remove_column :collections, :description
    #remove_column :collections, :properties

  end
end
