# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class PipelineInstanceWorkUnit < ProxyWorkUnit
  def children
    return @my_children if @my_children

    items = []

    jobs = {}
    results = Job.where(uuid: @proxied.job_ids.values).results
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
            items << JobWorkUnit.new(job, name, uuid)
          end
        else
          items << JobWorkUnit.new(c, name, uuid)
        end
      else
        @unreadable_children = true
        break
      end
    end

    @my_children = items
  end

  def outputs
    items = []
    components = get(:components)
    components.each do |name, c|
      if c.is_a?(Hash)
        items << c[:output_uuid] if c[:output_uuid]
      end
    end
    items
  end

  def uri
    uuid = get(:uuid)
    "/pipeline_instances/#{uuid}"
  end

  def title
    "pipeline"
  end

  def template_uuid
    get(:pipeline_template_uuid)
  end

  def state_label
    if get(:state) != "Failed"
      return super
    end
    if get(:components_summary).andand[:failed].andand > 0
      return super
    end
    # Show "Cancelled" instead of "Failed" if there are no failed
    # components. #12840
    get(:components).each do |_, c|
      jstate = c[:job][:state] rescue nil
      if jstate == "Failed"
        return "Failed"
      end
    end
    "Cancelled"
  end
end
