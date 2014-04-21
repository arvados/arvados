# encoding: UTF-8
# This file is auto-generated from the current state of the database. Instead
# of editing this file, please use the migrations feature of Active Record to
# incrementally modify your database, and then regenerate this schema definition.
#
# Note that this schema.rb definition is the authoritative source for your
# database schema. If you need to create the application database on another
# system, you should be using db:schema:load, not running all the migrations
# from scratch. The latter is a flawed and unsustainable approach (the more migrations
# you'll amass, the slower it'll run and the greater likelihood for issues).
#
# It's strongly recommended to check this file into your version control system.

ActiveRecord::Schema.define(:version => 20140421140924) do

  create_table "api_client_authorizations", :force => true do |t|
    t.string   "api_token",                                           :null => false
    t.integer  "api_client_id",                                       :null => false
    t.integer  "user_id",                                             :null => false
    t.string   "created_by_ip_address"
    t.string   "last_used_by_ip_address"
    t.datetime "last_used_at"
    t.datetime "expires_at"
    t.datetime "created_at",                                          :null => false
    t.datetime "updated_at",                                          :null => false
    t.string   "default_owner_uuid"
    t.text     "scopes",                  :default => "---\n- all\n", :null => false
  end

  add_index "api_client_authorizations", ["api_client_id"], :name => "index_api_client_authorizations_on_api_client_id"
  add_index "api_client_authorizations", ["api_token"], :name => "index_api_client_authorizations_on_api_token", :unique => true
  add_index "api_client_authorizations", ["expires_at"], :name => "index_api_client_authorizations_on_expires_at"
  add_index "api_client_authorizations", ["user_id"], :name => "index_api_client_authorizations_on_user_id"

  create_table "api_clients", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "name"
    t.string   "url_prefix"
    t.datetime "created_at",                                 :null => false
    t.datetime "updated_at",                                 :null => false
    t.boolean  "is_trusted",              :default => false
  end

  add_index "api_clients", ["created_at"], :name => "index_api_clients_on_created_at"
  add_index "api_clients", ["modified_at"], :name => "index_api_clients_on_modified_at"
  add_index "api_clients", ["uuid"], :name => "index_api_clients_on_uuid", :unique => true

  create_table "authorized_keys", :force => true do |t|
    t.string   "uuid",                    :null => false
    t.string   "owner_uuid",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "name"
    t.string   "key_type"
    t.string   "authorized_user_uuid"
    t.text     "public_key"
    t.datetime "expires_at"
    t.datetime "created_at",              :null => false
    t.datetime "updated_at",              :null => false
  end

  add_index "authorized_keys", ["authorized_user_uuid", "expires_at"], :name => "index_authorized_keys_on_authorized_user_uuid_and_expires_at"
  add_index "authorized_keys", ["uuid"], :name => "index_authorized_keys_on_uuid", :unique => true

  create_table "collections", :force => true do |t|
    t.string   "locator"
    t.string   "owner_uuid"
    t.datetime "created_at"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "portable_data_hash"
    t.string   "name"
    t.integer  "redundancy"
    t.string   "redundancy_confirmed_by_client_uuid"
    t.datetime "redundancy_confirmed_at"
    t.integer  "redundancy_confirmed_as"
    t.datetime "updated_at"
    t.string   "uuid"
    t.text     "manifest_text"
  end

  add_index "collections", ["created_at"], :name => "index_collections_on_created_at"
  add_index "collections", ["modified_at"], :name => "index_collections_on_modified_at"
  add_index "collections", ["uuid"], :name => "index_collections_on_uuid", :unique => true

  create_table "commit_ancestors", :force => true do |t|
    t.string   "repository_name"
    t.string   "descendant",                         :null => false
    t.string   "ancestor",                           :null => false
    t.boolean  "is",              :default => false, :null => false
    t.datetime "created_at",                         :null => false
    t.datetime "updated_at",                         :null => false
  end

  add_index "commit_ancestors", ["descendant", "ancestor"], :name => "index_commit_ancestors_on_descendant_and_ancestor", :unique => true

  create_table "commits", :force => true do |t|
    t.string   "repository_name"
    t.string   "sha1"
    t.string   "message"
    t.datetime "created_at"
    t.datetime "updated_at"
  end

  add_index "commits", ["repository_name", "sha1"], :name => "index_commits_on_repository_name_and_sha1", :unique => true

  create_table "groups", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "name"
    t.text     "description"
    t.datetime "updated_at",              :null => false
    t.string   "group_class"
  end

  add_index "groups", ["created_at"], :name => "index_groups_on_created_at"
  add_index "groups", ["group_class"], :name => "index_groups_on_group_class"
  add_index "groups", ["modified_at"], :name => "index_groups_on_modified_at"
  add_index "groups", ["uuid"], :name => "index_groups_on_uuid", :unique => true

  create_table "humans", :force => true do |t|
    t.string   "uuid",                    :null => false
    t.string   "owner_uuid",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.text     "properties"
    t.datetime "created_at"
    t.datetime "updated_at"
  end

  add_index "humans", ["uuid"], :name => "index_humans_on_uuid", :unique => true

  create_table "job_tasks", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "job_uuid"
    t.integer  "sequence"
    t.text     "parameters"
    t.text     "output"
    t.float    "progress"
    t.boolean  "success"
    t.datetime "created_at",                            :null => false
    t.datetime "updated_at",                            :null => false
    t.string   "created_by_job_task_uuid"
    t.integer  "qsequence",                :limit => 8
  end

  add_index "job_tasks", ["created_at"], :name => "index_job_tasks_on_created_at"
  add_index "job_tasks", ["job_uuid"], :name => "index_job_tasks_on_job_uuid"
  add_index "job_tasks", ["modified_at"], :name => "index_job_tasks_on_modified_at"
  add_index "job_tasks", ["sequence"], :name => "index_job_tasks_on_sequence"
  add_index "job_tasks", ["success"], :name => "index_job_tasks_on_success"
  add_index "job_tasks", ["uuid"], :name => "index_job_tasks_on_uuid", :unique => true

  create_table "jobs", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "submit_id"
    t.string   "script"
    t.string   "script_version"
    t.text     "script_parameters"
    t.string   "cancelled_by_client_uuid"
    t.string   "cancelled_by_user_uuid"
    t.datetime "cancelled_at"
    t.datetime "started_at"
    t.datetime "finished_at"
    t.boolean  "running"
    t.boolean  "success"
    t.string   "output"
    t.datetime "created_at"
    t.datetime "updated_at"
    t.string   "priority"
    t.string   "is_locked_by_uuid"
    t.string   "log"
    t.text     "tasks_summary"
    t.text     "runtime_constraints"
    t.boolean  "nondeterministic"
    t.string   "repository"
    t.boolean  "output_is_persistent",     :default => false, :null => false
  end

  add_index "jobs", ["created_at"], :name => "index_jobs_on_created_at"
  add_index "jobs", ["finished_at"], :name => "index_jobs_on_finished_at"
  add_index "jobs", ["modified_at"], :name => "index_jobs_on_modified_at"
  add_index "jobs", ["output"], :name => "index_jobs_on_output"
  add_index "jobs", ["script"], :name => "index_jobs_on_script"
  add_index "jobs", ["started_at"], :name => "index_jobs_on_started_at"
  add_index "jobs", ["submit_id"], :name => "index_jobs_on_submit_id", :unique => true
  add_index "jobs", ["uuid"], :name => "index_jobs_on_uuid", :unique => true

  create_table "keep_disks", :force => true do |t|
    t.string   "uuid",                                      :null => false
    t.string   "owner_uuid",                                :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "ping_secret",                               :null => false
    t.string   "node_uuid"
    t.string   "filesystem_uuid"
    t.integer  "bytes_total"
    t.integer  "bytes_free"
    t.boolean  "is_readable",             :default => true, :null => false
    t.boolean  "is_writable",             :default => true, :null => false
    t.datetime "last_read_at"
    t.datetime "last_write_at"
    t.datetime "last_ping_at"
    t.datetime "created_at",                                :null => false
    t.datetime "updated_at",                                :null => false
    t.string   "service_host"
    t.integer  "service_port"
    t.boolean  "service_ssl_flag"
  end

  add_index "keep_disks", ["filesystem_uuid"], :name => "index_keep_disks_on_filesystem_uuid"
  add_index "keep_disks", ["last_ping_at"], :name => "index_keep_disks_on_last_ping_at"
  add_index "keep_disks", ["node_uuid"], :name => "index_keep_disks_on_node_uuid"
  add_index "keep_disks", ["service_host", "service_port", "last_ping_at"], :name => "keep_disks_service_host_port_ping_at_index"
  add_index "keep_disks", ["uuid"], :name => "index_keep_disks_on_uuid", :unique => true

  create_table "links", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "tail_uuid"
    t.string   "link_class"
    t.string   "name"
    t.string   "head_uuid"
    t.text     "properties"
    t.datetime "updated_at"
  end

  add_index "links", ["created_at"], :name => "index_links_on_created_at"
  add_index "links", ["head_uuid"], :name => "index_links_on_head_uuid"
  add_index "links", ["modified_at"], :name => "index_links_on_modified_at"
  add_index "links", ["tail_uuid"], :name => "index_links_on_tail_uuid"
  add_index "links", ["uuid"], :name => "index_links_on_uuid", :unique => true

  create_table "logs", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.string   "object_uuid"
    t.datetime "event_at"
    t.string   "event_type"
    t.text     "summary"
    t.text     "properties"
    t.datetime "created_at",              :null => false
    t.datetime "updated_at",              :null => false
    t.datetime "modified_at"
  end

  add_index "logs", ["created_at"], :name => "index_logs_on_created_at"
  add_index "logs", ["event_at"], :name => "index_logs_on_event_at"
  add_index "logs", ["event_type"], :name => "index_logs_on_event_type"
  add_index "logs", ["modified_at"], :name => "index_logs_on_modified_at"
  add_index "logs", ["object_uuid"], :name => "index_logs_on_object_uuid"
  add_index "logs", ["summary"], :name => "index_logs_on_summary"
  add_index "logs", ["uuid"], :name => "index_logs_on_uuid", :unique => true

  create_table "nodes", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.integer  "slot_number"
    t.string   "hostname"
    t.string   "domain"
    t.string   "ip_address"
    t.datetime "first_ping_at"
    t.datetime "last_ping_at"
    t.text     "info"
    t.datetime "updated_at",              :null => false
  end

  add_index "nodes", ["created_at"], :name => "index_nodes_on_created_at"
  add_index "nodes", ["hostname"], :name => "index_nodes_on_hostname"
  add_index "nodes", ["modified_at"], :name => "index_nodes_on_modified_at"
  add_index "nodes", ["slot_number"], :name => "index_nodes_on_slot_number", :unique => true
  add_index "nodes", ["uuid"], :name => "index_nodes_on_uuid", :unique => true

  create_table "pipeline_instances", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at"
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "pipeline_template_uuid"
    t.string   "name"
    t.text     "components"
    t.boolean  "success"
    t.boolean  "active",                  :default => false
    t.datetime "updated_at"
    t.text     "properties"
  end

  add_index "pipeline_instances", ["created_at"], :name => "index_pipeline_instances_on_created_at"
  add_index "pipeline_instances", ["modified_at"], :name => "index_pipeline_instances_on_modified_at"
  add_index "pipeline_instances", ["uuid"], :name => "index_pipeline_instances_on_uuid", :unique => true

  create_table "pipeline_templates", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "name"
    t.text     "components"
    t.datetime "updated_at",              :null => false
  end

  add_index "pipeline_templates", ["created_at"], :name => "index_pipeline_templates_on_created_at"
  add_index "pipeline_templates", ["modified_at"], :name => "index_pipeline_templates_on_modified_at"
  add_index "pipeline_templates", ["uuid"], :name => "index_pipeline_templates_on_uuid", :unique => true

  create_table "repositories", :force => true do |t|
    t.string   "uuid",                    :null => false
    t.string   "owner_uuid",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "name"
    t.string   "fetch_url"
    t.string   "push_url"
    t.datetime "created_at",              :null => false
    t.datetime "updated_at",              :null => false
  end

  add_index "repositories", ["name"], :name => "index_repositories_on_name", :unique => true
  add_index "repositories", ["uuid"], :name => "index_repositories_on_uuid", :unique => true

  create_table "specimens", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "material"
    t.datetime "updated_at",              :null => false
    t.text     "properties"
  end

  add_index "specimens", ["created_at"], :name => "index_specimens_on_created_at"
  add_index "specimens", ["modified_at"], :name => "index_specimens_on_modified_at"
  add_index "specimens", ["uuid"], :name => "index_specimens_on_uuid", :unique => true

  create_table "traits", :force => true do |t|
    t.string   "uuid",                    :null => false
    t.string   "owner_uuid",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "name"
    t.text     "properties"
    t.datetime "created_at",              :null => false
    t.datetime "updated_at",              :null => false
  end

  add_index "traits", ["name"], :name => "index_traits_on_name"
  add_index "traits", ["uuid"], :name => "index_traits_on_uuid", :unique => true

  create_table "users", :force => true do |t|
    t.string   "uuid"
    t.string   "owner_uuid"
    t.datetime "created_at",                                 :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "email"
    t.string   "first_name"
    t.string   "last_name"
    t.string   "identity_url"
    t.boolean  "is_admin"
    t.text     "prefs"
    t.datetime "updated_at",                                 :null => false
    t.string   "default_owner_uuid"
    t.boolean  "is_active",               :default => false
  end

  add_index "users", ["created_at"], :name => "index_users_on_created_at"
  add_index "users", ["modified_at"], :name => "index_users_on_modified_at"
  add_index "users", ["uuid"], :name => "index_users_on_uuid", :unique => true

  create_table "virtual_machines", :force => true do |t|
    t.string   "uuid",                    :null => false
    t.string   "owner_uuid",              :null => false
    t.string   "modified_by_client_uuid"
    t.string   "modified_by_user_uuid"
    t.datetime "modified_at"
    t.string   "hostname"
    t.datetime "created_at",              :null => false
    t.datetime "updated_at",              :null => false
  end

  add_index "virtual_machines", ["hostname"], :name => "index_virtual_machines_on_hostname"
  add_index "virtual_machines", ["uuid"], :name => "index_virtual_machines_on_uuid", :unique => true

end
