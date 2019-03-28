# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require "arvados/keep"

class AddFileInfoToCollection < ActiveRecord::Migration
  def do_batch(pdhs)
    pdhs_str = ''
    pdhs.each do |pdh|
      pdhs_str << "'" << pdh << "'" << ","
    end

    collections = ActiveRecord::Base.connection.exec_query(
      "SELECT DISTINCT portable_data_hash, manifest_text FROM collections "\
      "WHERE portable_data_hash IN (#{pdhs_str[0..-2]}) "
    )

    collections.rows.each do |row|
      manifest = Keep::Manifest.new(row[1])
      ActiveRecord::Base.connection.exec_query("BEGIN")
      ActiveRecord::Base.connection.exec_query("UPDATE collections SET file_count=#{manifest.files_count}, "\
                                               "file_size_total=#{manifest.files_size} "\
                                               "WHERE portable_data_hash='#{row[0]}'")
      ActiveRecord::Base.connection.exec_query("COMMIT")
    end
  end

  def up
    add_column :collections, :file_count, :integer, default: 0, null: false
    add_column :collections, :file_size_total, :integer, limit: 8, default: 0, null: false

    distinct_pdh_count = ActiveRecord::Base.connection.exec_query(
      "SELECT DISTINCT portable_data_hash FROM collections"
    ).rows.count

    # Generator that queries for all the distince pdhs greater than last_pdh
    ordered_pdh_query = lambda { |last_pdh, &block|
      pdhs = ActiveRecord::Base.connection.exec_query(
        "SELECT DISTINCT portable_data_hash FROM collections "\
        "WHERE portable_data_hash > '#{last_pdh}' "\
        "ORDER BY portable_data_hash LIMIT 1000"
      )
      pdhs.rows.each do |row|
        block.call(row[0])
      end
    }

    batch_size_max = 1 << 28 # 256 MiB
    Container.group_pdhs_for_multiple_transactions(ordered_pdh_query,
                                                   distinct_pdh_count,
                                                   batch_size_max,
                                                   "AddFileInfoToCollection") do |pdhs|
      do_batch(pdhs)
    end
  end

  def down
    remove_column :collections, :file_count
    remove_column :collections, :file_size_total
  end
end
