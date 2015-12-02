class CreateContainerRequests < ActiveRecord::Migration
  def change
    create_table :container_requests do |t|
      t.string :uuid
      t.string :owner_uuid
      t.datetime :created_at
      t.datetime :modified_at
      t.string :modified_by_client_uuid
      t.string :modified_by_user_uuid
      t.string :name
      t.text :description
      t.string :properties
      t.string :state
      t.string :requesting_container_uuid
      t.string :container_uuid
      t.int :container_count_max
      t.string :mounts
      t.string :runtime_constraints
      t.string :container_image
      t.string :environment
      t.string :cwd
      t.string :command
      t.string :output_path
      t.int :priority
      t.datetime :expires_at
      t.string :filters

      t.timestamps
    end
  end
end
