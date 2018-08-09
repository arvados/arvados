# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class IndexAllFilenames < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN file_names TYPE text'
    Collection.find_each(batch_size: 20) do |c|
      ActiveRecord::Base.connection.execute "UPDATE collections
                    SET file_names = #{ActiveRecord::Base.connection.quote(c.manifest_files)}
                    WHERE uuid = #{ActiveRecord::Base.connection.quote(c.uuid)}
                    AND portable_data_hash = #{ActiveRecord::Base.connection.quote(c.portable_data_hash)}"
    end
  end
  def down
    ActiveRecord::Base.connection.execute 'ALTER TABLE collections ALTER COLUMN file_names TYPE varchar(8192)'
  end
end
