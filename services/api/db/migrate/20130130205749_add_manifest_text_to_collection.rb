class AddManifestTextToCollection < ActiveRecord::Migration
  def change
    add_column :collections, :manifest_text, :text
  end
end
