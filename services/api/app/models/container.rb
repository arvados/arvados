# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'log_reuse_info'
require 'whitelist_update'
require 'safe_json'
require 'update_priority'

class Container < ArvadosModel
  include ArvadosModelUpdates
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include WhitelistUpdate
  extend CurrentApiClient
  extend DbCurrentTime
  extend LogReuseInfo

  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array
  serialize :scheduling_parameters, Hash
  serialize :secret_mounts, Hash

  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :set_timestamps
  validates :command, :container_image, :output_path, :cwd, :priority, { presence: true }
  validates :priority, numericality: { only_integer: true, greater_than_or_equal_to: 0 }
  validate :validate_state_change
  validate :validate_change
  validate :validate_lock
  validate :validate_output
  after_validation :assign_auth
  before_save :sort_serialized_attrs
  before_save :update_secret_mounts_md5
  before_save :scrub_secret_mounts
  after_save :handle_completed
  after_save :propagate_priority
  after_commit { UpdatePriority.run_update_thread }

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

  def self.limit_index_columns_read
    ["mounts"]
  end

  def self.full_text_searchable_columns
    super - ["secret_mounts", "secret_mounts_md5"]
  end

  def self.searchable_columns *args
    super - ["secret_mounts_md5"]
  end

  def logged_attributes
    super.except('secret_mounts')
  end

  def state_transitions
    State_transitions
  end

  # Container priority is the highest "computed priority" of any
  # matching request. The computed priority of a container-submitted
  # request is the priority of the submitting container. The computed
  # priority of a user-submitted request is a function of
  # user-assigned priority and request creation time.
  def update_priority!
    return if ![Queued, Locked, Running].include?(state)
    p = ContainerRequest.
        where('container_uuid=? and priority>0', uuid).
        includes(:requesting_container).
        lock(true).
        map do |cr|
      if cr.requesting_container
        cr.requesting_container.priority
      else
        (cr.priority << 50) - (cr.created_at.to_time.to_f * 1000).to_i
      end
    end.max || 0
    update_attributes!(priority: p)
  end

  def propagate_priority
    return true unless priority_changed?
    act_as_system_user do
      # Update the priority of child container requests to match new
      # priority of the parent container (ignoring requests with no
      # container assigned, because their priority doesn't matter).
      ContainerRequest.
        where(requesting_container_uuid: self.uuid,
              state: ContainerRequest::Committed).
        where('container_uuid is not null').
        includes(:container).
        map(&:container).
        map(&:update_priority!)
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
      secret_mounts: req.secret_mounts,
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
    log_reuse_info { "starting with #{Container.all.count} container records in database" }
    candidates = Container.where_serialized(:command, attrs[:command])
    log_reuse_info(candidates) { "after filtering on command #{attrs[:command].inspect}" }

    candidates = candidates.where('cwd = ?', attrs[:cwd])
    log_reuse_info(candidates) { "after filtering on cwd #{attrs[:cwd].inspect}" }

    candidates = candidates.where_serialized(:environment, attrs[:environment])
    log_reuse_info(candidates) { "after filtering on environment #{attrs[:environment].inspect}" }

    candidates = candidates.where('output_path = ?', attrs[:output_path])
    log_reuse_info(candidates) { "after filtering on output_path #{attrs[:output_path].inspect}" }

    image = resolve_container_image(attrs[:container_image])
    candidates = candidates.where('container_image = ?', image)
    log_reuse_info(candidates) { "after filtering on container_image #{image.inspect} (resolved from #{attrs[:container_image].inspect})" }

    candidates = candidates.where_serialized(:mounts, resolve_mounts(attrs[:mounts]))
    log_reuse_info(candidates) { "after filtering on mounts #{attrs[:mounts].inspect}" }

    candidates = candidates.where('secret_mounts_md5 = ?', Digest::MD5.hexdigest(SafeJSON.dump(self.deep_sort_hash(attrs[:secret_mounts]))))
    log_reuse_info(candidates) { "after filtering on mounts #{attrs[:mounts].inspect}" }

    candidates = candidates.where_serialized(:runtime_constraints, resolve_runtime_constraints(attrs[:runtime_constraints]))
    log_reuse_info(candidates) { "after filtering on runtime_constraints #{attrs[:runtime_constraints].inspect}" }

    log_reuse_info { "checking for state=Complete with readable output and log..." }

    select_readable_pdh = Collection.
      readable_by(current_user).
      select(:portable_data_hash).
      to_sql

    usable = candidates.where(state: Complete, exit_code: 0)
    log_reuse_info(usable) { "with state=Complete, exit_code=0" }

    usable = usable.where("log IN (#{select_readable_pdh})")
    log_reuse_info(usable) { "with readable log" }

    usable = usable.where("output IN (#{select_readable_pdh})")
    log_reuse_info(usable) { "with readable output" }

    usable = usable.order('finished_at ASC').limit(1).first
    if usable
      log_reuse_info { "done, reusing container #{usable.uuid} with state=Complete" }
      return usable
    end

    # Check for Running candidates and return the most likely to finish sooner.
    log_reuse_info { "checking for state=Running..." }
    running = candidates.where(state: Running).
              order('progress desc, started_at asc').
              limit(1).first
    if running
      log_reuse_info { "done, reusing container #{running.uuid} with state=Running" }
      return running
    else
      log_reuse_info { "have no containers in Running state" }
    end

    # Check for Locked or Queued ones and return the most likely to start first.
    locked_or_queued = candidates.
                       where("state IN (?)", [Locked, Queued]).
                       order('state asc, priority desc, created_at asc').
                       limit(1).first
    if locked_or_queued
      log_reuse_info { "done, reusing container #{locked_or_queued.uuid} with state=#{locked_or_queued.state}" }
      return locked_or_queued
    else
      log_reuse_info { "have no containers in Locked or Queued state" }
    end

    log_reuse_info { "done, no reusable container found" }
    nil
  end

  def check_lock_fail
    if self.state != Queued
      raise LockFailedError.new("cannot lock when #{self.state}")
    elsif self.priority <= 0
      raise LockFailedError.new("cannot lock when priority<=0")
    end
  end

  def lock
    # Check invalid state transitions once before getting the lock
    # (because it's cheaper that way) and once after getting the lock
    # (because state might have changed while acquiring the lock).
    check_lock_fail
    transaction do
      reload
      check_lock_fail
      update_attributes!(state: Locked)
    end
  end

  def check_unlock_fail
    if self.state != Locked
      raise InvalidStateTransitionError.new("cannot unlock when #{self.state}")
    elsif self.locked_by_uuid != current_api_client_authorization.uuid
      raise InvalidStateTransitionError.new("locked by a different token")
    end
  end

  def unlock
    # Check invalid state transitions twice (see lock)
    check_unlock_fail
    transaction do
      reload(lock: 'FOR UPDATE')
      check_unlock_fail
      update_attributes!(state: Queued)
    end
  end

  def self.readable_by(*users_list)
    # Load optional keyword arguments, if they exist.
    if users_list.last.is_a? Hash
      kwargs = users_list.pop
    else
      kwargs = {}
    end
    Container.where(ContainerRequest.readable_by(*users_list).where("containers.uuid = container_requests.container_uuid").exists)
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
    self.priority ||= 0
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
                     :runtime_constraints, :scheduling_parameters,
                     :secret_mounts)
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
        permitted.push :finished_at, :log
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
      c = Collection.
            readable_by(current_user, {include_trash: true}).
            where(portable_data_hash: self.output).
            first
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

  def update_secret_mounts_md5
    if self.secret_mounts_changed?
      self.secret_mounts_md5 = Digest::MD5.hexdigest(
        SafeJSON.dump(self.class.deep_sort_hash(self.secret_mounts)))
    end
  end

  def scrub_secret_mounts
    # this runs after update_secret_mounts_md5, so the
    # secret_mounts_md5 will still reflect the secrets that are being
    # scrubbed here.
    if self.state_changed? && self.final?
      self.secret_mounts = {}
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
              leave_modified_by_user_alone do
                # Use row locking because this increments container_count
                cr.container_uuid = c.uuid
                cr.save!
              end
            end
          end
        end

        # Notify container requests associated with this container
        ContainerRequest.where(container_uuid: uuid,
                               state: ContainerRequest::Committed).each do |cr|
          leave_modified_by_user_alone do
            cr.finalize!
          end
        end

        # Cancel outstanding container requests made by this container.
        ContainerRequest.
          includes(:container).
          where(requesting_container_uuid: uuid,
                state: ContainerRequest::Committed).each do |cr|
          leave_modified_by_user_alone do
            cr.update_attributes!(priority: 0)
            cr.container.reload
            if cr.container.state == Container::Queued || cr.container.state == Container::Locked
              # If the child container hasn't started yet, finalize the
              # child CR now instead of leaving it "on hold", i.e.,
              # Queued with priority 0.  (OTOH, if the child is already
              # running, leave it alone so it can get cancelled the
              # usual way, get a copy of the log collection, etc.)
              cr.update_attributes!(state: ContainerRequest::Final)
            end
          end
        end
      end
    end
  end
end
