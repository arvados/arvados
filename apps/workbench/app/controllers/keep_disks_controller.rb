# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class KeepDisksController < ApplicationController
  def create
    defaults = { is_readable: true, is_writable: true }
    @object = KeepDisk.new defaults.merge(params[:keep_disk] || {})
    super
  end

  def index
    # Retrieve cache age histogram info from logs.

    # In the logs we expect to find it in an ordered list with entries
    # of the form (mtime, disk proportion free).

    # An entry of the form (1388747781, 0.52) means that if we deleted
    # the oldest non-presisted blocks until we had 52% of the disk
    # free, then all blocks with an mtime greater than 1388747781
    # would be preserved.

    # The chart we want to produce, will tell us how much of the disk
    # will be free if we use a cache age of x days. Therefore we will
    # produce output specifying the age, cache and persisted. age is
    # specified in milliseconds. cache is the size of the cache if we
    # delete all blocks older than age. persistent is the size of the
    # persisted blocks. It is constant regardless of age, but it lets
    # us show a stacked graph.

    # Finally each entry in cache_age_histogram is a dictionary,
    # because that's what our charting package wats.

    @cache_age_histogram = []
    @histogram_pretty_date = nil
    histogram_log = Log.
      filter([[:event_type, '=', 'block-age-free-space-histogram']]).
      order(:created_at => :desc).
      with_count('none').
      limit(1)
    histogram_log.each do |log_entry|
      # We expect this block to only execute at most once since we
      # specified limit(1)
      @cache_age_histogram = log_entry['properties'][:histogram]
      # Javascript wants dates in milliseconds.
      histogram_date_ms = log_entry['event_at'].to_i * 1000
      @histogram_pretty_date = log_entry['event_at'].strftime('%b %-d, %Y')

      total_free_cache = @cache_age_histogram[-1][1]
      persisted_storage = 1 - total_free_cache
      @cache_age_histogram.map! { |x| {:age => histogram_date_ms - x[0]*1000,
          :cache => total_free_cache - x[1],
          :persisted => persisted_storage} }
    end

    # Do the regular control work needed.
    super
  end
end
