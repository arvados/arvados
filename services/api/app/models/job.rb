# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0
#
#
# Legacy jobs API aka crunch v1
#
# This is superceded by containers / container_requests (aka crunch v2)
#
# Arvados installations since the end of 2017 should have never
# used jobs, and are unaffected by this change.
#
# So that older Arvados sites don't lose access to legacy records, the
# API has been converted to read-only.  Creating and updating jobs
# (and related types job_task, pipeline_template and
# pipeline_instance) is disabled and much of the business logic
# related has been removed, along with the crunch-dispatch.rb and
# various other code specific to the jobs API.
#
# If you need to resurrect any of this code, here is the last commit
# on master before the branch removing jobs API support:
#
# Wed Aug 7 14:49:38 2019 -0400 07d92519438a592d531f2c7558cd51788da262ca

require 'log_reuse_info'
require 'safe_json'

class Job < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  extend CurrentApiClient
  extend LogReuseInfo
  serialize :components, Hash
  serialize :script_parameters, Hash
  serialize :runtime_constraints, Hash
  serialize :tasks_summary, Hash
  before_create :ensure_unique_submit_id
  before_validation :set_priority
  before_validation :update_state_from_old_state_attrs
  before_validation :update_script_parameters_digest
  validate :ensure_script_version_is_commit
  validate :find_docker_image_locator
  validate :find_arvados_sdk_version
  validate :validate_status
  validate :validate_state_change
  validate :ensure_no_collection_uuids_in_script_params
  before_save :tag_version_in_internal_repository
  before_save :update_timestamps_when_state_changes
  before_create :create_disabled
  before_update :update_disabled

  has_many(:nodes, foreign_key: :job_uuid, primary_key: :uuid)

  class SubmitIdReused < RequestError
  end

  api_accessible :user, extend: :common do |t|
    t.add :submit_id
    t.add :priority
    t.add :script
    t.add :script_parameters
    t.add :script_version
    t.add :cancelled_at
    t.add :cancelled_by_client_uuid
    t.add :cancelled_by_user_uuid
    t.add :started_at
    t.add :finished_at
    t.add :output
    t.add :success
    t.add :running
    t.add :state
    t.add :is_locked_by_uuid
    t.add :log
    t.add :runtime_constraints
    t.add :tasks_summary
    t.add :nondeterministic
    t.add :repository
    t.add :supplied_script_version
    t.add :arvados_sdk_version
    t.add :docker_image_locator
    t.add :queue_position
    t.add :node_uuids
    t.add :description
    t.add :components
  end

  # Supported states for a job
  States = [
            (Queued = 'Queued'),
            (Running = 'Running'),
            (Cancelled = 'Cancelled'),
            (Failed = 'Failed'),
            (Complete = 'Complete'),
           ]

  after_initialize do
    @need_crunch_dispatch_trigger = false
  end

  def self.limit_index_columns_read
    ["components"]
  end

  def self.protected_attributes
    [:arvados_sdk_version, :docker_image_locator]
  end

  def assert_finished
    update(finished_at: finished_at || db_current_time,
                      success: success.nil? ? false : success,
                      running: false)
  end

  def node_uuids
    nodes.map(&:uuid)
  end

  def self.queue
    self.where('state = ?', Queued).order('priority desc, created_at')
  end

  def queue_position
    # We used to report this accurately, but the implementation made queue
    # API requests O(n**2) for the size of the queue.  See #8800.
    # We've soft-disabled it because it's not clear we even want this
    # functionality: now that we have Node Manager with support for multiple
    # node sizes, "queue position" tells you very little about when a job will
    # run.
    state == Queued ? 0 : nil
  end

  def self.running
    self.where('running = ?', true).
      order('priority desc, created_at')
  end

  def lock locked_by_uuid
    with_lock do
      unless self.state == Queued and self.is_locked_by_uuid.nil?
        raise AlreadyLockedError
      end
      self.state = Running
      self.is_locked_by_uuid = locked_by_uuid
      self.save!
    end
  end

  def update_script_parameters_digest
    self.script_parameters_digest = self.class.sorted_hash_digest(script_parameters)
  end

  def self.searchable_columns operator
    super - ["script_parameters_digest"]
  end

  def self.full_text_searchable_columns
    super - ["script_parameters_digest"]
  end

  def self.load_job_specific_filters attrs, orig_filters, read_users
    # Convert Job-specific @filters entries into general SQL filters.
    script_info = {"repository" => nil, "script" => nil}
    git_filters = Hash.new do |hash, key|
      hash[key] = {"max_version" => "HEAD", "exclude_versions" => []}
    end
    filters = []
    orig_filters.each do |attr, operator, operand|
      if (script_info.has_key? attr) and (operator == "=")
        if script_info[attr].nil?
          script_info[attr] = operand
        elsif script_info[attr] != operand
          raise ArgumentError.new("incompatible #{attr} filters")
        end
      end
      case operator
      when "in git"
        git_filters[attr]["min_version"] = operand
      when "not in git"
        git_filters[attr]["exclude_versions"] += Array.wrap(operand)
      when "in docker", "not in docker"
        image_hashes = Array.wrap(operand).flat_map do |search_term|
          image_search, image_tag = search_term.split(':', 2)
          Collection.
            find_all_for_docker_image(image_search, image_tag, read_users, filter_compatible_format: false).
            map(&:portable_data_hash)
        end
        filters << [attr, operator.sub(/ docker$/, ""), image_hashes]
      else
        filters << [attr, operator, operand]
      end
    end

    # Build a real script_version filter from any "not? in git" filters.
    git_filters.each_pair do |attr, filter|
      case attr
      when "script_version"
        script_info.each_pair do |key, value|
          if value.nil?
            raise ArgumentError.new("script_version filter needs #{key} filter")
          end
        end
        filter["repository"] = script_info["repository"]
        if attrs[:script_version]
          filter["max_version"] = attrs[:script_version]
        else
          # Using HEAD, set earlier by the hash default, is fine.
        end
      when "arvados_sdk_version"
        filter["repository"] = "arvados"
      else
        raise ArgumentError.new("unknown attribute for git filter: #{attr}")
      end
      revisions = CommitsHelper::find_commit_range(filter["repository"],
                                           filter["min_version"],
                                           filter["max_version"],
                                           filter["exclude_versions"])
      if revisions.empty?
        raise ArgumentError.
          new("error searching #{filter['repository']} from " +
              "'#{filter['min_version']}' to '#{filter['max_version']}', " +
              "excluding #{filter['exclude_versions']}")
      end
      filters.append([attr, "in", revisions])
    end

    filters
  end

  def self.default_git_filters(attr_name, repo_name, refspec)
    # Add a filter to @filters for `attr_name` = the latest commit available
    # in `repo_name` at `refspec`.  No filter is added if refspec can't be
    # resolved.
    commits = CommitsHelper::find_commit_range(repo_name, nil, refspec, nil)
    if commit_hash = commits.first
      [[attr_name, "=", commit_hash]]
    else
      []
    end
  end

  def cancel(cascade: false, need_transaction: true)
    raise "No longer supported"
  end

  protected

  def self.sorted_hash_digest h
    Digest::MD5.hexdigest(Oj.dump(deep_sort_hash(h)))
  end

  def foreign_key_attributes
    super + %w(output log)
  end

  def skip_uuid_read_permission_check
    super + %w(cancelled_by_client_uuid)
  end

  def skip_uuid_existence_check
    super + %w(output log)
  end

  def set_priority
    if self.priority.nil?
      self.priority = 0
    end
    true
  end

  def ensure_script_version_is_commit
    if state == Running
      # Apparently client has already decided to go for it. This is
      # needed to run a local job using a local working directory
      # instead of a commit-ish.
      return true
    end
    if new_record? or repository_changed? or script_version_changed?
      sha1 = CommitsHelper::find_commit_range(repository,
                                      nil, script_version, nil).first
      if not sha1
        errors.add :script_version, "#{script_version} does not resolve to a commit"
        return false
      end
      if supplied_script_version.nil? or supplied_script_version.empty?
        self.supplied_script_version = script_version
      end
      self.script_version = sha1
    end
    true
  end

  def tag_version_in_internal_repository
    if state == Running
      # No point now. See ensure_script_version_is_commit.
      true
    elsif errors.any?
      # Won't be saved, and script_version might not even be valid.
      true
    elsif new_record? or repository_changed? or script_version_changed?
      uuid_was = uuid
      begin
        assign_uuid
        CommitsHelper::tag_in_internal_repository repository, script_version, uuid
      rescue
        self.uuid = uuid_was
        raise
      end
    end
  end

  def ensure_unique_submit_id
    if !submit_id.nil?
      if Job.where('submit_id=?',self.submit_id).first
        raise SubmitIdReused.new
      end
    end
    true
  end

  def resolve_runtime_constraint(key, attr_sym)
    if ((runtime_constraints.is_a? Hash) and
        (search = runtime_constraints[key]))
      ok, result = yield search
    else
      ok, result = true, nil
    end
    if ok
      send("#{attr_sym}=".to_sym, result)
    else
      errors.add(attr_sym, result)
    end
    ok
  end

  def find_arvados_sdk_version
    resolve_runtime_constraint("arvados_sdk_version",
                               :arvados_sdk_version) do |git_search|
      commits = CommitsHelper::find_commit_range("arvados",
                                         nil, git_search, nil)
      if commits.empty?
        [false, "#{git_search} does not resolve to a commit"]
      elsif not runtime_constraints["docker_image"]
        [false, "cannot be specified without a Docker image constraint"]
      else
        [true, commits.first]
      end
    end
  end

  def find_docker_image_locator
    if runtime_constraints.is_a? Hash and Rails.configuration.Containers.JobsAPI.DefaultDockerImage != ""
      runtime_constraints['docker_image'] ||=
        Rails.configuration.Containers.JobsAPI.DefaultDockerImage
    end

    resolve_runtime_constraint("docker_image",
                               :docker_image_locator) do |image_search|
      image_tag = runtime_constraints['docker_image_tag']
      if coll = Collection.for_latest_docker_image(image_search, image_tag)
        [true, coll.portable_data_hash]
      else
        [false, "not found for #{image_search}"]
      end
    end
  end

  def permission_to_update
    if is_locked_by_uuid_was and !(current_user and
                                   (current_user.uuid == is_locked_by_uuid_was or
                                    current_user.uuid == system_user.uuid))
      if script_changed? or
          script_parameters_changed? or
          script_version_changed? or
          (!cancelled_at_was.nil? and
           (cancelled_by_client_uuid_changed? or
            cancelled_by_user_uuid_changed? or
            cancelled_at_changed?)) or
          started_at_changed? or
          finished_at_changed? or
          running_changed? or
          success_changed? or
          output_changed? or
          log_changed? or
          tasks_summary_changed? or
          (state_changed? && state != Cancelled) or
          components_changed?
        logger.warn "User #{current_user.uuid if current_user} tried to change protected job attributes on locked #{self.class.to_s} #{uuid_was}"
        return false
      end
    end
    if !is_locked_by_uuid_changed?
      super
    else
      if !current_user
        logger.warn "Anonymous user tried to change lock on #{self.class.to_s} #{uuid_was}"
        false
      elsif is_locked_by_uuid_was and is_locked_by_uuid_was != current_user.uuid
        logger.warn "User #{current_user.uuid} tried to steal lock on #{self.class.to_s} #{uuid_was} from #{is_locked_by_uuid_was}"
        false
      elsif !is_locked_by_uuid.nil? and is_locked_by_uuid != current_user.uuid
        logger.warn "User #{current_user.uuid} tried to lock #{self.class.to_s} #{uuid_was} with uuid #{is_locked_by_uuid}"
        false
      else
        super
      end
    end
  end

  def update_modified_by_fields
    if self.cancelled_at_changed?
      # Ensure cancelled_at cannot be set to arbitrary non-now times,
      # or changed once it is set.
      if self.cancelled_at and not self.cancelled_at_was
        self.cancelled_at = db_current_time
        self.cancelled_by_user_uuid = current_user.uuid
        self.cancelled_by_client_uuid = current_api_client.andand.uuid
        @need_crunch_dispatch_trigger = true
      else
        self.cancelled_at = self.cancelled_at_was
        self.cancelled_by_user_uuid = self.cancelled_by_user_uuid_was
        self.cancelled_by_client_uuid = self.cancelled_by_client_uuid_was
      end
    end
    super
  end

  def update_timestamps_when_state_changes
    return if not (state_changed? or new_record?)

    case state
    when Running
      self.started_at ||= db_current_time
    when Failed, Complete
      self.finished_at ||= db_current_time
    when Cancelled
      self.cancelled_at ||= db_current_time
    end

    # TODO: Remove the following case block when old "success" and
    # "running" attrs go away. Until then, this ensures we still
    # expose correct success/running flags to older clients, even if
    # some new clients are writing only the new state attribute.
    case state
    when Queued
      self.running = false
      self.success = nil
    when Running
      self.running = true
      self.success = nil
    when Cancelled, Failed
      self.running = false
      self.success = false
    when Complete
      self.running = false
      self.success = true
    end
    self.running ||= false # Default to false instead of nil.

    @need_crunch_dispatch_trigger = true

    true
  end

  def update_state_from_old_state_attrs
    # If a client has touched the legacy state attrs, update the
    # "state" attr to agree with the updated values of the legacy
    # attrs.
    #
    # TODO: Remove this method when old "success" and "running" attrs
    # go away.
    if cancelled_at_changed? or
        success_changed? or
        running_changed? or
        state.nil?
      if cancelled_at
        self.state = Cancelled
      elsif success == false
        self.state = Failed
      elsif success == true
        self.state = Complete
      elsif running == true
        self.state = Running
      else
        self.state = Queued
      end
    end
    true
  end

  def validate_status
    if self.state.in?(States)
      true
    else
      errors.add :state, "#{state.inspect} must be one of: #{States.inspect}"
      false
    end
  end

  def validate_state_change
    ok = true
    if self.state_changed?
      ok = case self.state_was
           when nil
             # state isn't set yet
             true
           when Queued
             # Permit going from queued to any state
             true
           when Running
             # From running, may only transition to a finished state
             [Complete, Failed, Cancelled].include? self.state
           when Complete, Failed, Cancelled
             # Once in a finished state, don't permit any more state changes
             false
           else
             # Any other state transition is also invalid
             false
           end
      if not ok
        errors.add :state, "invalid change from #{self.state_was} to #{self.state}"
      end
    end
    ok
  end

  def ensure_no_collection_uuids_in_script_params
    # Fail validation if any script_parameters field includes a string containing a
    # collection uuid pattern.
    if self.script_parameters_changed?
      if recursive_hash_search(self.script_parameters, Collection.uuid_regex)
        self.errors.add :script_parameters, "must use portable_data_hash instead of collection uuid"
        return false
      end
    end
    true
  end

  # recursive_hash_search searches recursively through hashes and
  # arrays in 'thing' for string fields matching regular expression
  # 'pattern'.  Returns true if pattern is found, false otherwise.
  def recursive_hash_search thing, pattern
    if thing.is_a? Hash
      thing.each do |k, v|
        return true if recursive_hash_search v, pattern
      end
    elsif thing.is_a? Array
      thing.each do |k|
        return true if recursive_hash_search k, pattern
      end
    elsif thing.is_a? String
      return true if thing.match pattern
    end
    false
  end

  def create_disabled
    raise "Disabled"
  end

  def update_disabled
    raise "Disabled"
  end
end
