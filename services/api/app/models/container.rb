# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'log_reuse_info'
require 'whitelist_update'
require 'safe_json'
require 'update_priorities'

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

  has_many :container_requests,
           class_name: 'ContainerRequest',
           foreign_key: 'container_uuid',
           primary_key: 'uuid'
  belongs_to :auth,
             class_name: 'ApiClientAuthorization',
             foreign_key: 'auth_uuid',
             primary_key: 'uuid',
             optional: true

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
    t.add :cost
    t.add :subrequests_cost
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
    update_priorities uuid
    reload
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
    if rc['keep_cache_disk'] == 0 and rc['keep_cache_ram'] == 0
      rc['keep_cache_disk'] = bound_keep_cache_disk(rc['ram'])
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

    resolved_runtime_constraints = resolve_runtime_constraints(attrs[:runtime_constraints])
    # Ideally we would completely ignore Keep cache constraints when making
    # reuse considerations, but our database structure makes that impractical.
    # The best we can do is generate a search that matches on all likely values.
    runtime_constraint_variations = {
      keep_cache_disk: [
        # Check for constraints without keep_cache_disk
        # (containers that predate the constraint)
        nil,
        # Containers that use keep_cache_ram instead
        0,
        # The default value
        bound_keep_cache_disk(resolved_runtime_constraints['ram']),
        # The minimum default bound
        bound_keep_cache_disk(0),
        # The maximum default bound (presumably)
        bound_keep_cache_disk(1 << 60),
        # The requested value
        resolved_runtime_constraints.delete('keep_cache_disk'),
      ].uniq,
      keep_cache_ram: [
        # Containers that use keep_cache_disk instead
        0,
        # The default value
        Rails.configuration.Containers.DefaultKeepCacheRAM,
        # The requested value
        resolved_runtime_constraints.delete('keep_cache_ram'),
      ].uniq,
    }
    resolved_cuda = resolved_runtime_constraints['cuda']
    if resolved_cuda.nil? or resolved_cuda['device_count'] == 0
      runtime_constraint_variations[:cuda] = [
        # Check for constraints without cuda
        # (containers that predate the constraint)
        nil,
        # The default "don't need CUDA" value
        {
          'device_count' => 0,
          'driver_version' => '',
          'hardware_capability' => '',
        },
        # The requested value
        resolved_runtime_constraints.delete('cuda')
      ].uniq
    end
    reusable_runtime_constraints = hash_product(**runtime_constraint_variations)
                                     .map { |v| resolved_runtime_constraints.merge(v) }

    candidates = candidates.where_serialized(:runtime_constraints, reusable_runtime_constraints, md5: true, multivalue: true)
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
              where("(runtime_status->'error') is null and priority > 0").
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
                       limit(1)
    if !attrs[:scheduling_parameters]['preemptible']
      locked_or_queued = locked_or_queued.
                           where("not ((scheduling_parameters::jsonb)->>'preemptible')::boolean")
    end
    chosen = locked_or_queued.first
    if chosen
      log_reuse_info { "done, reusing container #{chosen.uuid} with state=#{chosen.state}" }
      return chosen
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
      self.update!(state: Locked)
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
      self.update!(state: Queued)
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

  def self.bound_keep_cache_disk(value)
    value ||= 0
    min_value = 2 << 30
    max_value = 32 << 30
    if value < min_value
      min_value
    elsif value > max_value
      max_value
    else
      value
    end
  end

  def self.hash_product(**kwargs)
    # kwargs is a hash that maps parameters to an array of values.
    # This function enumerates every possible hash where each key has one of
    # the values from its array.
    # The output keys are strings since that's what container hash attributes
    # want.
    # A nil value yields a hash without that key.
    [[:_, nil]].product(
      *kwargs.map { |(key, values)| [key.to_s].product(values) },
    ).map { |param_pairs| Hash[param_pairs].compact }
  end

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
    final_attrs = [:finished_at]
    progress_attrs = [:progress, :runtime_status, :subrequests_cost, :cost,
                      :log, :output, :output_properties, :exit_code]

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
        permitted.push :finished_at, :log, :runtime_status, :cost
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
      ContainerRequest.where(container_uuid: self.uuid, state: ContainerRequest::Committed).each do |cr|
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
      self.auth.andand.update(expires_at: db_transaction_time)
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
            # Cancelled means the container didn't run to completion.
            # This happens either because it was cancelled by the user
            # or because there was an infrastructure failure.  We want
            # to retry infrastructure failures automatically.
            #
            # Seach for live container requests to determine if we
            # should retry the container.
            retryable_requests = ContainerRequest.
                                   joins('left outer join containers as requesting_container on container_requests.requesting_container_uuid = requesting_container.uuid').
                                   where("container_requests.container_uuid = ? and "+
                                         "container_requests.priority > 0 and "+
                                         "container_requests.owner_uuid not in (select group_uuid from trashed_groups) and "+
                                         "(requesting_container.priority is null or (requesting_container.state = 'Running' and requesting_container.priority > 0)) and "+
                                         "container_requests.state = 'Committed' and "+
                                         "container_requests.container_count < container_requests.container_count_max", uuid).
                                   order('container_requests.uuid asc')
          else
            retryable_requests = []
          end

          if retryable_requests.any?
            scheduling_parameters = {
              # partitions: empty if any are empty, else the union of all parameters
              "partitions": retryable_requests
                              .map { |req| req.scheduling_parameters["partitions"] || [] }
                              .reduce { |cur, new| (cur.empty? or new.empty?) ? [] : (cur | new) },

              # preemptible: true if all are true, else false
              "preemptible": retryable_requests
                               .map { |req| req.scheduling_parameters["preemptible"] }
                               .all?,

              # supervisor: true if all any true, else false
              "supervisor": retryable_requests
                               .map { |req| req.scheduling_parameters["supervisor"] }
                               .any?,

              # max_run_time: 0 if any are 0 (unlimited), else the maximum
              "max_run_time": retryable_requests
                                .map { |req| req.scheduling_parameters["max_run_time"] || 0 }
                                .reduce do |cur, new|
                if cur == 0 or new == 0
                  0
                elsif new > cur
                  new
                else
                  cur
                end
              end,
            }

            c_attrs = {
              command: self.command,
              cwd: self.cwd,
              environment: self.environment,
              output_path: self.output_path,
              container_image: self.container_image,
              mounts: self.mounts,
              runtime_constraints: self.runtime_constraints,
              scheduling_parameters: scheduling_parameters,
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
                  cr.cumulative_cost += self.cost + self.subrequests_cost
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
            where(requesting_container_uuid: uuid,
                  state: ContainerRequest::Committed).
            in_batches(of: 15).each_record do |cr|
            leave_modified_by_user_alone do
              cr.set_priority_zero
              container_state = Container.where(uuid: cr.container_uuid).pluck(:state).first
              if container_state == Container::Queued || container_state == Container::Locked
                # If the child container hasn't started yet, finalize the
                # child CR now instead of leaving it "on hold", i.e.,
                # Queued with priority 0.  (OTOH, if the child is already
                # running, leave it alone so it can get cancelled the
                # usual way, get a copy of the log collection, etc.)
                cr.update!(state: ContainerRequest::Final)
              end
            end
          end
        end
      end
    end
  end
end
