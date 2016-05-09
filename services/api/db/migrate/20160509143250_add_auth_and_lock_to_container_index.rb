class AddAuthAndLockToContainerIndex < ActiveRecord::Migration
  Columns_were = ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "state", "log", "cwd", "output_path", "output", "container_image"]
  Columns = Columns_were + ["auth_uuid", "locked_by_uuid"]
  def up
    begin
      remove_index :containers, :name => 'containers_search_index'
    rescue
    end
    add_index(:containers, Columns, name: "containers_search_index")
  end

  def down
    begin
      remove_index :containers, :name => 'containers_search_index'
    rescue
    end
    add_index(:containers, Columns_were, name: "containers_search_index")
  end
end
