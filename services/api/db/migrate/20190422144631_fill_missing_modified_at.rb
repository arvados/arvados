# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class FillMissingModifiedAt < ActiveRecord::Migration[5.0]
  def up
    Collection.where('modified_at is null').update_all('modified_at = created_at')
  end
  def down
  end
end
