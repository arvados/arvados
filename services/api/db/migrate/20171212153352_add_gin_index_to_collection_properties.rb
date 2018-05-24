# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddGinIndexToCollectionProperties < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute("CREATE INDEX collection_index_on_properties ON collections USING gin (properties);")
  end
  def down
    ActiveRecord::Base.connection.execute("DROP INDEX collection_index_on_properties")
  end
end
