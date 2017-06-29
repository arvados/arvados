# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class NoDowngradeAfterJson < ActiveRecord::Migration
  def up
  end

  def down
    raise ActiveRecord::IrreversibleMigration.
      new("cannot downgrade: older versions cannot read JSON from DB tables")
  end
end
