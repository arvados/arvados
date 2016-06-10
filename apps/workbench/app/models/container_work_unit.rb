class ContainerWorkUnit < ProxyWorkUnit
  def children
    return self.my_children if self.my_children

    items = []

    crs = {}
    reqs = ContainerRequest.where(requesting_container_uuid: uuid).results
    reqs.each { |cr| crs[cr.container_uuid] = cr.name }

    containers = Container.where(uuid: crs.keys).results
    containers.each do |c|
      items << c.work_unit(crs[c.uuid])
    end

    self.my_children = items
  end

  def docker_image
    get(:container_image)
  end

  def runtime_constraints
    get(:runtime_constraints)
  end

  def priority
    get(:priority)
  end

  def log_collection
    get(:log)
  end

  def outputs
    items = []
    items << get(:output) if get(:output)
    items
  end

  def uri
    uuid = get(:uuid)
    "/containers/#{uuid}"
  end

  def title
    "container"
  end

  def can_cancel?
    true
  end
end
