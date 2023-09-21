# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RecomputeFileNamesIndex < ActiveRecord::Migration[4.2]
  def do_batch(pdhs:)
    ActiveRecord::Base.connection.exec_query('BEGIN')
    Collection.select(:portable_data_hash, :manifest_text).where(portable_data_hash: pdhs).distinct(:portable_data_hash).each do |c|
      ActiveRecord::Base.connection.exec_query("update collections set file_names=$1 where portable_data_hash=$2",
                                               "update file_names index",
                                               [c.manifest_files, c.portable_data_hash])
    end
    ActiveRecord::Base.connection.exec_query('COMMIT')
  end
  def up
    # Process collections in multiple transactions, where the total
    # size of all manifest_texts processed in a transaction is no more
    # than batch_size_max.  Collections whose manifest_text is bigger
    # than batch_size_max are updated in their own individual
    # transactions.
    batch_size_max = 1 << 28    # 256 MiB
    batch_size = 0
    batch_pdhs = {}
    last_pdh = '0'
    total = Collection.distinct.count(:portable_data_hash)
    done = 0
    any = true
    while any
      any = false
      Collection.
        unscoped.
        select(:portable_data_hash).distinct.
        order(:portable_data_hash).
        where('portable_data_hash > ?', last_pdh).
        limit(1000).each do |c|
        any = true
        last_pdh = c.portable_data_hash
        manifest_size = c.portable_data_hash.split('+')[1].to_i
        if batch_size > 0 && batch_size + manifest_size > batch_size_max
          do_batch(pdhs: batch_pdhs.keys)
          done += batch_pdhs.size
          Rails.logger.info("RecomputeFileNamesIndex: #{done}/#{total}")
          batch_pdhs = {}
          batch_size = 0
        end
        batch_pdhs[c.portable_data_hash] = true
        batch_size += manifest_size
      end
    end
    do_batch(pdhs: batch_pdhs.keys)
    Rails.logger.info("RecomputeFileNamesIndex: finished")
  end
  def down
  end
end
