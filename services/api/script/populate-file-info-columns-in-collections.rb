#!/usr/bin/env ruby
# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Arvados version 1.4.0 introduces two new columns on the collections table named
#   file_count
#   file_size_total
#
# The database migration that adds these columns does not populate them with data,
# it initializes them set to zero.
#
# This script will populate the columns, if file_count is zero. It will ignore
# collections that have invalid manifests, but it will spit out details for those
# collections.
#
# Run the script as
#
# cd scripts
# RAILS_ENV=production bundle exec populate-file-info-columns-in-collections.rb
#

ENV["RAILS_ENV"] = ARGV[0] || ENV["RAILS_ENV"] || "development"
require File.dirname(__FILE__) + '/../config/boot'
require File.dirname(__FILE__) + '/../config/environment'

require "arvados/keep"
require "group_pdhs"

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
      begin
        manifest = Keep::Manifest.new(row[1])
        ActiveRecord::Base.connection.exec_query("BEGIN")
        ActiveRecord::Base.connection.exec_query("UPDATE collections SET file_count=#{manifest.files_count}, "\
                                                 "file_size_total=#{manifest.files_size} "\
                                                 "WHERE portable_data_hash='#{row[0]}'")
        ActiveRecord::Base.connection.exec_query("COMMIT")
      rescue ArgumentError => detail
        require 'pp'
        puts
        puts "*************** Row detail ***************"
        puts
        pp row
        puts
        puts "************ Collection detail ***********"
        puts
        pp Collection.find_by_portable_data_hash(row[0])
        puts
        puts "************** Error detail **************"
        puts
        pp detail
        puts
        puts "Skipping this collection, continuing!"
        next
      end
    end
  end


def main

  distinct_pdh_count = ActiveRecord::Base.connection.exec_query(
    "SELECT DISTINCT portable_data_hash FROM collections where file_count=0"
  ).rows.count

  # Generator that queries for all the distinct pdhs greater than last_pdh
  ordered_pdh_query = lambda { |last_pdh, &block|
    pdhs = ActiveRecord::Base.connection.exec_query(
      "SELECT DISTINCT portable_data_hash FROM collections "\
      "WHERE file_count=0 and portable_data_hash > '#{last_pdh}' "\
      "ORDER BY portable_data_hash LIMIT 1000"
    )
    pdhs.rows.each do |row|
      block.call(row[0])
    end
  }

  batch_size_max = 1 << 28 # 256 MiB
  GroupPdhs.group_pdhs_for_multiple_transactions(ordered_pdh_query,
                                                 distinct_pdh_count,
                                                 batch_size_max,
                                                 "AddFileInfoToCollection") do |pdhs|
    do_batch(pdhs)
  end
end

main
