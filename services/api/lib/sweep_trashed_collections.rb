require 'current_api_client'

module SweepTrashedCollections
  extend CurrentApiClient

  def self.sweep_now
    act_as_system_user do
      Collection.unscoped.
        where('delete_at is not null and delete_at < statement_timestamp()').
        destroy_all
      Collection.unscoped.
        where('is_trashed = false and trash_at < statement_timestamp()').
        update_all('is_trashed = true')
    end
  end

  def self.sweep_if_stale
    return if Rails.configuration.trash_sweep_interval <= 0
    exp = Rails.configuration.trash_sweep_interval.seconds
    need = false
    Rails.cache.fetch('SweepTrashedCollections', expires_in: exp) do
      need = true
    end
    if need
      Thread.new do
        Thread.current.abort_on_exception = false
        begin
          sweep_now
        rescue => e
          Rails.logger.error "#{e.class}: #{e}\n#{e.backtrace.join("\n\t")}"
        ensure
          ActiveRecord::Base.connection.close
        end
      end
    end
  end
end
