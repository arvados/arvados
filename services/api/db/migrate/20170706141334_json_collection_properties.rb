require './db/migrate/20161213172944_full_text_search_indexes'

class JsonCollectionProperties < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute 'DROP INDEX IF EXISTS collections_full_text_search_idx'
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN properties TYPE jsonb USING properties::jsonb'
    FullTextSearchIndexes.new.replace_index('collections')
  end

  def down
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN properties TYPE text'
  end
end
