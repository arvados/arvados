class KeepDisksController < ApplicationController
  def create
    defaults = { is_readable: true, is_writable: true }
    @object = KeepDisk.new defaults.merge(params[:keep_disk] || {})
    super
  end

  def index
    # Retrieve cache age histogram info
    @cache_age_histogram = []
    @histogram_date = 0
    histogram_log = Log.
      filter([[:event_type, '=', 'block-age-free-space-histogram']]).
      order(:created_at => :desc).
      limit(1)
    histogram_log.each do |log_entry|
      # We expect this block to only execute at most once since we
      # specified limit(1)
      @cache_age_histogram = log_entry['properties'][:histogram]
      # Javascript wants dates in milliseconds.
      @histogram_date = log_entry['event_at'].to_i * 1000

      total_free_cache = @cache_age_histogram[-1][1]
      persisted_storage = 1 - total_free_cache
      @cache_age_histogram.map! { |x| {:age => @histogram_date - x[0]*1000,
          :cache => total_free_cache - x[1],
          :persisted => persisted_storage} }
    end

    # Do the regular control work needed.
    super
  end
end
