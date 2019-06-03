# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddFileInfoToCollection < ActiveRecord::Migration[4.2]
  def up
    add_column :collections, :file_count, :integer, default: 0, null: false
    add_column :collections, :file_size_total, :integer, limit: 8, default: 0, null: false

    puts "Collections now have two new columns, file_count and file_size_total."
    puts "They were initialized with a zero value. If you are upgrading an Arvados"
    puts "installation, please run the populate-file-info-columns-in-collections.rb"
    puts "script to populate the columns. If this is a new installation, that is not"
    puts "necessary."
  end

  def down
    remove_column :collections, :file_count
    remove_column :collections, :file_size_total
  end
end
