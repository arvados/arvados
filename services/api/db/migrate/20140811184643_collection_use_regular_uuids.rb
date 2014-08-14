class CollectionUseRegularUuids < ActiveRecord::Migration
  def up
    add_column :collections, :name, :string
    add_column :collections, :description, :string
    add_column :collections, :properties, :text
    add_column :collections, :expire_time, :date
    remove_column :collections, :locator
    add_column :jobs, :name, :string

    say_with_time "Step 1. Move manifest hashes into portable_data_hash field" do
      ActiveRecord::Base.connection.execute("update collections set portable_data_hash=uuid, uuid=null")
    end

    say_with_time "Step 2. Create new collection objects from the name links in the table." do
      from_clause = %{
from links inner join collections on head_uuid=collections.portable_data_hash
where link_class='name' and collections.uuid is null
}
      links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, tail_uuid, links.name,
manifest_text, links.created_at, links.updated_at
#{from_clause}
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
      end
      ActiveRecord::Base.connection.execute "delete from links where links.uuid in (select links.uuid #{from_clause})"
    end

    say_with_time "Step 3. Create new collection objects from the can_read links in the table." do
      from_clause = %{
from links inner join collections on head_uuid=collections.portable_data_hash
where link_class='permission' and links.name='can_read' and collections.uuid is null
}
      links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, tail_uuid, manifest_text, links.created_at, links.updated_at
#{from_clause}
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
      end
      ActiveRecord::Base.connection.execute "delete from links where links.uuid in (select links.uuid #{from_clause})"
    end

    say_with_time "Step 4. Migrate remaining orphan collection objects" do
      links = ActiveRecord::Base.connection.select_all %{
select portable_data_hash, owner_uuid, manifest_text, created_at, updated_at
from collections
where uuid is null and portable_data_hash not in (select portable_data_hash from collections where uuid is not null)
}
      links.each do |d|
        ActiveRecord::Base.connection.execute %{
insert into collections (uuid, portable_data_hash, owner_uuid, manifest_text, created_at, updated_at)
values (#{ActiveRecord::Base.connection.quote Collection.generate_uuid},
#{ActiveRecord::Base.connection.quote d['portable_data_hash']},
#{ActiveRecord::Base.connection.quote d['owner_uuid']},
#{ActiveRecord::Base.connection.quote d['manifest_text']},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']})
}
      end
    end

    say_with_time "Step 5. Delete old collection objects." do
      ActiveRecord::Base.connection.execute("delete from collections where uuid is null")
    end

    say_with_time "Step 6. Delete permission links where tail_uuid is a collection (invalid records)" do
      ActiveRecord::Base.connection.execute %{
delete from links where links.uuid in (select links.uuid
from links
where tail_uuid like '________________________________+%' and link_class='permission' )
}
    end

    say_with_time "Step 7. Migrate collection -> collection provenance links to jobs" do
      from_clause = %{
from links
where head_uuid like '________________________________+%' and tail_uuid like '________________________________+%' and links.link_class = 'provenance'
}
      links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, tail_uuid, links.created_at, links.updated_at, links.owner_uuid
#{from_clause}
}
      links.each do |d|
        newuuid = Job.generate_uuid
        ActiveRecord::Base.connection.execute %{
insert into jobs (uuid, script_parameters, output, running, success, created_at, updated_at, owner_uuid)
values (#{ActiveRecord::Base.connection.quote newuuid},
#{ActiveRecord::Base.connection.quote "---\ninput: "+d['tail_uuid']},
#{ActiveRecord::Base.connection.quote d['head_uuid']},
#{ActiveRecord::Base.connection.quote false},
#{ActiveRecord::Base.connection.quote true},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']},
#{ActiveRecord::Base.connection.quote d['owner_uuid']})
}
      end
      ActiveRecord::Base.connection.execute "delete from links where links.uuid in (select links.uuid #{from_clause})"
    end

    say_with_time "Step 8. Migrate remaining links with head_uuid pointing to collections" do
      from_clause = %{
from links inner join collections on links.head_uuid=portable_data_hash
where collections.uuid is not null
}
      links = ActiveRecord::Base.connection.select_all %{
select links.uuid, collections.uuid as collectionuuid, tail_uuid, link_class, links.properties,
links.name, links.created_at, links.updated_at, links.owner_uuid
#{from_clause}
}
      links.each do |d|
        ActiveRecord::Base.connection.execute %{
insert into links (uuid, head_uuid, tail_uuid, link_class, name, properties, created_at, updated_at, owner_uuid)
values (#{ActiveRecord::Base.connection.quote Link.generate_uuid},
#{ActiveRecord::Base.connection.quote d['collectionuuid']},
#{ActiveRecord::Base.connection.quote d['tail_uuid']},
#{ActiveRecord::Base.connection.quote d['link_class']},
#{ActiveRecord::Base.connection.quote d['name']},
#{ActiveRecord::Base.connection.quote d['properties']},
#{ActiveRecord::Base.connection.quote d['created_at']},
#{ActiveRecord::Base.connection.quote d['updated_at']},
#{ActiveRecord::Base.connection.quote d['owner_uuid']})
}
      end
      ActiveRecord::Base.connection.execute "delete from links where links.uuid in (select links.uuid #{from_clause})"
    end

    say_with_time "Step 9. Delete any remaining name links" do
      ActiveRecord::Base.connection.execute("delete from links where link_class='name'")
    end

    say_with_time "Step 10. Validate links table" do
      links = ActiveRecord::Base.connection.select_all %{
select links.uuid, head_uuid, tail_uuid, link_class, name
from links
where head_uuid like '________________________________+%' or tail_uuid like '________________________________+%'
}
      links.each do |d|
        raise "Bad row #{d}"
      end
    end

  end

  def down
    # Not gonna happen.
  end
end
