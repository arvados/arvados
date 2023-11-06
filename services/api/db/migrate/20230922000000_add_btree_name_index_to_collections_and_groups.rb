# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddBtreeNameIndexToCollectionsAndGroups < ActiveRecord::Migration[5.2]
  #
  # We previously added 'index_groups_on_name' and
  # 'index_collections_on_name' but those are 'gin_trgm_ops' which is
  # used with 'ilike' searches but despite documentation suggesting
  # they would be, experience has shown these indexes don't get used
  # for '=' (and/or they are much slower than the btree for exact
  # matches).
  #
  # So we want to add a regular btree index.
  #
  def up
    ActiveRecord::Base.connection.execute 'CREATE INDEX index_groups_on_name_btree on groups USING btree (name)'
    ActiveRecord::Base.connection.execute 'CREATE INDEX index_collections_on_name_btree on collections USING btree (name)'
  end
  def down
    ActiveRecord::Base.connection.execute 'DROP INDEX IF EXISTS index_collections_on_name_btree'
    ActiveRecord::Base.connection.execute 'DROP INDEX IF EXISTS index_groups_on_name_btree'
  end
end
