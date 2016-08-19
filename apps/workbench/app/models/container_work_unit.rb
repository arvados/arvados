class ContainerWorkUnit < ProxyWorkUnit
  attr_accessor :container

  def initialize proxied, label
    super
    if @proxied.is_a?(ContainerRequest)
      container_uuid = get(:container_uuid)
      if container_uuid
        @container = Container.where(uuid: container_uuid).first
      end
    end
  end

  def children
    return self.my_children if self.my_children

    container_uuid = nil
    container_uuid = if @proxied.is_a?(Container) then uuid else get(:container_uuid) end

    items = []
    if container_uuid
      reqs = ContainerRequest.where(requesting_container_uuid: container_uuid).results
      reqs.each do |cr|
        items << cr.work_unit(cr.name || 'this container')
      end
    end

    self.my_children = items
  end

  def title
    "container"
  end

  def uri
    uuid = get(:uuid)

    return nil unless uuid

    if @proxied.class.respond_to? :table_name
      "/#{@proxied.class.table_name}/#{uuid}"
    else
      resource_class = ArvadosBase.resource_class_for_uuid(uuid)
      "#{resource_class.table_name}/#{uuid}" if resource_class
    end
  end

  def can_cancel?
    @proxied.is_a?(ContainerRequest) && state_label.in?(["Queued", "Locked", "Running"]) && priority > 0
  end

  def container_uuid
    get(:container_uuid)
  end

  # For the following properties, use value from the @container if exists
  # This applies to a ContainerRequest with container_uuid

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

  def log_object_uuids
    [get_combined(:uuid), get(:uuid)].uniq
  end

  def live_log_lines(limit=2000)
    event_types = ["stdout", "stderr", "arv-mount", "crunch-run"]
    log_lines = Log.where(event_type: event_types, object_uuid: log_object_uuids).order("id DESC").limit(limit)
    log_lines.results.reverse.
      flat_map { |log| log.properties[:text].split("\n") rescue [] }
  end

  def render_log
    collection = Collection.find(log_collection) rescue nil
    if collection
      return {log: collection, partial: 'collections/show_files', locals: {object: collection, no_checkboxes: true}}
    end
  end

  def template_uuid
    properties = get(:properties)
    if properties
      properties[:workflow_uuid]
    end
  end

  # End combined propeties

  protected
  def get_combined key
    get(key, @container) || get(key, @proxied)
  end
end
