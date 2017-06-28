# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'rake'

Rake.application.rake_require "tasks/delete_old_container_logs"
Rake::Task.define_task(:environment)

class DeleteOldContainerLogsTaskTest < ActiveSupport::TestCase
  TASK_NAME = "db:delete_old_container_logs"

  def log_uuids(*fixture_names)
    fixture_names.map { |name| logs(name).uuid }
  end

  def run_with_expiry(clean_after)
    Rails.configuration.clean_container_log_rows_after = clean_after
    Rake::Task[TASK_NAME].reenable
    Rake.application.invoke_task TASK_NAME
  end

  def check_log_existence(test_method, fixture_uuids)
    uuids_now = Log.where("object_uuid LIKE :pattern AND event_type in ('stdout', 'stderr', 'arv-mount', 'crunch-run', 'crunchstat')", pattern: "%-dz642-%").map(&:uuid)
    fixture_uuids.each do |expect_uuid|
      send(test_method, uuids_now, expect_uuid)
    end
  end

  test "delete all finished logs" do
    uuids_to_keep = log_uuids(:stderr_for_running_container,
                              :crunchstat_for_running_container)
    uuids_to_clean = log_uuids(:stderr_for_previous_container,
                               :crunchstat_for_previous_container,
                               :stderr_for_ancient_container,
                               :crunchstat_for_ancient_container)
    run_with_expiry(1)
    check_log_existence(:assert_includes, uuids_to_keep)
    check_log_existence(:refute_includes, uuids_to_clean)
  end

  test "delete old finished logs" do
    uuids_to_keep = log_uuids(:stderr_for_running_container,
                              :crunchstat_for_running_container,
                              :stderr_for_previous_container,
                              :crunchstat_for_previous_container)
    uuids_to_clean = log_uuids(:stderr_for_ancient_container,
                               :crunchstat_for_ancient_container)
    run_with_expiry(360.days)
    check_log_existence(:assert_includes, uuids_to_keep)
    check_log_existence(:refute_includes, uuids_to_clean)
  end
end
