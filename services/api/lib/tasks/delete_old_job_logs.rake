# This task finds jobs that have been finished for at least as long as
# the duration specified in the `clean_job_log_rows_after`
# configuration setting, and deletes their stderr logs from the logs table.

namespace :db do
  desc "Remove old job stderr entries from the logs table"
  task delete_old_job_logs: :environment do
    Log.select("logs.id").
        joins("JOIN jobs ON object_uuid = jobs.uuid").
        where("event_type = :etype AND jobs.log IS NOT NULL AND jobs.finished_at < :age",
              etype: "stderr",
              age: Rails.configuration.clean_job_log_rows_after.ago).
        find_in_batches do |old_log_ids|
      Log.where(id: old_log_ids.map(&:id)).delete_all
    end
  end
end
