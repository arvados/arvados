# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ContainerWorkUnit < ProxyWorkUnit
  attr_accessor :container
  attr_accessor :child_proxies

  def initialize proxied, label, parent, child_objects=nil
    super proxied, label, parent
    if @proxied.is_a?(ContainerRequest)
      container_uuid = get(:container_uuid)
      if container_uuid
        @container = Container.find(container_uuid)
      end
    end
    @child_proxies = child_objects
  end

  def children
    return @my_children if @my_children

    items = []
    container_uuid = if @proxied.is_a?(Container) then uuid else get(:container_uuid) end
    if container_uuid
      cols = ContainerRequest.columns.map(&:name) - %w(id updated_at mounts secret_mounts)
      my_children = @child_proxies || ContainerRequest.select(cols).where(requesting_container_uuid: container_uuid).results if !my_children
      my_child_containers = my_children.map(&:container_uuid).compact.uniq
      grandchildren = {}
      my_child_containers.each { |c| grandchildren[c] = []} if my_child_containers.any?
      reqs = ContainerRequest.select(cols).where(requesting_container_uuid: my_child_containers).results if my_child_containers.any?
      reqs.each {|cr| grandchildren[cr.requesting_container_uuid] << cr} if reqs

      my_children.each do |cr|
        items << cr.work_unit(cr.name || 'this container', child_objects=grandchildren[cr.container_uuid])
      end
    end

    @child_proxies = nil #no need of this any longer
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
    @proxied.is_a?(ContainerRequest) &&
      @proxied.state == "Committed" &&
      (@proxied.priority > 0 || get(:state, @container) != 'Running') &&
      @proxied.editable?
  end

  def container_uuid
    get(:container_uuid)
  end

  def requesting_container_uuid
    get(:requesting_container_uuid)
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
    if get(:state) == 'Final' && get(:state, @container) != 'Complete'
      # Request was finalized before its container started (or the
      # container was cancelled)
      return 'Cancelled'
    end
    state = get(:state, @container) || get(:state, @proxied)
    case state
    when 'Locked', 'Queued'
      if priority == 0
        'On hold'
      else
        'Queued'
      end
    when 'Complete'
      if exit_code == 0
        state
      else
        'Failed'
      end
    else
      # Cancelled, Running, or Uncommitted (no container assigned)
      state
    end
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
    if @proxied.is_a?(ContainerRequest)
      get(:log_uuid)
    else
      get(:log)
    end
  end

  def outputs
    items = []
    if @proxied.is_a?(ContainerRequest)
      out = get(:output_uuid)
    else
      out = get(:output)
    end
    items << out if out
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
    from_container = get(key, @container)
    from_proxied = get(key, @proxied)

    if from_container.is_a? Hash or from_container.is_a? Array
      if from_container.any? then from_container else from_proxied end
    else
      from_container || from_proxied
    end
  end
end
