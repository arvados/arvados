class SearchIndex < ActiveRecord::Migration
  def tables_with_searchable_columns
    {
      "api_client_authorizations" => ["api_token", "created_by_ip_address", "last_used_by_ip_address", "default_owner_uuid", "scopes"],
      "api_clients" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "url_prefix"],
      "authorized_keys" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "key_type", "authorized_user_uuid", "public_key"],
      "collections" => ["owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "portable_data_hash", "redundancy_confirmed_by_client_uuid", "uuid", "manifest_text", "name", "description", "properties"],
      "groups" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "description", "group_class"],
      "humans" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "properties"],
      "job_tasks" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "job_uuid", "parameters", "output", "created_by_job_task_uuid"],
      "jobs" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "submit_id", "script", "script_version", "script_parameters", "cancelled_by_client_uuid", "cancelled_by_user_uuid", "output", "is_locked_by_uuid", "log", "tasks_summary", "runtime_constraints", "repository", "supplied_script_version", "docker_image_locator", "description", "state", "arvados_sdk_version"],
      "keep_disks" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "ping_secret", "node_uuid", "filesystem_uuid", "keep_service_uuid"],
      "keep_services" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "service_host", "service_type"],
      "links" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "tail_uuid", "link_class", "name", "head_uuid", "properties"],
      "logs" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "object_uuid", "event_type", "summary", "properties", "object_owner_uuid"],
      "nodes" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "hostname", "domain", "ip_address", "info", "properties", "job_uuid"],
      "pipeline_instances" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "pipeline_template_uuid", "name", "components", "properties", "state", "components_summary", "description"],
      "pipeline_templates" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "components", "description"],
      "repositories" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "fetch_url", "push_url"],
      "specimens" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "material", "properties"],
      "traits" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "name", "properties"],
      "users" => ["uuid", "owner_uuid", "modified_by_client_uuid", "modified_by_user_uuid", "email", "first_name", "last_name", "identity_url", "prefs", "default_owner_uuid"],
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
      remove_index(table.to_sym, name: "#{table}_search_index")
    end
  end
end
