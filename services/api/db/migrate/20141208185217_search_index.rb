class SearchIndex < ActiveRecord::Migration
  def tables_with_searchable_columns
    {
      "api_client_authorizations" => ["api_token", "created_by_ip_address", "last_used_by_ip_address", "default_owner_uuid"],
      "api_clients" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "url_prefix"],
      "authorized_keys" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "key_type", "authorized_user_uuid"],
      "collections" => ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "redundancy_confirmed_by_client_uuid", "uuid", "name", "description"],
      "groups" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "description", "group_class"],
      "humans" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid"],
      "job_tasks" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "job_uuid", "created_by_job_task_uuid"],
      "jobs" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "submit_id", "script", "script_version", "cancelled_by_client_uuid", "cancelled_by_user_uuid", "output", "is_locked_by_uuid", "log", "repository", "supplied_script_version", "docker_image_locator", "description", "state", "arvados_sdk_version"],
      "keep_disks" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "ping_secret", "node_uuid", "filesystem_uuid", "keep_service_uuid"],
      "keep_services" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "service_host", "service_type"],
      "links" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "tail_uuid", "link_class", "name", "head_uuid"],
      "logs" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "object_uuid", "event_type", "object_owner_uuid"],
      "nodes" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "hostname", "domain", "ip_address", "job_uuid"],
      "pipeline_instances" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "pipeline_template_uuid", "name", "state", "description"],
      "pipeline_templates" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "description"],
      "repositories" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "fetch_url", "push_url"],
      "specimens" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "material"],
      "traits" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name"],
      "users" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "email", "first_name", "last_name", "identity_url", "default_owner_uuid"],
      "virtual_machines" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "hostname"],
    }
  end

  def up
    tables_with_searchable_columns.each do |table, columns|
      add_index(table.to_sym, columns, name: "#{table}_search_index")
    end
  end

  def down
    tables_with_searchable_columns.each do |table, columns|
      indexes = ActiveRecord::Base.connection.indexes(table)
      search_index = indexes.select do |index|
        index.name == "#{table}_search_index"
      end
      if !search_index.empty?
        remove_index(table.to_sym, name: "#{table}_search_index")
      end
    end
  end
end
