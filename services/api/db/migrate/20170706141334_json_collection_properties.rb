# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require './db/migrate/20161213172944_full_text_search_indexes'

class JsonCollectionProperties < ActiveRecord::Migration
  def up
    # Drop the FT index before changing column type to avoid
    # "PG::DatatypeMismatch: ERROR: COALESCE types jsonb and text
    # cannot be matched".
    ActiveRecord::Base.connection.execute 'DROP INDEX IF EXISTS collections_full_text_search_idx'
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN properties TYPE jsonb USING properties::jsonb'
    FullTextSearchIndexes.new.replace_index('collections')
  end

  def down
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN properties TYPE text'
  end
end
