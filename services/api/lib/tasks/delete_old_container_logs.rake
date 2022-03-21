# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This task finds containers that have been finished for at least as long as
# the duration specified in the `clean_container_log_rows_after` config setting,
# and deletes their stdout, stderr, arv-mount, crunch-run, and  crunchstat logs
# from the logs table.

namespace :db do
  desc "Remove old container log entries from the logs table"

  task delete_old_container_logs: :environment do
    delete_sql = "DELETE FROM logs WHERE id in (SELECT logs.id FROM logs JOIN containers ON logs.object_uuid = containers.uuid WHERE event_type IN ('stdout', 'stderr', 'arv-mount', 'crunch-run', 'crunchstat') AND containers.log IS NOT NULL AND now() - containers.finished_at > interval '#{Rails.configuration.Containers.Logging.MaxAge.to_i} seconds')"

    ActiveRecord::Base.connection.execute(delete_sql)
  end
end
