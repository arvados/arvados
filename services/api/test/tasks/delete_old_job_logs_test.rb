require 'test_helper'
require 'rake'

Rake.application.rake_require "tasks/delete_old_job_logs"
Rake::Task.define_task(:environment)

class DeleteOldJobLogsTaskTest < ActiveSupport::TestCase
  TASK_NAME = "db:delete_old_job_logs"

  def log_uuids(*fixture_names)
    fixture_names.map { |name| logs(name).uuid }
  end

  def run_with_expiry(clean_after)
    Rails.configuration.clean_job_log_rows_after = clean_after
    Rake::Task[TASK_NAME].reenable
    Rake.application.invoke_task TASK_NAME
  end

  def job_stderr_logs
    Log.where("object_uuid LIKE :pattern AND event_type = :etype",
              pattern: "_____-8i9sb-_______________",
              etype: "stderr")
  end

  def check_existence(test_method, fixture_uuids)
    uuids_now = job_stderr_logs.map(&:uuid)
    fixture_uuids.each do |expect_uuid|
      send(test_method, uuids_now, expect_uuid)
    end
  end

  test "delete all logs" do
    uuids_to_keep = log_uuids(:crunchstat_for_running_job)
    uuids_to_clean = log_uuids(:crunchstat_for_previous_job,
                               :crunchstat_for_ancient_job)
    run_with_expiry(1)
    check_existence(:assert_includes, uuids_to_keep)
    check_existence(:refute_includes, uuids_to_clean)
  end

  test "delete only old logs" do
    uuids_to_keep = log_uuids(:crunchstat_for_running_job,
                              :crunchstat_for_previous_job)
    uuids_to_clean = log_uuids(:crunchstat_for_ancient_job)
    run_with_expiry(360.days)
    check_existence(:assert_includes, uuids_to_keep)
    check_existence(:refute_includes, uuids_to_clean)
  end
end
