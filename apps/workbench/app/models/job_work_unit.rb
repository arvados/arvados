class JobWorkUnit < ProxyWorkUnit
  def children
    # Job tasks
    tasks = JobTask.filter([['job_uuid', '=', self.proxied.uuid]]).results
    items = []
    tasks.each do |t|
      items << t.work_unit("task #{items.size}")
    end

    # Jobs submitted by this job  --  TBD

    items
  end

  def progress
    if self.proxied.state == 'Complete'
      return 1.0
    end

    failed = self.proxied.tasks_summary[:failed] || 0 rescue 0
    done = self.proxied.tasks_summary[:done] || 0 rescue 0
    running = self.proxied.tasks_summary[:running] || 0 rescue 0
    todo = self.proxied.tasks_summary[:todo] || 0 rescue 0
    if done + running + failed + todo > 0
      total_tasks = done + running + failed + todo
      (done+failed).to_f / total_tasks
    else
      0.0
    end
  end

  def docker_image
    self.proxied[:docker_image_locator]
  end

  def nondeterministic
    self.proxied[:nondeterministic]
  end

  def priority
    self.proxied[:priority]
  end

  def log_collection
    [self.proxied.log]
  end

  def output
    self.proxied.output
  end
end
