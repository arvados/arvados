# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameProjectsToGroups < ActiveRecord::Migration
  def up
    rename_table :projects, :groups
    rename_index :groups, :index_projects_on_created_at, :index_groups_on_created_at
    rename_index :groups, :index_projects_on_modified_at, :index_groups_on_modified_at
    rename_index :groups, :index_projects_on_uuid, :index_groups_on_uuid
    Link.update_all({head_kind:'orvos#group'}, ['head_kind=?','orvos#project'])
    Link.update_all({tail_kind:'orvos#group'}, ['tail_kind=?','orvos#project'])
    Log.update_all({object_kind:'orvos#group'}, ['object_kind=?','orvos#project'])
  end

  def down
    Log.update_all({object_kind:'orvos#project'}, ['object_kind=?','orvos#group'])
    Link.update_all({tail_kind:'orvos#project'}, ['tail_kind=?','orvos#group'])
    Link.update_all({head_kind:'orvos#project'}, ['head_kind=?','orvos#group'])
    rename_index :groups, :index_groups_on_created_at, :index_projects_on_created_at
    rename_index :groups, :index_groups_on_modified_at, :index_projects_on_modified_at
    rename_index :groups, :index_groups_on_uuid, :index_projects_on_uuid
    rename_table :groups, :projects
  end
end
