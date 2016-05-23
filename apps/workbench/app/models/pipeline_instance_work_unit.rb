class PipelineInstanceWorkUnit < ProxyWorkUnit
  def children
    items = []

    jobs = {}
    results = Job.where(uuid: self.proxied.job_ids.values).results
    results.each do |j|
      jobs[j.uuid] = j
    end

    components = (self.proxied.components if self.proxied.respond_to?(:components)) || self.proxied[:components]
    components.each do |name, c|
      if c.is_a?(Hash)
        job = c[:job]
        if job
          if job[:uuid] and jobs[job[:uuid]]
            items << jobs[job[:uuid]].work_unit(name)
          else
            items << JobWorkUnit.new(job, name)
          end
        else
          items << ProxyWorkUnit.new(c, name)
        end
      else
        break
      end
    end

    items
  end

  def progress
    state = (self.proxied.state if self.proxied.respond_to?(:state)) || self.proxied[:state]
    if state == 'Complete'
      return 1.0
    end

    done = 0
    failed = 0
    todo = 0
    children.each do |c|
      if c.success?.nil?
        todo = todo+1
      elsif c.success?
        done = done+1
      else
        failed = failed+1
      end
    end

    if done + failed + todo > 0
      total = done + failed + todo
      (done+failed).to_f / total
    else
      0.0
    end
  end

  def can_cancel?
    true
  end

  def uri
    uuid = (self.proxied.uuid if self.proxied.respond_to?(:uuid)) || self.proxied[:uuid]
    "/pipeline_instances/#{uuid}"
  end

  def title
    "pipeline"
  end
end
