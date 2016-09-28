# This task finds jobs that have been finished for at least as long as
# the duration specified in the `clean_job_log_rows_after`
# configuration setting, and deletes their stderr logs from the logs table.

namespace :db do
  desc "Remove old job stderr entries from the logs table"
  task delete_old_job_logs: :environment do
    delete_sql = "DELETE FROM logs WHERE id in (SELECT logs.id FROM logs JOIN jobs ON logs.object_uuid = jobs.uuid WHERE event_type = 'stderr' AND jobs.log IS NOT NULL AND jobs.finished_at < '#{Rails.configuration.clean_job_log_rows_after.ago}')"

    ActiveRecord::Base.connection.execute(delete_sql)
  end
end
