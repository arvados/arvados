# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddFileInfoToCollection < ActiveRecord::Migration
  def do_batch(pdhs)
    pdhs_str = ''
    pdhs.each do |pdh|
      pdhs_str << "'" << pdh[0] << "'" << ','
    end

    collections = ActiveRecord::Base.connection.exec_query(
      'SELECT DISTINCT portable_data_hash, manifest_text FROM collections '\
      "WHERE portable_data_hash IN (#{pdhs_str[0..-2]}) "
    )

    collections.rows.each do |row|
      file_count = 0
      file_size_total = 0
      row[1].scan(/\S+/) do |token|
        is_file = token.match(/^[[:digit:]]+:[[:digit:]]+:([^\000-\040\\]|\\[0-3][0-7][0-7])+$/)
        if is_file
          _, filesize, filename = token.split(':', 3)

          # Avoid counting empty dir placeholders
          break if filename == '.' && filesize.zero?

          file_count += 1
          file_size_total += filesize.to_i
        end
      end
      ActiveRecord::Base.connection.exec_query('BEGIN')
      ActiveRecord::Base.connection.exec_query("UPDATE collections SET file_count=#{file_count}, "\
                                               "file_size_total=#{file_size_total} "\
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
