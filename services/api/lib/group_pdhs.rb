# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module GroupPdhs
  # NOTE: Migration 20190322174136_add_file_info_to_collection.rb relies on this function.
  #
  # Change with caution!
  #
  # Correctly groups pdhs to use for batch database updates. Helps avoid
  # updating too many database rows in a single transaction.
  def self.group_pdhs_for_multiple_transactions(distinct_ordered_pdhs, distinct_pdh_count, batch_size_max, log_prefix)
    batch_size = 0
    batch_pdhs = {}
    last_pdh = '0'
    done = 0
    any = true

    while any
      any = false
      distinct_ordered_pdhs.call(last_pdh) do |pdh|
        any = true
        last_pdh = pdh
        manifest_size = pdh.split('+')[1].to_i
        if batch_size > 0 && batch_size + manifest_size > batch_size_max
          yield batch_pdhs.keys
          done += batch_pdhs.size
          Rails.logger.info(log_prefix + ": #{done}/#{distinct_pdh_count}")
          batch_pdhs = {}
          batch_size = 0
        end
        batch_pdhs[pdh] = true
        batch_size += manifest_size
      end
    end
    yield batch_pdhs.keys
    Rails.logger.info(log_prefix + ": finished")
  end
end
