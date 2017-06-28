# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameForeignUuidAttributes < ActiveRecord::Migration
  def change
    rename_column :api_client_authorizations, :default_owner, :default_owner_uuid
    [:api_clients, :authorized_keys, :collections,
     :groups, :humans, :job_tasks, :jobs, :keep_disks,
     :links, :logs, :nodes, :pipeline_instances, :pipeline_templates,
     :repositories, :specimens, :traits, :users, :virtual_machines].each do |t|
      rename_column t, :owner, :owner_uuid
      rename_column t, :modified_by_client, :modified_by_client_uuid
      rename_column t, :modified_by_user, :modified_by_user_uuid
    end
    rename_column :collections, :redundancy_confirmed_by_client, :redundancy_confirmed_by_client_uuid
    rename_column :jobs, :is_locked_by, :is_locked_by_uuid
    rename_column :job_tasks, :created_by_job_task, :created_by_job_task_uuid
  end
end
