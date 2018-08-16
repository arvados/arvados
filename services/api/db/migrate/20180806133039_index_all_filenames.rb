# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class IndexAllFilenames < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN file_names TYPE text'
  end
  def down
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN file_names TYPE varchar(8192)'
  end
end
