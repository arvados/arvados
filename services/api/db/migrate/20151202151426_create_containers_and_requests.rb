class CreateContainersAndRequests < ActiveRecord::Migration
  def change
    create_table :containers do |t|
      t.string :uuid
      t.string :owner_uuid
      t.datetime :created_at
      t.datetime :modified_at
      t.string :modified_by_client_uuid
      t.string :modified_by_user_uuid
      t.string :state
      t.datetime :started_at
      t.datetime :finished_at
      t.string :log
      t.text :environment
      t.string :cwd
      t.text :command
      t.string :output_path
      t.text :mounts
      t.text :runtime_constraints
      t.string :output
      t.string :container_image
      t.float :progress
      t.integer :priority

      t.timestamps
    end

    create_table :container_requests do |t|
      t.string :uuid
      t.string :owner_uuid
      t.datetime :created_at
      t.datetime :modified_at
      t.string :modified_by_client_uuid
      t.string :modified_by_user_uuid
      t.string :name
      t.text :description
      t.text :properties
      t.string :state
      t.string :requesting_container_uuid
      t.string :container_uuid
      t.integer :container_count_max
      t.text :mounts
      t.text :runtime_constraints
      t.string :container_image
      t.text :environment
      t.string :cwd
      t.text :command
      t.string :output_path
      t.integer :priority
      t.datetime :expires_at
      t.text :filters

      t.timestamps
    end

    add_index :containers, :uuid, :unique => true
    add_index :container_requests, :uuid, :unique => true
  end
end
