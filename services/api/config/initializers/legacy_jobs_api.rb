# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Config must be done before we  files; otherwise they
# won't be able to use Rails.configuration.* to initialize their
# classes.

require 'enable_jobs_api'

Rails.application.configure do
  begin
    if ENV["ARVADOS_CONFIG"] != "none" && ActiveRecord::Base.connection.tables.include?('jobs')
      check_enable_legacy_jobs_api
    end
  rescue ActiveRecord::NoDatabaseError
    # Since rails 5.2, all initializers are run by rake tasks (like db:create),
    # see: https://github.com/rails/rails/issues/32870
  end
end
