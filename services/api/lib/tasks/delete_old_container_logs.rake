# This task finds containers that have been finished for at least as long as
# the duration specified in the `clean_container_log_rows_after` config setting,
# and deletes their stdout, stderr, arv-mount, crunch-run, and  crunchstat logs
# from the logs table.

namespace :db do
  desc "Remove old container log entries from the logs table"
  task delete_old_container_logs: :environment do
    Log.select("logs.id").
        joins("JOIN containers ON object_uuid = containers.uuid").
        where("event_type in ('stdout', 'stderr', 'arv-mount', 'crunch-run', 'crunchstat') AND containers.log IS NOT NULL AND containers.finished_at < :age",
              age: Rails.configuration.clean_container_log_rows_after.ago).
        find_in_batches do |old_log_ids|
      Log.where(id: old_log_ids.map(&:id)).delete_all
    end
  end
end
