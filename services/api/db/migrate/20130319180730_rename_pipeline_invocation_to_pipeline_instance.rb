# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenamePipelineInvocationToPipelineInstance < ActiveRecord::Migration
  def up
    rename_table :pipeline_invocations, :pipeline_instances
    rename_index :pipeline_instances, :index_pipeline_invocations_on_created_at, :index_pipeline_instances_on_created_at
    rename_index :pipeline_instances, :index_pipeline_invocations_on_modified_at, :index_pipeline_instances_on_modified_at
    rename_index :pipeline_instances, :index_pipeline_invocations_on_uuid, :index_pipeline_instances_on_uuid
    Link.update_all({head_kind:'orvos#pipeline_instance'}, ['head_kind=?','orvos#pipeline_invocation'])
    Link.update_all({tail_kind:'orvos#pipeline_instance'}, ['tail_kind=?','orvos#pipeline_invocation'])
    Log.update_all({object_kind:'orvos#pipeline_instance'}, ['object_kind=?','orvos#pipeline_invocation'])
  end

  def down
    Link.update_all({head_kind:'orvos#pipeline_invocation'}, ['head_kind=?','orvos#pipeline_instance'])
    Link.update_all({tail_kind:'orvos#pipeline_invocation'}, ['tail_kind=?','orvos#pipeline_instance'])
    Log.update_all({object_kind:'orvos#pipeline_invocation'}, ['object_kind=?','orvos#pipeline_instance'])
    rename_index :pipeline_instances, :index_pipeline_instances_on_created_at, :index_pipeline_invocations_on_created_at
    rename_index :pipeline_instances, :index_pipeline_instances_on_modified_at, :index_pipeline_invocations_on_modified_at
    rename_index :pipeline_instances, :index_pipeline_instances_on_uuid, :index_pipeline_invocations_on_uuid
    rename_table :pipeline_instances, :pipeline_invocations
  end
end
