# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenamePipelinesToPipelineTemplates < ActiveRecord::Migration
  def up
    rename_column :pipeline_instances, :pipeline_uuid, :pipeline_template_uuid
    rename_table :pipelines, :pipeline_templates
    rename_index :pipeline_templates, :index_pipelines_on_created_at, :index_pipeline_templates_on_created_at
    rename_index :pipeline_templates, :index_pipelines_on_modified_at, :index_pipeline_templates_on_modified_at
    rename_index :pipeline_templates, :index_pipelines_on_uuid, :index_pipeline_templates_on_uuid
    Link.update_all({head_kind:'orvos#pipeline'}, ['head_kind=?','orvos#pipeline_template'])
    Link.update_all({tail_kind:'orvos#pipeline'}, ['tail_kind=?','orvos#pipeline_template'])
    Log.update_all({object_kind:'orvos#pipeline'}, ['object_kind=?','orvos#pipeline_template'])
  end

  def down
    Link.update_all({head_kind:'orvos#pipeline_template'}, ['head_kind=?','orvos#pipeline'])
    Link.update_all({tail_kind:'orvos#pipeline_template'}, ['tail_kind=?','orvos#pipeline'])
    Log.update_all({object_kind:'orvos#pipeline_template'}, ['object_kind=?','orvos#pipeline'])
    rename_index :pipeline_templates, :index_pipeline_templates_on_created_at, :index_pipelines_on_created_at
    rename_index :pipeline_templates, :index_pipeline_templates_on_modified_at, :index_pipelines_on_modified_at
    rename_index :pipeline_templates, :index_pipeline_templates_on_uuid, :index_pipelines_on_uuid
    rename_table :pipeline_templates, :pipelines
    rename_column :pipeline_instances, :pipeline_template_uuid, :pipeline_uuid
  end
end
