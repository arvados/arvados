require 'whitelist_update'

class ContainerRequest < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include WhitelistUpdate

  serialize :properties, Hash
  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array

  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :set_container
  validates :command, :container_image, :output_path, :cwd, :presence => true
  validate :validate_state_change
  validate :validate_change
  validate :validate_runtime_constraints
  after_save :update_priority
  after_save :finalize_if_needed
  before_create :set_requesting_container_uuid

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_count
    t.add :container_count_max
    t.add :container_image
    t.add :container_uuid
    t.add :cwd
    t.add :description
    t.add :environment
    t.add :expires_at
    t.add :filters
    t.add :mounts
    t.add :name
    t.add :output_path
    t.add :priority
    t.add :properties
    t.add :requesting_container_uuid
    t.add :runtime_constraints
    t.add :state
    t.add :use_existing
  end

  # Supported states for a container request
  States =
    [
     (Uncommitted = 'Uncommitted'),
     (Committed = 'Committed'),
     (Final = 'Final'),
    ]

  State_transitions = {
    nil => [Uncommitted, Committed],
    Uncommitted => [Committed],
    Committed => [Final]
  }

  def state_transitions
    State_transitions
  end

  def skip_uuid_read_permission_check
    # XXX temporary until permissions are sorted out.
    %w(modified_by_client_uuid container_uuid requesting_container_uuid)
  end

  def finalize_if_needed
    if state == Committed && Container.find_by_uuid(container_uuid).final?
      reload
      act_as_system_user do
        finalize!
      end
    end
  end

  # Finalize the container request after the container has
  # finished/cancelled.
  def finalize!
    update_attributes!(state: Final)
    c = Container.find_by_uuid(container_uuid)
    ['output', 'log'].each do |out_type|
      pdh = c.send(out_type)
      next if pdh.nil?
      manifest = Collection.where(portable_data_hash: pdh).first.manifest_text
      Collection.create!(owner_uuid: owner_uuid,
                         manifest_text: manifest,
                         portable_data_hash: pdh,
                         name: "Container #{out_type} for request #{uuid}",
                         properties: {
                           'type' => out_type,
                           'container_request' => uuid,
                         })
    end
  end

  protected

  def fill_field_defaults
    self.state ||= Uncommitted
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.cwd ||= "."
    self.container_count_max ||= Rails.configuration.container_count_max
  end

  # Create a new container (or find an existing one) to satisfy this
  # request.
  def resolve
    c_mounts = mounts_for_container
    c_runtime_constraints = runtime_constraints_for_container
    c_container_image = container_image_for_container
    c = act_as_system_user do
      c_attrs = {command: self.command,
                 cwd: self.cwd,
                 environment: self.environment,
                 output_path: self.output_path,
                 container_image: c_container_image,
                 mounts: c_mounts,
                 runtime_constraints: c_runtime_constraints}

      reusable = self.use_existing ? Container.find_reusable(c_attrs) : nil
      if not reusable.nil?
        reusable
      else
        Container.create!(c_attrs)
      end
    end
    self.container_uuid = c.uuid
  end

  # Return a runtime_constraints hash that complies with
  # self.runtime_constraints but is suitable for saving in a container
  # record, i.e., has specific values instead of ranges.
  #
  # Doing this as a step separate from other resolutions, like "git
  # revision range to commit hash", makes sense only when there is no
  # opportunity to reuse an existing container (e.g., container reuse
  # is not implemented yet, or we have already found that no existing
  # containers are suitable).
  def runtime_constraints_for_container
    rc = {}
    runtime_constraints.each do |k, v|
      if v.is_a? Array
        rc[k] = v[0]
      else
        rc[k] = v
      end
    end
    rc
  end

  # Return a mounts hash suitable for a Container, i.e., with every
  # readonly collection UUID resolved to a PDH.
  def mounts_for_container
    c_mounts = {}
    mounts.each do |k, mount|
      mount = mount.dup
      c_mounts[k] = mount
      if mount['kind'] != 'collection'
        next
      end
      if (uuid = mount.delete 'uuid')
        c = Collection.
          readable_by(current_user).
          where(uuid: uuid).
          select(:portable_data_hash).
          first
        if !c
          raise ArvadosModel::UnresolvableContainerError.new "cannot mount collection #{uuid.inspect}: not found"
        end
        if mount['portable_data_hash'].nil?
          # PDH not supplied by client
          mount['portable_data_hash'] = c.portable_data_hash
        elsif mount['portable_data_hash'] != c.portable_data_hash
          # UUID and PDH supplied by client, but they don't agree
          raise ArgumentError.new "cannot mount collection #{uuid.inspect}: current portable_data_hash #{c.portable_data_hash.inspect} does not match #{c['portable_data_hash'].inspect} in request"
        end
      end
    end
    return c_mounts
  end

  # Return a container_image PDH suitable for a Container.
  def container_image_for_container
    coll = Collection.for_latest_docker_image(container_image)
    if !coll
      raise ArvadosModel::UnresolvableContainerError.new "docker image #{container_image.inspect} not found"
    end
    return coll.portable_data_hash
  end

  def set_container
    if (container_uuid_changed? and
        not current_user.andand.is_admin and
        not container_uuid.nil?)
      errors.add :container_uuid, "can only be updated to nil."
      return false
    end
    if state_changed? and state == Committed and container_uuid.nil?
      resolve
    end
    if self.container_uuid != self.container_uuid_was
      if self.container_count_changed?
        errors.add :container_count, "cannot be updated directly."
        return false
      else
        self.container_count += 1
      end
    end
  end

  def validate_runtime_constraints
    case self.state
    when Committed
      ['vcpus', 'ram'].each do |k|
        if not (runtime_constraints.include? k and
                runtime_constraints[k].is_a? Integer and
                runtime_constraints[k] > 0)
          errors.add :runtime_constraints, "#{k} must be a positive integer"
        end
      end
    end
  end

  def validate_change
    permitted = [:owner_uuid]

    case self.state
    when Uncommitted
      # Permit updating most fields
      permitted.push :command, :container_count_max,
                     :container_image, :cwd, :description, :environment,
                     :filters, :mounts, :name, :output_path, :priority,
                     :properties, :requesting_container_uuid, :runtime_constraints,
                     :state, :container_uuid, :use_existing

    when Committed
      if container_uuid.nil?
        errors.add :container_uuid, "has not been resolved to a container."
      end

      if priority.nil?
        errors.add :priority, "cannot be nil"
      end

      # Can update priority, container count, name and description
      permitted.push :priority, :container_count, :container_count_max, :container_uuid, :name, :description

      if self.state_changed?
        # Allow create-and-commit in a single operation.
        permitted.push :command, :container_image, :cwd, :description, :environment,
                       :filters, :mounts, :name, :output_path, :properties,
                       :requesting_container_uuid, :runtime_constraints,
                       :state, :container_uuid
      end

    when Final
      if not current_user.andand.is_admin and not (self.name_changed? || self.description_changed?)
        errors.add :state, "of container request can only be set to Final by system."
      end

      if self.state_changed? || self.name_changed? || self.description_changed?
          permitted.push :state, :name, :description
      else
        errors.add :state, "does not allow updates"
      end

    else
      errors.add :state, "invalid value"
    end

    check_update_whitelist permitted
  end

  def update_priority
    if self.state_changed? or
        self.priority_changed? or
        self.container_uuid_changed?
      act_as_system_user do
        Container.
          where('uuid in (?)',
                [self.container_uuid_was, self.container_uuid].compact).
          map(&:update_priority!)
      end
    end
  end

  def set_requesting_container_uuid
    return !new_record? if self.requesting_container_uuid   # already set

    token_uuid = current_api_client_authorization.andand.uuid
    container = Container.where('auth_uuid=?', token_uuid).order('created_at desc').first
    self.requesting_container_uuid = container.uuid if container
    true
  end
end
