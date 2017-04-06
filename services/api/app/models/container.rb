require 'whitelist_update'
require 'safe_json'

class Container < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include WhitelistUpdate
  extend CurrentApiClient

  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array
  serialize :scheduling_parameters, Hash

  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :set_timestamps
  validates :command, :container_image, :output_path, :cwd, :priority, :presence => true
  validate :validate_state_change
  validate :validate_change
  validate :validate_lock
  validate :validate_output
  after_validation :assign_auth
  before_save :sort_serialized_attrs
  after_save :handle_completed

  has_many :container_requests, :foreign_key => :container_uuid, :class_name => 'ContainerRequest', :primary_key => :uuid
  belongs_to :auth, :class_name => 'ApiClientAuthorization', :foreign_key => :auth_uuid, :primary_key => :uuid

  api_accessible :user, extend: :common do |t|
    t.add :command
    t.add :container_image
    t.add :cwd
    t.add :environment
    t.add :exit_code
    t.add :finished_at
    t.add :locked_by_uuid
    t.add :log
    t.add :mounts
    t.add :output
    t.add :output_path
    t.add :priority
    t.add :progress
    t.add :runtime_constraints
    t.add :started_at
    t.add :state
    t.add :auth_uuid
    t.add :scheduling_parameters
  end

  # Supported states for a container
  States =
    [
     (Queued = 'Queued'),
     (Locked = 'Locked'),
     (Running = 'Running'),
     (Complete = 'Complete'),
     (Cancelled = 'Cancelled')
    ]

  State_transitions = {
    nil => [Queued],
    Queued => [Locked, Cancelled],
    Locked => [Queued, Running, Cancelled],
    Running => [Complete, Cancelled]
  }

  def state_transitions
    State_transitions
  end

  def update_priority!
    if [Queued, Locked, Running].include? self.state
      # Update the priority of this container to the maximum priority of any of
      # its committed container requests and save the record.
      self.priority = ContainerRequest.
        where(container_uuid: uuid,
              state: ContainerRequest::Committed).
        maximum('priority')
      self.save!
    end
  end

  # Create a new container (or find an existing one) to satisfy the
  # given container request.
  def self.resolve(req)
    c_attrs = {
      command: req.command,
      cwd: req.cwd,
      environment: req.environment,
      output_path: req.output_path,
      container_image: resolve_container_image(req.container_image),
      mounts: resolve_mounts(req.mounts),
      runtime_constraints: resolve_runtime_constraints(req.runtime_constraints),
      scheduling_parameters: req.scheduling_parameters,
    }
    act_as_system_user do
      if req.use_existing && (reusable = find_reusable(c_attrs))
        reusable
      else
        Container.create!(c_attrs)
      end
    end
  end

  # Return a runtime_constraints hash that complies with requested but
  # is suitable for saving in a container record, i.e., has specific
  # values instead of ranges.
  #
  # Doing this as a step separate from other resolutions, like "git
  # revision range to commit hash", makes sense only when there is no
  # opportunity to reuse an existing container (e.g., container reuse
  # is not implemented yet, or we have already found that no existing
  # containers are suitable).
  def self.resolve_runtime_constraints(runtime_constraints)
    rc = {}
    defaults = {
      'keep_cache_ram' =>
      Rails.configuration.container_default_keep_cache_ram,
    }
    defaults.merge(runtime_constraints).each do |k, v|
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
  def self.resolve_mounts(mounts)
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
  def self.resolve_container_image(container_image)
    coll = Collection.for_latest_docker_image(container_image)
    if !coll
      raise ArvadosModel::UnresolvableContainerError.new "docker image #{container_image.inspect} not found"
    end
    coll.portable_data_hash
  end

  def self.find_reusable(attrs)
    candidates = Container.
      where_serialized(:command, attrs[:command]).
      where('cwd = ?', attrs[:cwd]).
      where_serialized(:environment, attrs[:environment]).
      where('output_path = ?', attrs[:output_path]).
      where('container_image = ?', resolve_container_image(attrs[:container_image])).
      where_serialized(:mounts, resolve_mounts(attrs[:mounts])).
      where_serialized(:runtime_constraints, resolve_runtime_constraints(attrs[:runtime_constraints]))

    # Check for Completed candidates whose output and log are both readable.
    select_readable_pdh = Collection.
      readable_by(current_user).
      select(:portable_data_hash).
      to_sql
    usable = candidates.
      where(state: Complete).
      where(exit_code: 0).
      where("log IN (#{select_readable_pdh})").
      where("output IN (#{select_readable_pdh})").
      order('finished_at ASC').
      limit(1).
      first
    return usable if usable

    # Check for Running candidates and return the most likely to finish sooner.
    running = candidates.where(state: Running).
      order('progress desc, started_at asc').limit(1).first
    return running if not running.nil?

    # Check for Locked or Queued ones and return the most likely to start first.
    locked_or_queued = candidates.where("state IN (?)", [Locked, Queued]).
      order('state asc, priority desc, created_at asc').limit(1).first
    return locked_or_queued if not locked_or_queued.nil?

    # No suitable candidate found.
    nil
  end

  def lock
    with_lock do
      if self.state == Locked
        raise AlreadyLockedError
      end
      self.state = Locked
      self.save!
    end
  end

  def unlock
    with_lock do
      if self.state == Queued
        raise InvalidStateTransitionError
      end
      self.state = Queued
      self.save!
    end
  end

  def self.readable_by(*users_list)
    if users_list.select { |u| u.is_admin }.any?
      return self
    end
    user_uuids = users_list.map { |u| u.uuid }
    uuid_list = user_uuids + users_list.flat_map { |u| u.groups_i_can(:read) }
    uuid_list.uniq!
    permitted = "(SELECT head_uuid FROM links WHERE link_class='permission' AND tail_uuid IN (:uuids))"
    joins(:container_requests).
      where("container_requests.uuid IN #{permitted} OR "+
            "container_requests.owner_uuid IN (:uuids)",
            uuids: uuid_list)
  end

  def final?
    [Complete, Cancelled].include?(self.state)
  end

  protected

  def fill_field_defaults
    self.state ||= Queued
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.cwd ||= "."
    self.priority ||= 1
    self.scheduling_parameters ||= {}
  end

  def permission_to_create
    current_user.andand.is_admin
  end

  def permission_to_update
    # Override base permission check to allow auth_uuid to set progress and
    # output (only).  Whether it is legal to set progress and output in the current
    # state has already been checked in validate_change.
    current_user.andand.is_admin ||
      (!current_api_client_authorization.nil? and
       [self.auth_uuid, self.locked_by_uuid].include? current_api_client_authorization.uuid)
  end

  def ensure_owner_uuid_is_permitted
    # Override base permission check to allow auth_uuid to set progress and
    # output (only).  Whether it is legal to set progress and output in the current
    # state has already been checked in validate_change.
    if !current_api_client_authorization.nil? and self.auth_uuid == current_api_client_authorization.uuid
      check_update_whitelist [:progress, :output]
    else
      super
    end
  end

  def set_timestamps
    if self.state_changed? and self.state == Running
      self.started_at ||= db_current_time
    end

    if self.state_changed? and [Complete, Cancelled].include? self.state
      self.finished_at ||= db_current_time
    end
  end

  def validate_change
    permitted = [:state]

    if self.new_record?
      permitted.push(:owner_uuid, :command, :container_image, :cwd,
                     :environment, :mounts, :output_path, :priority,
                     :runtime_constraints, :scheduling_parameters)
    end

    case self.state
    when Queued, Locked
      permitted.push :priority

    when Running
      permitted.push :priority, :progress, :output
      if self.state_changed?
        permitted.push :started_at
      end

    when Complete
      if self.state_was == Running
        permitted.push :finished_at, :output, :log, :exit_code
      end

    when Cancelled
      case self.state_was
      when Running
        permitted.push :finished_at, :output, :log
      when Queued, Locked
        permitted.push :finished_at
      end

    else
      # The state_transitions check will add an error message for this
      return false
    end

    check_update_whitelist permitted
  end

  def validate_lock
    if [Locked, Running].include? self.state
      # If the Container was already locked, locked_by_uuid must not
      # changes. Otherwise, the current auth gets the lock.
      need_lock = locked_by_uuid_was || current_api_client_authorization.andand.uuid
    else
      need_lock = nil
    end

    # The caller can provide a new value for locked_by_uuid, but only
    # if it's exactly what we expect. This allows a caller to perform
    # an update like {"state":"Unlocked","locked_by_uuid":null}.
    if self.locked_by_uuid_changed?
      if self.locked_by_uuid != need_lock
        return errors.add :locked_by_uuid, "can only change to #{need_lock}"
      end
    end
    self.locked_by_uuid = need_lock
  end

  def validate_output
    # Output must exist and be readable by the current user.  This is so
    # that a container cannot "claim" a collection that it doesn't otherwise
    # have access to just by setting the output field to the collection PDH.
    if output_changed?
      c = Collection.unscoped do
        Collection.
            readable_by(current_user).
            where(portable_data_hash: self.output).
            first
      end
      if !c
        errors.add :output, "collection must exist and be readable by current user."
      end
    end
  end

  def assign_auth
    if self.auth_uuid_changed?
      return errors.add :auth_uuid, 'is readonly'
    end
    if not [Locked, Running].include? self.state
      # don't need one
      self.auth.andand.update_attributes(expires_at: db_current_time)
      self.auth = nil
      return
    elsif self.auth
      # already have one
      return
    end
    cr = ContainerRequest.
      where('container_uuid=? and priority>0', self.uuid).
      order('priority desc').
      first
    if !cr
      return errors.add :auth_uuid, "cannot be assigned because priority <= 0"
    end
    self.auth = ApiClientAuthorization.
      create!(user_id: User.find_by_uuid(cr.modified_by_user_uuid).id,
              api_client_id: 0)
  end

  def sort_serialized_attrs
    if self.environment_changed?
      self.environment = self.class.deep_sort_hash(self.environment)
    end
    if self.mounts_changed?
      self.mounts = self.class.deep_sort_hash(self.mounts)
    end
    if self.runtime_constraints_changed?
      self.runtime_constraints = self.class.deep_sort_hash(self.runtime_constraints)
    end
    if self.scheduling_parameters_changed?
      self.scheduling_parameters = self.class.deep_sort_hash(self.scheduling_parameters)
    end
  end

  def handle_completed
    # This container is finished so finalize any associated container requests
    # that are associated with this container.
    if self.state_changed? and self.final?
      act_as_system_user do

        if self.state == Cancelled
          retryable_requests = ContainerRequest.where("container_uuid = ? and priority > 0 and state = 'Committed' and container_count < container_count_max", uuid)
        else
          retryable_requests = []
        end

        if retryable_requests.any?
          c_attrs = {
            command: self.command,
            cwd: self.cwd,
            environment: self.environment,
            output_path: self.output_path,
            container_image: self.container_image,
            mounts: self.mounts,
            runtime_constraints: self.runtime_constraints,
            scheduling_parameters: self.scheduling_parameters
          }
          c = Container.create! c_attrs
          retryable_requests.each do |cr|
            cr.with_lock do
              # Use row locking because this increments container_count
              cr.container_uuid = c.uuid
              cr.save
            end
          end
        end

        # Notify container requests associated with this container
        ContainerRequest.where(container_uuid: uuid,
                               state: ContainerRequest::Committed).each do |cr|
          cr.finalize!
        end

        # Try to cancel any outstanding container requests made by this container.
        ContainerRequest.where(requesting_container_uuid: uuid,
                               state: ContainerRequest::Committed).each do |cr|
          cr.priority = 0
          cr.save
        end
      end
    end
  end

end
