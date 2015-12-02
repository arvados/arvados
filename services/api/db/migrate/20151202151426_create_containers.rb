class CreateContainers < ActiveRecord::Migration
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
      t.string :command
      t.string :output_path
      t.string :mounts
      t.string :runtime_constraints
      t.string :output
      t.string :container_image
      t.float :progress
      t.int :priority

      t.timestamps
    end
  end
end
