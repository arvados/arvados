class JobWorkUnit < ProxyWorkUnit
  def children
    # Job tasks
    uuid = (self.proxied.uuid if self.proxied.respond_to?(:uuid)) || self.proxied[:uuid]
    tasks = JobTask.filter([['job_uuid', '=', uuid]]).results
    items = []
    tasks.each do |t|
      items << t.work_unit("task #{items.size}")
    end

    # Jobs submitted by this job  --  TBD

    items
  end

  def progress
    state = (self.proxied.state if self.proxied.respond_to?(:state)) || self.proxied[:state]
    if state == 'Complete'
      return 1.0
    end

    tasks_summary = (self.proxied.tasks_summary if self.proxied.respond_to?(:tasks_summary)) || self.proxied[:tasks_summary]
    failed = tasks_summary[:failed] || 0 rescue 0
    done = tasks_summary[:done] || 0 rescue 0
    running = tasks_summary[:running] || 0 rescue 0
    todo = tasks_summary[:todo] || 0 rescue 0
    if done + running + failed + todo > 0
      total_tasks = done + running + failed + todo
      (done+failed).to_f / total_tasks
    else
      0.0
    end
  end

  def docker_image
    (self.proxied.docker_image_locator if self.proxied.respond_to?(:docker_image_locator)) || self.proxied[:docker_image_locator]
  end

  def nondeterministic
    (self.proxied.nondeterministic if self.proxied.respond_to?(:nondeterministic)) || self.proxied[:nondeterministic]
  end

  def priority
    (self.proxied.priority if self.proxied.respond_to?(:priority)) || self.proxied[:priority]
  end

  def log_collection
    (self.proxied.log if self.proxied.respond_to?(:log)) || self.proxied[:log]
  end

  def output
    (self.proxied.output if self.proxied.respond_to?(:output)) || self.proxied[:output]
  end

  def uri
    uuid = (self.proxied.uuid if self.proxied.respond_to?(:uuid)) || self.proxied[:uuid]
    "/jobs/#{uuid}"
  end

  def child_summary
    (self.proxied.tasks_summary if self.proxied.respond_to?(:tasks_summary)) || self.proxied[:tasks_summary]
  end

  def title
    "job"
  end
end
