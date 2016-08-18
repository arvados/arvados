class CreateWorkflows < ActiveRecord::Migration
  def up
    create_table :workflows do |t|
      t.string :uuid
      t.string :owner_uuid
      t.datetime :created_at
      t.datetime :modified_at
      t.string :modified_by_client_uuid
      t.string :modified_by_user_uuid
      t.string :name
      t.text :description
      t.text :workflow

      t.timestamps
    end

    add_index :workflows, :uuid, :unique => true
    add_index :workflows, :owner_uuid
    add_index :workflows, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name"], name: 'workflows_search_idx'
    execute "CREATE INDEX workflows_full_text_search_idx ON workflows USING gin(#{Workflow.full_text_tsvector});"
  end

  def down
    remove_index :workflows, :name => 'workflows_full_text_search_idx'
    remove_index :workflows, :name => 'workflows_search_idx'
    remove_index :workflows, :owner_uuid
    remove_index :workflows, :uuid
    drop_table :workflows
  end
end
