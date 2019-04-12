# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddPropertiesToSpecimen < ActiveRecord::Migration[4.2]
  def change
    add_column :specimens, :properties, :text
  end
end
