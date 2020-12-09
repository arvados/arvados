# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'fix_collection_versions_timestamps'

class FixCollectionVersionsTimestamps < ActiveRecord::Migration[5.2]
  def up
    # Defined in a function for easy testing.
    fix_collection_versions_timestamps
  end

  def down
    # This migration is not reversible.  However, the results are
    # backwards compatible.
  end
end
