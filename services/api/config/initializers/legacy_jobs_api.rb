# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Config must be done before we  files; otherwise they
# won't be able to use Rails.configuration.* to initialize their
# classes.
require_relative 'load_config.rb'

require 'enable_jobs_api'

Server::Application.configure do
  if ActiveRecord::Base.connection.tables.include?('jobs')
    check_enable_legacy_jobs_api
  end
end
