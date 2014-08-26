class AddUniqueNameConstraints < ActiveRecord::Migration
  def change
    # Ensure uniqueness before adding constraints.
    ["collections", "pipeline_templates", "pipeline_instances", "jobs", "groups"].each do |table|
      rows = ActiveRecord::Base.connection.select_all %{
select uuid, owner_uuid, name from #{table} order by owner_uuid, name
}
      prev = {}
      n = 1
      rows.each do |r|
        if r["owner_uuid"] == prev["owner_uuid"] and !r["name"].nil? and r["name"] == prev["name"]
          n += 1
          ActiveRecord::Base.connection.execute %{
update #{table} set name='#{r["name"]} #{n}' where uuid='#{r["uuid"]}'
}
        else
          n = 1
        end
        prev = r
      end
    end

    add_index(:collections, [:owner_uuid, :name], unique: true,
              name: 'collection_owner_uuid_name_unique')
    add_index(:pipeline_templates, [:owner_uuid, :name], unique: true,
              name: 'pipeline_template_owner_uuid_name_unique')
    add_index(:groups, [:owner_uuid, :name], unique: true,
              name: 'groups_owner_uuid_name_unique')
  end
end
