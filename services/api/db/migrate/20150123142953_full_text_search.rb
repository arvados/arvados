class FullTextSearch < ActiveRecord::Migration

  def up
    execute "CREATE INDEX collections_full_text_search_idx ON collections USING gin(#{Collection.full_text_tsvector});"
    execute "CREATE INDEX groups_full_text_search_idx ON groups USING gin(#{Group.full_text_tsvector});"
    execute "CREATE INDEX jobs_full_text_search_idx ON jobs USING gin(#{Job.full_text_tsvector});"
    execute "CREATE INDEX pipeline_instances_full_text_search_idx ON pipeline_instances USING gin(#{PipelineInstance.full_text_tsvector});"
    execute "CREATE INDEX pipeline_templates_full_text_search_idx ON pipeline_templates USING gin(#{PipelineTemplate.full_text_tsvector});"
  end

  def down
    remove_index :pipeline_templates, :name => 'pipeline_templates_full_text_search_idx'
    remove_index :pipeline_instances, :name => 'pipeline_instances_full_text_search_idx'
    remove_index :jobs, :name => 'jobs_full_text_search_idx'
    remove_index :groups, :name => 'groups_full_text_search_idx'
    remove_index :collections, :name => 'collections_full_text_search_idx'
  end
end
