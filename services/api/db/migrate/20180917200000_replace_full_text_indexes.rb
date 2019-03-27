# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require './db/migrate/20161213172944_full_text_search_indexes'

class ReplaceFullTextIndexes < ActiveRecord::Migration[4.2]
  def up
    FullTextSearchIndexes.new.up
  end

  def down
  end
end
