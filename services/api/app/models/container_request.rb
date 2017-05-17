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
  validates :output_ttl, numericality: { only_integer: true, greater_than_or_equal_to: 0 }
  validate :validate_state_change
  validate :check_update_whitelist
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
    t.add :output_ttl
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

  AttrsPermittedAlways = [:owner_uuid, :state, :name, :description]
  AttrsPermittedBeforeCommit = [:command, :container_count_max,
  :container_image, :cwd, :environment, :filters, :mounts,
  :output_path, :priority, :properties, :requesting_container_uuid,
  :runtime_constraints, :state, :container_uuid, :use_existing,
  :scheduling_parameters, :output_name, :output_ttl]

  def self.limit_index_columns_read
    ["mounts"]
  end

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
      coll_name = "Container #{out_type} for request #{uuid}"
      trash_at = nil
      if out_type == 'output'
        if self.output_name
          coll_name = self.output_name
        end
        if self.output_ttl > 0
          trash_at = db_current_time + self.output_ttl
        end
      end
      manifest = Collection.unscoped do
        Collection.where(portable_data_hash: pdh).first.manifest_text
      end

      coll = Collection.new(owner_uuid: owner_uuid,
                            manifest_text: manifest,
                            portable_data_hash: pdh,
                            name: coll_name,
                            trash_at: trash_at,
                            delete_at: trash_at,
                            properties: {
                              'type' => out_type,
                              'container_request' => uuid,
                            })
      coll.save_with_unique_name!
      if out_type == 'output'
        out_coll = coll.uuid
      else
        log_coll = coll.uuid
      end
    end
    update_attributes!(state: Final, output_uuid: out_coll, log_uuid: log_coll)
  end

  def self.full_text_searchable_columns
    super - ["mounts"]
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
    self.output_ttl ||= 0
  end

  def set_container
    if (container_uuid_changed? and
        not current_user.andand.is_admin and
        not container_uuid.nil?)
      errors.add :container_uuid, "can only be updated to nil."
      return false
    end
    if state_changed? and state == Committed and container_uuid.nil?
      self.container_uuid = Container.resolve(self).uuid
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
      [['vcpus', true],
       ['ram', true],
       ['keep_cache_ram', false]].each do |k, required|
        if !required && !runtime_constraints.include?(k)
          next
        end
        v = runtime_constraints[k]
        unless (v.is_a?(Integer) && v > 0)
          errors.add(:runtime_constraints,
                     "[#{k}]=#{v.inspect} must be a positive integer")
        end
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

  def check_update_whitelist
    permitted = AttrsPermittedAlways.dup

    if self.new_record? || self.state_was == Uncommitted
      # Allow create-and-commit in a single operation.
      permitted.push *AttrsPermittedBeforeCommit
    end

    case self.state
    when Committed
      permitted.push :priority, :container_count_max, :container_uuid

      if self.container_uuid.nil?
        self.errors.add :container_uuid, "has not been resolved to a container."
      end

      if self.priority.nil?
        self.errors.add :priority, "cannot be nil"
      end

      # Allow container count to increment by 1
      if (self.container_uuid &&
          self.container_uuid != self.container_uuid_was &&
          self.container_count == 1 + (self.container_count_was || 0))
        permitted.push :container_count
      end

    when Final
      if self.state_changed? and not current_user.andand.is_admin
        self.errors.add :state, "of container request can only be set to Final by system."
      end

      if self.state_was == Committed
        permitted.push :output_uuid, :log_uuid
      end

    end

    super(permitted)
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
