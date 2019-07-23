# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

Disable_jobs_api_method_list = {"jobs.create"=>{},
                                "pipeline_instances.create"=>{},
                                "pipeline_templates.create"=>{},
                                "jobs.get"=>{},
                                "pipeline_instances.get"=>{},
                                "pipeline_templates.get"=>{},
                                "jobs.list"=>{},
                                "pipeline_instances.list"=>{},
                                "pipeline_templates.list"=>{},
                                "jobs.index"=>{},
                                "pipeline_instances.index"=>{},
                                "pipeline_templates.index"=>{},
                                "jobs.update"=>{},
                                "pipeline_instances.update"=>{},
                                "pipeline_templates.update"=>{},
                                "jobs.queue"=>{},
                                "jobs.queue_size"=>{},
                                "job_tasks.create"=>{},
                                "job_tasks.get"=>{},
                                "job_tasks.list"=>{},
                                "job_tasks.index"=>{},
                                "job_tasks.update"=>{},
                                "jobs.show"=>{},
                                "pipeline_instances.show"=>{},
                                "pipeline_templates.show"=>{},
                                "job_tasks.show"=>{}}

def check_enable_legacy_jobs_api
  if Rails.configuration.Containers.JobsAPI.Enable == "false" ||
     (Rails.configuration.Containers.JobsAPI.Enable == "auto" &&
      Job.count == 0)
    Rails.configuration.API.DisabledAPIs.merge! Disable_jobs_api_method_list
  end
end
