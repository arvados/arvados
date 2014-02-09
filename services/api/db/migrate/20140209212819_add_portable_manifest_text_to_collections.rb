class AddPortableManifestTextToCollections < ActiveRecord::Migration
  def up
    add_column :collections, :portable_manifest_text, :text
    update_sql <<-EOS
UPDATE collections
 SET portable_manifest_text =
     regexp_replace(manifest_text,'\\+K@[a-z0-9]+', '', 'g')
 WHERE portable_manifest_text IS NULL
EOS
  end

  def down
    remove_column :collections, :portable_manifest_text
  end
end
