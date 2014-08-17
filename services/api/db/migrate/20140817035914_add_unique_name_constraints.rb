class AddUniqueNameConstraints < ActiveRecord::Migration
  def change
    # Need some code to ensure uniqueness before adding constraints.

    add_index(:collections, [:owner_uuid, :name], unique: true,
              name: 'collection_owner_uuid_name_unique')
    add_index(:pipeline_templates, [:owner_uuid, :name], unique: true,
              name: 'pipeline_template_owner_uuid_name_unique')
    add_index(:pipeline_instances, [:owner_uuid, :name], unique: true,
              name: 'pipeline_instance_owner_uuid_name_unique')
    add_index(:jobs, [:owner_uuid, :name], unique: true,
              name: 'jobs_owner_uuid_name_unique')
    add_index(:groups, [:owner_uuid, :name], unique: true,
              name: 'groups_owner_uuid_name_unique')
  end
end
