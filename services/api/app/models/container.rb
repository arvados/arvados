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

  # Posgresql JSONB columns should NOT be declared as serialized, Rails 5
  # already know how to properly treat them.
  attribute :secret_mounts, :jsonbHash, default: {}
  attribute :runtime_status, :jsonbHash, default: {}
  attribute :runtime_auth_scopes, :jsonbArray, default: []
  attribute :output_storage_classes, :jsonbArray, default: lambda { Rails.configuration.DefaultStorageClasses }
  attribute :output_properties, :jsonbHash, default: {}

  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array
  serialize :scheduling_parameters, Hash

  after_find :fill_container_defaults_after_find
  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :set_timestamps
  before_validation :check_lock
  before_validation :check_unlock
  validates :command, :container_image, :output_path, :cwd, :priority, { presence: true }
  validates :priority, numericality: { only_integer: true, greater_than_or_equal_to: 0 }
  validate :validate_runtime_status
  validate :validate_state_change
  validate :validate_change
  validate :validate_lock
  validate :validate_output
  after_validation :assign_auth
  before_save :sort_serialized_attrs
  before_save :update_secret_mounts_md5
  before_save :scrub_secrets
  before_save :clear_runtime_status_when_queued
  after_save :update_cr_logs
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
    t.add :runtime_status
    t.add :started_at
    t.add :state
    t.add :auth_uuid
    t.add :scheduling_parameters
    t.add :runtime_user_uuid
    t.add :runtime_auth_scopes
    t.add :lock_count
    t.add :gateway_address
    t.add :interactive_session_started
    t.add :output_storage_classes
    t.add :output_properties
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
    Running => [Complete, Cancelled],
    Complete => [Cancelled]
  }

  def self.limit_index_columns_read
    ["mounts"]
  end

  def self.full_text_searchable_columns
    super - ["secret_mounts", "secret_mounts_md5", "runtime_token", "gateway_address", "output_storage_classes"]
  end

  def self.searchable_columns *args
    super - ["secret_mounts_md5", "runtime_token", "gateway_address", "output_storage_classes"]
  end

  def logged_attributes
    super.except('secret_mounts', 'runtime_token')
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
    return true unless saved_change_to_priority?
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
    if req.runtime_token.nil?
      runtime_user = if req.modified_by_user_uuid.nil?
                       current_user
                     else
                       User.find_by_uuid(req.modified_by_user_uuid)
                     end
      runtime_auth_scopes = ["all"]
    else
      auth = ApiClientAuthorization.validate(token: req.runtime_token)
      if auth.nil?
        raise ArgumentError.new "Invalid runtime token"
      end
      runtime_user = User.find_by_id(auth.user_id)
      runtime_auth_scopes = auth.scopes
    end
    c_attrs = act_as_user runtime_user do
      {
        command: req.command,
        cwd: req.cwd,
        environment: req.environment,
        output_path: req.output_path,
        container_image: resolve_container_image(req.container_image),
        mounts: resolve_mounts(req.mounts),
        runtime_constraints: resolve_runtime_constraints(req.runtime_constraints),
        scheduling_parameters: req.scheduling_parameters,
        secret_mounts: req.secret_mounts,
        runtime_token: req.runtime_token,
        runtime_user_uuid: runtime_user.uuid,
        runtime_auth_scopes: runtime_auth_scopes,
        output_storage_classes: req.output_storage_classes,
      }
    end
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
    runtime_constraints.each do |k, v|
      if v.is_a? Array
        rc[k] = v[0]
      else
        rc[k] = v
      end
    end
    if rc['keep_cache_ram'] == 0
      rc['keep_cache_ram'] = Rails.configuration.Containers.DefaultKeepCacheRAM
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

      uuid = mount.delete 'uuid'

      if mount['portable_data_hash'].nil? and !uuid.nil?
        # PDH not supplied, try by UUID
        c = Collection.
          readable_by(current_user).
          where(uuid: uuid).
          select(:portable_data_hash).
          first
        if !c
          raise ArvadosModel::UnresolvableContainerError.new "cannot mount collection #{uuid.inspect}: not found"
        end
        mount['portable_data_hash'] = c.portable_data_hash
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
    candidates = Container.where_serialized(:command, attrs[:command], md5: true)
    log_reuse_info(candidates) { "after filtering on command #{attrs[:command].inspect}" }

    candidates = candidates.where('cwd = ?', attrs[:cwd])
    log_reuse_info(candidates) { "after filtering on cwd #{attrs[:cwd].inspect}" }

    candidates = candidates.where_serialized(:environment, attrs[:environment], md5: true)
    log_reuse_info(candidates) { "after filtering on environment #{attrs[:environment].inspect}" }

    candidates = candidates.where('output_path = ?', attrs[:output_path])
    log_reuse_info(candidates) { "after filtering on output_path #{attrs[:output_path].inspect}" }

    image = resolve_container_image(attrs[:container_image])
    candidates = candidates.where('container_image = ?', image)
    log_reuse_info(candidates) { "after filtering on container_image #{image.inspect} (resolved from #{attrs[:container_image].inspect})" }

    candidates = candidates.where_serialized(:mounts, resolve_mounts(attrs[:mounts]), md5: true)
    log_reuse_info(candidates) { "after filtering on mounts #{attrs[:mounts].inspect}" }

    secret_mounts_md5 = Digest::MD5.hexdigest(SafeJSON.dump(self.deep_sort_hash(attrs[:secret_mounts])))
    candidates = candidates.where('secret_mounts_md5 = ?', secret_mounts_md5)
    log_reuse_info(candidates) { "after filtering on secret_mounts_md5 #{secret_mounts_md5.inspect}" }

    if attrs[:runtime_constraints]['cuda'].nil?
      attrs[:runtime_constraints]['cuda'] = {
        'device_count' => 0,
        'driver_version' => '',
        'hardware_capability' => '',
      }
    end
    resolved_runtime_constraints = [resolve_runtime_constraints(attrs[:runtime_constraints])]
    if resolved_runtime_constraints[0]['cuda']['device_count'] == 0
      # If no CUDA requested, extend search to include older container
      # records that don't have a 'cuda' section in runtime_constraints
      resolved_runtime_constraints << resolved_runtime_constraints[0].except('cuda')
    end

    candidates = candidates.where_serialized(:runtime_constraints, resolved_runtime_constraints, md5: true, multivalue: true)
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

    # Check for non-failing Running candidates and return the most likely to finish sooner.
    log_reuse_info { "checking for state=Running..." }
    running = candidates.where(state: Running).
              where("(runtime_status->'error') is null").
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

  def lock
    self.with_lock do
      if self.state != Queued
        raise LockFailedError.new("cannot lock when #{self.state}")
      end
      self.update_attributes!(state: Locked)
    end
  end

  def check_lock
    if state_was == Queued and state == Locked
      if self.priority <= 0
        raise LockFailedError.new("cannot lock when priority<=0")
      end
      self.lock_count = self.lock_count+1
    end
  end

  def unlock
    self.with_lock do
      if self.state != Locked
        raise InvalidStateTransitionError.new("cannot unlock when #{self.state}")
      end
      self.update_attributes!(state: Queued)
    end
  end

  def check_unlock
    if state_was == Locked and state == Queued
      if self.locked_by_uuid != current_api_client_authorization.uuid
        raise ArvadosModel::PermissionDeniedError.new("locked by a different token")
      end
      if self.lock_count >= Rails.configuration.Containers.MaxDispatchAttempts
        self.state = Cancelled
        self.runtime_status = {error: "Failed to start container.  Cancelled after exceeding 'Containers.MaxDispatchAttempts' (lock_count=#{self.lock_count})"}
      end
    end
  end

  def self.readable_by(*users_list)
    # Load optional keyword arguments, if they exist.
    if users_list.last.is_a? Hash
      kwargs = users_list.pop
    else
      kwargs = {}
    end
    if users_list.select { |u| u.is_admin }.any?
      return super
    end
    Container.where(ContainerRequest.readable_by(*users_list).where("containers.uuid = container_requests.container_uuid").arel.exists)
  end

  def final?
    [Complete, Cancelled].include?(self.state)
  end

  def self.for_current_token
    return if !current_api_client_authorization
    _, _, _, container_uuid = Thread.current[:token].split('/')
    if container_uuid.nil?
      Container.where(auth_uuid: current_api_client_authorization.uuid).first
    else
      Container.where('auth_uuid=? or (uuid=? and runtime_token=?)',
                      current_api_client_authorization.uuid,
                      container_uuid,
                      current_api_client_authorization.token).first
    end
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

  def permission_to_destroy
    current_user.andand.is_admin
  end

  def ensure_owner_uuid_is_permitted
    # validate_change ensures owner_uuid can't be changed at all --
    # except during create, which requires admin privileges. Checking
    # permission here would be superfluous.
    true
  end

  def set_timestamps
    if self.state_changed? and self.state == Running
      self.started_at ||= db_current_time
    end

    if self.state_changed? and [Complete, Cancelled].include? self.state
      self.finished_at ||= db_current_time
    end
  end

  # Check that well-known runtime status keys have desired data types
  def validate_runtime_status
    [
      'error', 'errorDetail', 'warning', 'warningDetail', 'activity'
    ].each do |k|
      if self.runtime_status.andand.include?(k) && !self.runtime_status[k].is_a?(String)
        errors.add(:runtime_status, "'#{k}' value must be a string")
      end
    end
  end

  def validate_change
    permitted = [:state]
    progress_attrs = [:progress, :runtime_status, :log, :output, :output_properties, :exit_code]
    final_attrs = [:finished_at]

    if self.new_record?
      permitted.push(:owner_uuid, :command, :container_image, :cwd,
                     :environment, :mounts, :output_path, :priority,
                     :runtime_constraints, :scheduling_parameters,
                     :secret_mounts, :runtime_token,
                     :runtime_user_uuid, :runtime_auth_scopes,
                     :output_storage_classes)
    end

    case self.state
    when Locked
      permitted.push :priority, :runtime_status, :log, :lock_count

    when Queued
      permitted.push :priority

    when Running
      permitted.push :priority, :output_properties, :gateway_address, *progress_attrs
      if self.state_changed?
        permitted.push :started_at
      end
      if !self.interactive_session_started_was
        permitted.push :interactive_session_started
      end

    when Complete
      if self.state_was == Running
        permitted.push *final_attrs, *progress_attrs
      end

    when Cancelled
      case self.state_was
      when Running
        permitted.push :finished_at, *progress_attrs
      when Queued, Locked
        permitted.push :finished_at, :log, :runtime_status
      end

    else
      # The state_transitions check will add an error message for this
      return false
    end

    if self.state_was == Running &&
       !current_api_client_authorization.nil? &&
       (current_api_client_authorization.uuid == self.auth_uuid ||
        current_api_client_authorization.token == self.runtime_token)
      # The contained process itself can write final attrs but can't
      # change priority or log.
      permitted.push *final_attrs
      permitted = permitted - [:log, :priority]
    elsif !current_user.andand.is_admin
      raise PermissionDeniedError
    elsif self.locked_by_uuid && self.locked_by_uuid != current_api_client_authorization.andand.uuid
      # When locked, progress fields cannot be updated by the wrong
      # dispatcher, even though it has admin privileges.
      permitted = permitted - progress_attrs
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

  def update_cr_logs
    # If self.final?, this update is superfluous: the final log/output
    # update will be done when handle_completed calls finalize! on
    # each requesting CR.
    return if self.final? || !saved_change_to_log?
    leave_modified_by_user_alone do
      ContainerRequest.where(container_uuid: self.uuid).each do |cr|
        cr.update_collections(container: self, collections: ['log'])
        cr.save!
      end
    end
  end

  def assign_auth
    if self.auth_uuid_changed?
         return errors.add :auth_uuid, 'is readonly'
    end
    if not [Locked, Running].include? self.state
      # Don't need one. If auth already exists, expire it.
      #
      # We use db_transaction_time here (not db_current_time) to
      # ensure the token doesn't validate later in the same
      # transaction (e.g., in a test case) by satisfying expires_at >
      # transaction timestamp.
      self.auth.andand.update_attributes(expires_at: db_transaction_time)
      self.auth = nil
      return
    elsif self.auth
      # already have one
      return
    end
    if self.runtime_token.nil?
      if self.runtime_user_uuid.nil?
        # legacy behavior, we don't have a runtime_user_uuid so get
        # the user from the highest priority container request, needed
        # when performing an upgrade and there are queued containers,
        # and some tests.
        cr = ContainerRequest.
               where('container_uuid=? and priority>0', self.uuid).
               order('priority desc').
               first
        if !cr
          return errors.add :auth_uuid, "cannot be assigned because priority <= 0"
        end
        self.runtime_user_uuid = cr.modified_by_user_uuid
        self.runtime_auth_scopes = ["all"]
      end

      # Generate a new token. This runs with admin credentials as it's done by a
      # dispatcher user, so expires_at isn't enforced by API.MaxTokenLifetime.
      self.auth = ApiClientAuthorization.
                    create!(user_id: User.find_by_uuid(self.runtime_user_uuid).id,
                            api_client_id: 0,
                            scopes: self.runtime_auth_scopes)
    end
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
    if self.runtime_auth_scopes_changed?
      self.runtime_auth_scopes = self.runtime_auth_scopes.sort
    end
  end

  def update_secret_mounts_md5
    if self.secret_mounts_changed?
      self.secret_mounts_md5 = Digest::MD5.hexdigest(
        SafeJSON.dump(self.class.deep_sort_hash(self.secret_mounts)))
    end
  end

  def scrub_secrets
    # this runs after update_secret_mounts_md5, so the
    # secret_mounts_md5 will still reflect the secrets that are being
    # scrubbed here.
    if self.state_changed? && self.final?
      self.secret_mounts = {}
      self.runtime_token = nil
    end
  end

  def clear_runtime_status_when_queued
    # Avoid leaking status messages between different dispatch attempts
    if self.state_was == Locked && self.state == Queued
      self.runtime_status = {}
    end
  end

  def handle_completed
    # This container is finished so finalize any associated container requests
    # that are associated with this container.
    if saved_change_to_state? and self.final?
      # These get wiped out by with_lock (which reloads the record),
      # so record them now in case we need to schedule a retry.
      prev_secret_mounts = secret_mounts_before_last_save
      prev_runtime_token = runtime_token_before_last_save

      # Need to take a lock on the container to ensure that any
      # concurrent container requests that might try to reuse this
      # container will block until the container completion
      # transaction finishes.  This ensure that concurrent container
      # requests that try to reuse this container are finalized (on
      # Complete) or don't reuse it (on Cancelled).
      self.with_lock do
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
              scheduling_parameters: self.scheduling_parameters,
              secret_mounts: prev_secret_mounts,
              runtime_token: prev_runtime_token,
              runtime_user_uuid: self.runtime_user_uuid,
              runtime_auth_scopes: self.runtime_auth_scopes
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
end
