class PipelineInstanceWorkUnit < ProxyWorkUnit
  def children
    return self.my_children if self.my_children

    items = []

    jobs = {}
    results = Job.where(uuid: self.proxied.job_ids.values).results
    results.each do |j|
      jobs[j.uuid] = j
    end

    components = get(:components)
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
        self.unreadable_children = true
        break
      end
    end

    self.my_children = items
  end

  def uri
    uuid = get(:uuid)
    "/pipeline_instances/#{uuid}"
  end

  def title
    "pipeline"
  end
end
