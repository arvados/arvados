# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class EmptyCollection < ActiveRecord::Migration
  include CurrentApiClient

  def up
    empty_collection
  end

  def down
    # do nothing when migrating down (having the empty collection
    # and a permission link for it is harmless)
  end
end
