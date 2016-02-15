class AddUuidToApiTokenSearchIndex < ActiveRecord::Migration
  def up
    begin
      remove_index :api_client_authorizations, :name => 'api_client_authorizations_search_index'
    rescue
    end
    add_index :api_client_authorizations,
              ["api_token", "created_by_ip_address", "last_used_by_ip_address", "default_owner_uuid", "uuid"],
              name: "api_client_authorizations_search_index"
  end

  def down
    begin
      remove_index :api_client_authorizations, :name => 'api_client_authorizations_search_index'
    rescue
    end
	  add_index :api_client_authorizations,
              ["api_token", "created_by_ip_address", "last_used_by_ip_address", "default_owner_uuid"],
              name: "api_client_authorizations_search_index"
  end
end
