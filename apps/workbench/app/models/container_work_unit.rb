class ContainerWorkUnit < ProxyWorkUnit
  attr_accessor :container

  def initialize proxied, label, parent
    super
    if @proxied.is_a?(ContainerRequest)
      container_uuid = get(:container_uuid)
      if container_uuid
        @container = Container.where(uuid: container_uuid).first
      end
    end
  end

  def children
    return @my_children if @my_children

    container_uuid = nil
    container_uuid = if @proxied.is_a?(Container) then uuid else get(:container_uuid) end

    items = []
    if container_uuid
      reqs = ContainerRequest.where(requesting_container_uuid: container_uuid).results
      reqs.each do |cr|
        items << cr.work_unit(cr.name || 'this container')
      end
    end

    @my_children = items
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
    @proxied.is_a?(ContainerRequest) && @proxied.state == "Committed" && @proxied.priority > 0 && @proxied.editable?
  end

  def container_uuid
    get(:container_uuid)
  end

  def priority
    @proxied.priority
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
    ec = exit_code
    return "Failed" if (ec && ec != 0)
    state = get_combined(:state)
    return "Ready" if ((priority == 0) and (["Queued", "Locked"].include?(state)))
    state
  end

  def exit_code
    get_combined(:exit_code)
  end

  def docker_image
    get_combined(:container_image)
  end

  def runtime_constraints
    get_combined(:runtime_constraints)
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
    [get(:uuid, @container), get(:uuid, @proxied)].compact
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
      properties[:template_uuid]
    end
  end

  # End combined properties

  protected
  def get_combined key
    get(key, @container) || get(key, @proxied)
  end
end
