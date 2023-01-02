# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class BigintId < ActiveRecord::Migration[5.2]
  disable_ddl_transaction!
  def up
    change_column :api_client_authorizations, :id, :bigint
    change_column :api_client_authorizations, :api_client_id, :bigint
    change_column :api_client_authorizations, :user_id, :bigint
    change_column :api_clients, :id, :bigint
    change_column :authorized_keys, :id, :bigint
    change_column :collections, :id, :bigint
    change_column :container_requests, :id, :bigint
    change_column :containers, :id, :bigint
    change_column :groups, :id, :bigint
    change_column :humans, :id, :bigint
    change_column :job_tasks, :id, :bigint
    change_column :jobs, :id, :bigint
    change_column :keep_disks, :id, :bigint
    change_column :keep_services, :id, :bigint
    change_column :links, :id, :bigint
    change_column :logs, :id, :bigint
    change_column :nodes, :id, :bigint
    change_column :users, :id, :bigint
    change_column :pipeline_instances, :id, :bigint
    change_column :pipeline_templates, :id, :bigint
    change_column :repositories, :id, :bigint
    change_column :specimens, :id, :bigint
    change_column :traits, :id, :bigint
    change_column :virtual_machines, :id, :bigint
    change_column :workflows, :id, :bigint
  end

  def down
  end
end
