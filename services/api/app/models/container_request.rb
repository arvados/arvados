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
  serialize :scheduling_parameters, Hash

  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :validate_runtime_constraints
  before_validation :validate_scheduling_parameters
  before_validation :set_container
  validates :command, :container_image, :output_path, :cwd, :presence => true
  validate :validate_state_change
  validate :validate_change
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
    t.add :log_uuid
    t.add :mounts
    t.add :name
    t.add :output_name
    t.add :output_path
    t.add :output_uuid
    t.add :priority
    t.add :properties
    t.add :requesting_container_uuid
    t.add :runtime_constraints
    t.add :scheduling_parameters
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
    out_coll = nil
    log_coll = nil
    c = Container.find_by_uuid(container_uuid)
    ['output', 'log'].each do |out_type|
      pdh = c.send(out_type)
      next if pdh.nil?
      if self.output_name and out_type == 'output'
        coll_name = self.output_name
      else
        coll_name = "Container #{out_type} for request #{uuid}"
      end
      manifest = Collection.where(portable_data_hash: pdh).first.manifest_text
      begin
        coll = Collection.create!(owner_uuid: owner_uuid,
                                  manifest_text: manifest,
                                  portable_data_hash: pdh,
                                  name: coll_name,
                                  properties: {
                                    'type' => out_type,
                                    'container_request' => uuid,
                                  })
      rescue ActiveRecord::RecordNotUnique => rn
        ActiveRecord::Base.connection.execute 'ROLLBACK'
        raise unless out_type == 'output' and self.output_name
        # Postgres specific unique name check. See ApplicationController#create for
        # a detailed explanation.
        raise unless rn.original_exception.is_a? PG::UniqueViolation
        err = rn.original_exception
        detail = err.result.error_field(PG::Result::PG_DIAG_MESSAGE_DETAIL)
        raise unless /^Key \(owner_uuid, name\)=\([a-z0-9]{5}-[a-z0-9]{5}-[a-z0-9]{15}, .*?\) already exists\./.match detail
        # Output collection name collision detected: append a timestamp.
        coll_name = "#{self.output_name} #{Time.now.getgm.strftime('%FT%TZ')}"
        retry
      end
      if out_type == 'output'
        out_coll = coll.uuid
      else
        log_coll = coll.uuid
      end
    end
    update_attributes!(state: Final, output_uuid: out_coll, log_uuid: log_coll)
  end

  protected

  def fill_field_defaults
    self.state ||= Uncommitted
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.cwd ||= "."
    self.container_count_max ||= Rails.configuration.container_count_max
    self.scheduling_parameters ||= {}
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
        c_attrs[:scheduling_parameters] = self.scheduling_parameters
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

      if runtime_constraints.include? 'keep_cache_ram' and
         (!runtime_constraints['keep_cache_ram'].is_a?(Integer) or
          runtime_constraints['keep_cache_ram'] <= 0)
            errors.add :runtime_constraints, "keep_cache_ram must be a positive integer"
      elsif !runtime_constraints.include? 'keep_cache_ram'
        runtime_constraints['keep_cache_ram'] = Rails.configuration.container_default_keep_cache_ram
      end
    end
  end

  def validate_scheduling_parameters
    if self.state == Committed
      if scheduling_parameters.include? 'partitions' and
         (!scheduling_parameters['partitions'].is_a?(Array) ||
          scheduling_parameters['partitions'].reject{|x| !x.is_a?(String)}.size !=
            scheduling_parameters['partitions'].size)
            errors.add :scheduling_parameters, "partitions must be an array of strings"
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
                     :state, :container_uuid, :use_existing, :scheduling_parameters,
                     :output_name

    when Committed
      if container_uuid.nil?
        errors.add :container_uuid, "has not been resolved to a container."
      end

      if priority.nil?
        errors.add :priority, "cannot be nil"
      end

      # Can update priority, container count, name and description
      permitted.push :priority, :container_count, :container_count_max, :container_uuid,
                     :name, :description

      if self.state_changed?
        # Allow create-and-commit in a single operation.
        permitted.push :command, :container_image, :cwd, :description, :environment,
                       :filters, :mounts, :name, :output_path, :properties,
                       :requesting_container_uuid, :runtime_constraints,
                       :state, :container_uuid, :use_existing, :scheduling_parameters,
                       :output_name
      end

    when Final
      if not current_user.andand.is_admin and not (self.name_changed? || self.description_changed?)
        errors.add :state, "of container request can only be set to Final by system."
      end

      if self.state_changed? || self.name_changed? || self.description_changed? || self.output_uuid_changed? || self.log_uuid_changed?
          permitted.push :state, :name, :description, :output_uuid, :log_uuid
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
