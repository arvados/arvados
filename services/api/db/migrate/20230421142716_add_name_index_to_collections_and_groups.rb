# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddNameIndexToCollectionsAndGroups < ActiveRecord::Migration[5.2]
  def up
    ActiveRecord::Base.connection.execute 'CREATE INDEX index_groups_on_name on groups USING gin (name gin_trgm_ops)'
    ActiveRecord::Base.connection.execute 'CREATE INDEX index_collections_on_name on collections USING gin (name gin_trgm_ops)'
  end
  def down
    ActiveRecord::Base.connection.execute 'DROP INDEX index_collections_on_name'
    ActiveRecord::Base.connection.execute 'DROP INDEX index_groups_on_name'
  end
end
