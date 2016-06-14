class ContainerWorkUnit < ProxyWorkUnit
  attr_accessor :related

  def initialize proxied, label
    super
    if @proxied.is_a?(ContainerRequest)
      container_uuid = get(:container_uuid)
      if container_uuid
        @related = Container.where(uuid: container_uuid).first rescue nil
      end
    end
  end

  def children
    return self.my_children if self.my_children

    items = []

    if @proxied.is_a?(Container)
      crs = {}
      reqs = ContainerRequest.where(requesting_container_uuid: uuid).results
      reqs.each { |cr| crs[cr.container_uuid] = cr.name }

      containers = Container.where(uuid: crs.keys).results
      containers.each do |c|
        items << c.work_unit(crs[c.uuid] || 'this container')
      end

      self.my_children = items
    else
      container_uuid = get(:container_uuid)
      if container_uuid
        reqs = ContainerRequest.where(requesting_container_uuid: container_uuid).results
        reqs.each do |cr|
          items << cr.work_unit(cr.name || 'this container')
        end
      end
    end

    self.my_children = items
  end

  def title
    "container"
  end

  def can_cancel?
    @proxied.is_a?(ContainerRequest) && state_label.in?(["Queued", "Locked", "Running"])
  end

  def container_uuid
    get(:container_uuid)
  end

  # For the following properties, use value from the @related container if exists
  # This applies to a ContainerRequest in Committed or Final state with container_uuid

  def started_at
    t = get_combined(:started_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def modified_at
    t = get_combined(:modified_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def finished_at
    t = get_combined(:finished_at)
    t = Time.parse(t) if (t.is_a? String)
    t
  end

  def state_label
    get_combined(:state)
  end

  def docker_image
    get_combined(:container_image)
  end

  def runtime_constraints
    get_combined(:runtime_constraints)
  end

  def priority
    get_combined(:priority)
  end

  def log_collection
    get_combined(:log)
  end

  def outputs
    items = []
    items << get_combined(:output) if get_combined(:output)
    items
  end

  def command
    get_combined(:command)
  end

  def cwd
    get_combined(:cwd)
  end

  def environment
    env = get_combined(:environment)
    env = nil if env.andand.empty?
    env
  end

  def mounts
    mnt = get_combined(:mounts)
    mnt = nil if mnt.andand.empty?
    mnt
  end

  def output_path
    get_combined(:output_path)
  end

  # End combined propeties

  protected
  def get_combined key
    get(key, @related) || get(key, @proxied)
  end
end
