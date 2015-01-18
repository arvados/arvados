class ManifestAndComponentsFullTextSearch < ActiveRecord::Migration

  def up
    execute "CREATE INDEX collections_manifest_full_text_search_idx ON collections USING gin(to_tsvector('english', translate(manifest_text, '/.', '  ')));"
    execute "CREATE INDEX pipeline_instances_components_full_text_search_idx ON pipeline_instances USING gin(to_tsvector('english', translate(components, '/.', '  ')));"
  end

  def down
    remove_index :pipeline_instances, :name => 'pipeline_instances_components_full_text_search_idx'
    remove_index :collections, :name => 'collections_manifest_full_text_search_idx'
  end
end
