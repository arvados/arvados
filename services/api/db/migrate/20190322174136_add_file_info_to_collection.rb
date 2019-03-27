# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require "arvados/keep"

class AddFileInfoToCollection < ActiveRecord::Migration
  def do_batch(pdhs)
    pdhs_str = ''
    pdhs.each do |pdh|
      pdhs_str << "'" << pdh << "'" << ','
    end

    collections = ActiveRecord::Base.connection.exec_query(
      'SELECT DISTINCT portable_data_hash, manifest_text FROM collections '\
      "WHERE portable_data_hash IN (#{pdhs_str[0..-2]}) "
    )

    collections.rows.each do |row|
      manifest = Keep::Manifest.new(row[1])
      ActiveRecord::Base.connection.exec_query('BEGIN')
      ActiveRecord::Base.connection.exec_query("UPDATE collections SET file_count=#{manifest.files_count}, "\
                                               "file_size_total=#{manifest.files_size} "\
                                               "WHERE portable_data_hash='#{row[0]}'")
      ActiveRecord::Base.connection.exec_query('COMMIT')
    end
  end

  def up
    add_column :collections, :file_count, :integer, default: 0, null: false
    add_column :collections, :file_size_total, :integer, default: 0, null: false

    Container.group_pdhs_for_multiple_transactions('AddFileInfoToCollection') do |pdhs|
      do_batch(pdhs)
    end
  end

  def down
    remove_column :collections, :file_count
    remove_column :collections, :file_size_total
  end
end
