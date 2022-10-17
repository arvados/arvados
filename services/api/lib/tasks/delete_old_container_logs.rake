# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# This task finds containers that have been finished for at least as long as
# the duration specified in the `clean_container_log_rows_after` config setting,
# and deletes their stdout, stderr, arv-mount, crunch-run, and  crunchstat logs
# from the logs table.

namespace :db do
  desc "deprecated / no-op"

  task delete_old_container_logs: :environment do
    Rails.logger.info "this db:delete_old_container_logs rake task is no longer used"
  end
end
