class CollectionUseRegularUuids < ActiveRecord::Migration
  def up
    add_column :collections, :name, :string
    add_column :collections, :description, :string
    add_column :collections, :properties, :string
    add_column :collections, :expire_time, :date
    remove_column :collections, :locator

    # Step 1.  Move manifest hashes into portable_data_hash field
    ActiveRecord::Base.connection.execute("update collections set portable_data_hash=uuid, uuid=null")

    # Step 2.  Create new collection objects from the name links in the table.
    links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, tail_uuid, links.name, manifest_text, links.created_at, links.updated_at
from links inner join collections on head_uuid=collections.portable_data_hash
where link_class='name' and collections.uuid is null
}
    links.each do |d|
      ActiveRecord::Base.connection.execute %{
insert into collections (uuid, portable_data_hash, owner_uuid, name, manifest_text, created_at, updated_at)
values (#{ActiveRecord::Base.connection.quote Collection.generate_uuid},
#{ActiveRecord::Base.connection.quote d['head_uuid']},
#{ActiveRecord::Base.connection.quote d['tail_uuid']},
#{ActiveRecord::Base.connection.quote d['name']},
#{ActiveRecord::Base.connection.quote d['manifest_text']},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']})
}
      ActiveRecord::Base.connection.execute("delete from links where uuid=#{ActiveRecord::Base.connection.quote d['uuid']}")
    end

    # Step 3.  Create new collection objects from the can_read links in the table.
    links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, tail_uuid, manifest_text, links.created_at, links.updated_at
from links inner join collections on head_uuid=collections.portable_data_hash
where link_class='permission' and links.name='can_read' and collections.uuid is null
}
    links.each do |d|
      ActiveRecord::Base.connection.execute %{
insert into collections (uuid, portable_data_hash, owner_uuid, manifest_text, created_at, updated_at)
values (#{ActiveRecord::Base.connection.quote Collection.generate_uuid},
#{ActiveRecord::Base.connection.quote d['head_uuid']},
#{ActiveRecord::Base.connection.quote d['tail_uuid']},
#{ActiveRecord::Base.connection.quote d['manifest_text']},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']})
}
      ActiveRecord::Base.connection.execute("delete from links where uuid=#{ActiveRecord::Base.connection.quote d['uuid']}")
    end

    # Step 4.  Delete permission links with tail_uuid of a collection (these records are just invalid)
    links = ActiveRecord::Base.connection.select_all "select links.uuid from links inner join collections on links.tail_uuid=collections.portable_data_hash where link_class='permission'"
    links.each do |d|
      ActiveRecord::Base.connection.execute("delete from links where uuid=#{ActiveRecord::Base.connection.quote d['uuid']}")
    end

    # Step 5. Migrate other links
    # 5.1 migrate head_uuid that look like collections
    links = ActiveRecord::Base.connection.select_all %{
select links.uuid, collections.uuid as coluuid, tail_uuid, link_class, links.properties, links.name, links.created_at, links.updated_at, links.owner_uuid
from links inner join collections on links.head_uuid=portable_data_hash
where collections.uuid is not null and links.link_class != 'name' and links.link_class != 'permission'
}
    links.each do |d|
      ActiveRecord::Base.connection.execute %{
insert into links (uuid, head_uuid, tail_uuid, link_class, name, properties, created_at, updated_at, owner_uuid)
values (#{ActiveRecord::Base.connection.quote Link.generate_uuid},
#{ActiveRecord::Base.connection.quote d['coluuid']},
#{ActiveRecord::Base.connection.quote d['tail_uuid']},
#{ActiveRecord::Base.connection.quote d['link_class']},
#{ActiveRecord::Base.connection.quote d['name']},
#{ActiveRecord::Base.connection.quote d['properties']},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']},
#{ActiveRecord::Base.connection.quote d['owner_uuid']})
}
      ActiveRecord::Base.connection.execute("delete from links where uuid=#{ActiveRecord::Base.connection.quote d['uuid']}")
    end

    # 5.2 migrate tail_uuid that look like collections
    links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, collections.uuid as coluuid, link_class, links.properties, links.name, links.created_at, links.updated_at, links.owner_uuid
from links inner join collections on links.tail_uuid=portable_data_hash
where tail_uuid like '________________________________+%' and collections.uuid is not null and links.link_class != 'name' and links.link_class != 'permission'
}
    links.each do |d|
      newuuid = Link.generate_uuid
      ActiveRecord::Base.connection.execute %{
insert into links (uuid, head_uuid, tail_uuid, link_class, name, properties, created_at, updated_at, owner_uuid)
values (#{ActiveRecord::Base.connection.quote newuuid},
#{ActiveRecord::Base.connection.quote d['head_uuid']},
#{ActiveRecord::Base.connection.quote d['coluuid']},
#{ActiveRecord::Base.connection.quote d['link_class']},
#{ActiveRecord::Base.connection.quote d['name']},
#{ActiveRecord::Base.connection.quote d['properties']},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']},
#{ActiveRecord::Base.connection.quote d['owner_uuid']})
}
      ActiveRecord::Base.connection.execute("delete from links where uuid=#{ActiveRecord::Base.connection.quote d['uuid']}")
    end

    # Step 7. Delete old collection objects.
    ActiveRecord::Base.connection.execute("delete from collections where uuid is null")
  end

  def down
    #remove_column :collections, :name
    #remove_column :collections, :description
    #remove_column :collections, :properties
    #remove_column :collections, :expire_time

  end
end
