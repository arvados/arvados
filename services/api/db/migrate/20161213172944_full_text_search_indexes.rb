class FullTextSearchIndexes < ActiveRecord::Migration
  def fts_indexes
    {
      "collections" => "collections_full_text_search_idx",
      "container_requests" => "container_requests_full_text_search_idx",
      "groups" => "groups_full_text_search_idx",
      "jobs" => "jobs_full_text_search_idx",
      "pipeline_instances" => "pipeline_instances_full_text_search_idx",
      "pipeline_templates" => "pipeline_templates_full_text_search_idx",
      "workflows" => "workflows_full_text_search_idx",
    }
  end

  def up
    # remove existing fts indexes and create up to date ones with no leading space
    fts_indexes.each do |t, i|
      t.classify.constantize.reset_column_information
      ActiveRecord::Base.connection.indexes(t).each do |idx|
        if idx.name == i
          remove_index t.to_sym, :name => i
          break
        end
      end
      execute "CREATE INDEX #{i} ON #{t} USING gin(#{t.classify.constantize.full_text_tsvector});"
    end
  end

  def down
    fts_indexes.each do |t, i|
      remove_index t.to_sym, :name => i
    end
  end
end
