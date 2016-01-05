class FixContainersIndex < ActiveRecord::Migration
  def up
    execute "CREATE INDEX container_requests_full_text_search_idx ON container_requests USING gin(#{ContainerRequest.full_text_tsvector});"
    add_index :container_requests, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "state", "requesting_container_uuid", "container_uuid", "container_image", "cwd", "output_path"], name: 'container_requests_search_index'
    add_index :containers, ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "state", "log", "cwd", "output_path", "output", "container_image"], name: 'containers_search_index'
    add_index :container_requests, :owner_uuid
    add_index :containers, :owner_uuid
  end

  def down
    remove_index :container_requests, :name => 'container_requests_full_text_search_idx'
    remove_index :container_requests, :name => 'container_requests_search_index'
    remove_index :containers, :name => 'containers_search_index'
    remove_index :container_requests, :owner_uuid
    remove_index :containers, :owner_uuid
  end
end
