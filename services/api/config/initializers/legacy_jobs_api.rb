# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Config must be done before we  files; otherwise they
# won't be able to use Rails.configuration.* to initialize their
# classes.
require_relative 'load_config.rb'

Server::Application.configure do
  if Rails.configuration.enable_legacy_jobs_api == false ||
     (Rails.configuration.enable_legacy_jobs_api == "auto" &&
      ActiveRecord::Base.connection.exec_query("select count(*) from jobs")[0] == 0)
    Rails.configuration.disable_api_methods = ["jobs.create",
                                               "pipeline_instances.create",
                                               "pipeline_templates.create",
                                               "jobs.get",
                                               "pipeline_instances.get",
                                               "pipeline_templates.get",
                                               "jobs.list",
                                               "pipeline_instances.list",
                                               "pipeline_templates.list",
                                               "jobs.index",
                                               "pipeline_instances.index",
                                               "pipeline_templates.index",
                                               "jobs.update",
                                               "pipeline_instances.update",
                                               "pipeline_templates.update",
                                               "jobs.queue",
                                               "jobs.queue_size",
                                               "job_tasks.create",
                                               "job_tasks.get",
                                               "job_tasks.list",
                                               "job_tasks.index",
                                               "job_tasks.update",
                                               "jobs.show",
                                               "pipeline_instances.show",
                                               "pipeline_templates.show",
                                               "jobs.show",
                                               "job_tasks.show"]
  end
end
