# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'whitelist_update'
require 'arvados/collection'

class ContainerRequest < ArvadosModel
  include ArvadosModelUpdates
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include WhitelistUpdate

  belongs_to :container, foreign_key: :container_uuid, primary_key: :uuid
  belongs_to :requesting_container, {
               class_name: 'Container',
               foreign_key: :requesting_container_uuid,
               primary_key: :uuid,
             }

  # Posgresql JSONB columns should NOT be declared as serialized, Rails 5
  # already know how to properly treat them.
  attribute :properties, :jsonbHash, default: {}
  attribute :secret_mounts, :jsonbHash, default: {}
  attribute :output_storage_classes, :jsonbArray, default: lambda { Rails.configuration.DefaultStorageClasses }
  attribute :output_properties, :jsonbHash, default: {}

  serialize :environment, Hash
  serialize :mounts, Hash
  serialize :runtime_constraints, Hash
  serialize :command, Array
  serialize :scheduling_parameters, Hash

  after_find :fill_container_defaults_after_find
  before_validation :fill_field_defaults, :if => :new_record?
  before_validation :fill_container_defaults
  validates :command, :container_image, :output_path, :cwd, :presence => true
  validates :output_ttl, numericality: { only_integer: true, greater_than_or_equal_to: 0 }
  validates :priority, numericality: { only_integer: true, greater_than_or_equal_to: 0, less_than_or_equal_to: 1000 }
  validate :validate_datatypes
  validate :validate_runtime_constraints
  validate :validate_scheduling_parameters
  validate :validate_state_change
  validate :check_update_whitelist
  validate :secret_mounts_key_conflict
  validate :validate_runtime_token
  after_validation :scrub_secrets
  after_validation :set_preemptible
  after_validation :set_container
  before_create :set_requesting_container_uuid
  before_destroy :set_priority_zero
  after_save :update_priority
  after_save :finalize_if_needed

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
    t.add :output_storage_classes
    t.add :output_properties
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

  AttrsPermittedAlways = [:owner_uuid, :state, :name, :description, :properties]
  AttrsPermittedBeforeCommit = [:command, :container_count_max,
  :container_image, :cwd, :environment, :filters, :mounts,
  :output_path, :priority, :runtime_token,
  :runtime_constraints, :state, :container_uuid, :use_existing,
  :scheduling_parameters, :secret_mounts, :output_name, :output_ttl,
  :output_storage_classes, :output_properties]

  def self.any_preemptible_instances?
    Rails.configuration.InstanceTypes.any? do |k, v|
      v["Preemptible"]
    end
  end

  def self.limit_index_columns_read
    ["mounts"]
  end

  def logged_attributes
    super.except('secret_mounts', 'runtime_token')
  end

  def state_transitions
    State_transitions
  end

  def skip_uuid_read_permission_check
    # The uuid_read_permission_check prevents users from making
    # references to objects they can't view.  However, in this case we
    # don't want to do that check since there's a circular dependency
    # where user can't view the container until the user has
    # constructed the container request that references the container.
    %w(container_uuid)
  end

  def finalize_if_needed
    return if state != Committed
    while true
      # get container lock first, then lock current container request
      # (same order as Container#handle_completed). Locking always
      # reloads the Container and ContainerRequest records.
      c = Container.find_by_uuid(container_uuid)
      c.lock! if !c.nil?
      self.lock!

      if !c.nil? && container_uuid != c.uuid
        # After locking, we've noticed a race, the container_uuid is
        # different than the container record we just loaded.  This
        # can happen if Container#handle_completed scheduled a new
        # container for retry and set container_uuid while we were
        # waiting on the container lock.  Restart the loop and get the
        # new container.
        redo
      end

      if !c.nil?
        if state == Committed && c.final?
          # The current container is
          act_as_system_user do
            leave_modified_by_user_alone do
              finalize!
            end
          end
        end
      elsif state == Committed
        # Behave as if the container is cancelled
        update_attributes!(state: Final)
      end
      return true
    end
  end

  # Finalize the container request after the container has
  # finished/cancelled.
  def finalize!
    container = Container.find_by_uuid(container_uuid)
    if !container.nil?
      update_collections(container: container)

      if container.state == Container::Complete
        log_col = Collection.where(portable_data_hash: container.log).first
        if log_col
          # Need to save collection
          completed_coll = Collection.new(
            owner_uuid: self.owner_uuid,
            name: "Container log for container #{container_uuid}",
            properties: {
              'type' => 'log',
              'container_request' => self.uuid,
              'container_uuid' => container_uuid,
            },
            portable_data_hash: log_col.portable_data_hash,
            manifest_text: log_col.manifest_text,
            storage_classes_desired: self.output_storage_classes
          )
          completed_coll.save_with_unique_name!
        end
      end
    end
    update_attributes!(state: Final)
  end

  def update_collections(container:, collections: ['log', 'output'])
    collections.each do |out_type|
      pdh = container.send(out_type)
      next if pdh.nil?
      c = Collection.where(portable_data_hash: pdh).first
      next if c.nil?
      manifest = c.manifest_text

      coll_name = "Container #{out_type} for request #{uuid}"
      trash_at = nil
      if out_type == 'output'
        if self.output_name and self.output_name != ""
          coll_name = self.output_name
        end
        if self.output_ttl > 0
          trash_at = db_current_time + self.output_ttl
        end
      end

      coll_uuid = self.send(out_type + '_uuid')
      coll = coll_uuid.nil? ? nil : Collection.where(uuid: coll_uuid).first
      if !coll
        coll = Collection.new(
          owner_uuid: self.owner_uuid,
          name: coll_name,
          manifest_text: "",
          storage_classes_desired: self.output_storage_classes)
      end

      if out_type == "log"
        # Copy the log into a merged collection
        src = Arv::Collection.new(manifest)
        dst = Arv::Collection.new(coll.manifest_text)
        dst.cp_r("./", ".", src)
        dst.cp_r("./", "log for container #{container.uuid}", src)
        manifest = dst.manifest_text
      end

      merged_properties = {}
      merged_properties['container_request'] = uuid

      if out_type == 'output' and !requesting_container_uuid.nil?
        # output of a child process, give it "intermediate" type by
        # default.
        merged_properties['type'] = 'intermediate'
      else
        merged_properties['type'] = out_type
      end

      if out_type == "output"
        merged_properties.update(container.output_properties)
        merged_properties.update(self.output_properties)
      end

      coll.assign_attributes(
        portable_data_hash: Digest::MD5.hexdigest(manifest) + '+' + manifest.bytesize.to_s,
        manifest_text: manifest,
        trash_at: trash_at,
        delete_at: trash_at,
        properties: merged_properties)
      coll.save_with_unique_name!
      self.send(out_type + '_uuid=', coll.uuid)
    end
  end

  def self.full_text_searchable_columns
    super - ["mounts", "secret_mounts", "secret_mounts_md5", "runtime_token", "output_storage_classes"]
  end

  protected

  def fill_field_defaults
    self.state ||= Uncommitted
    self.environment ||= {}
    self.runtime_constraints ||= {}
    self.mounts ||= {}
    self.secret_mounts ||= {}
    self.cwd ||= "."
    self.container_count_max ||= Rails.configuration.Containers.MaxRetryAttempts
    self.scheduling_parameters ||= {}
    self.output_ttl ||= 0
    self.priority ||= 0
  end

  def set_container
    if (container_uuid_changed? and
        not current_user.andand.is_admin and
        not container_uuid.nil?)
      errors.add :container_uuid, "can only be updated to nil."
      return false
    end
    if self.container_count_changed?
      errors.add :container_count, "cannot be updated directly."
      return false
    end
    if state_changed? and state == Committed and container_uuid.nil?
      while true
        c = Container.resolve(self)
        c.lock!
        if c.state == Container::Cancelled
          # Lost a race, we have a lock on the container but the
          # container was cancelled in a different request, restart
          # the loop and resolve request to a new container.
          redo
        end
        self.container_uuid = c.uuid
        break
      end
    end
    if self.container_uuid != self.container_uuid_was
      self.container_count += 1
      return if self.container_uuid_was.nil?

      old_container = Container.find_by_uuid(self.container_uuid_was)
      return if old_container.nil?

      old_logs = Collection.where(portable_data_hash: old_container.log).first
      return if old_logs.nil?

      log_coll = self.log_uuid.nil? ? nil : Collection.where(uuid: self.log_uuid).first
      if self.log_uuid.nil?
        log_coll = Collection.new(
          owner_uuid: self.owner_uuid,
          name: coll_name = "Container log for request #{uuid}",
          manifest_text: "",
          storage_classes_desired: self.output_storage_classes)
      end

      # copy logs from old container into CR's log collection
      src = Arv::Collection.new(old_logs.manifest_text)
      dst = Arv::Collection.new(log_coll.manifest_text)
      dst.cp_r("./", "log for container #{old_container.uuid}", src)
      manifest = dst.manifest_text

      log_coll.assign_attributes(
        portable_data_hash: Digest::MD5.hexdigest(manifest) + '+' + manifest.bytesize.to_s,
        manifest_text: manifest)
      log_coll.save_with_unique_name!
      self.log_uuid = log_coll.uuid
    end
  end

  def set_preemptible
    if (new_record? || state_changed?) &&
       state == Committed &&
       Rails.configuration.Containers.AlwaysUsePreemptibleInstances &&
       get_requesting_container_uuid() &&
       self.class.any_preemptible_instances?
      self.scheduling_parameters['preemptible'] = true
    end
  end

  def validate_runtime_constraints
    case self.state
    when Committed
      ['vcpus', 'ram'].each do |k|
        v = runtime_constraints[k]
        if !v.is_a?(Integer) || v <= 0
          errors.add(:runtime_constraints,
                     "[#{k}]=#{v.inspect} must be a positive integer")
        end
      end
      if runtime_constraints['cuda']
        ['device_count'].each do |k|
          v = runtime_constraints['cuda'][k]
          if !v.is_a?(Integer) || v < 0
            errors.add(:runtime_constraints,
                       "[cuda.#{k}]=#{v.inspect} must be a positive or zero integer")
          end
        end
        ['driver_version', 'hardware_capability'].each do |k|
          v = runtime_constraints['cuda'][k]
          if !v.is_a?(String) || (runtime_constraints['cuda']['device_count'] > 0 && v.to_f == 0.0)
            errors.add(:runtime_constraints,
                       "[cuda.#{k}]=#{v.inspect} must be a string in format 'X.Y'")
          end
        end
      end
    end
  end

  def validate_datatypes
    command.each do |c|
      if !c.is_a? String
        errors.add(:command, "must be an array of strings but has entry #{c.class}")
      end
    end
    environment.each do |k,v|
      if !k.is_a?(String) || !v.is_a?(String)
        errors.add(:environment, "must be an map of String to String but has entry #{k.class} to #{v.class}")
      end
    end
    [:mounts, :secret_mounts].each do |m|
      self[m].each do |k, v|
        if !k.is_a?(String) || !v.is_a?(Hash)
          errors.add(m, "must be an map of String to Hash but is has entry #{k.class} to #{v.class}")
        end
        if v["kind"].nil?
          errors.add(m, "each item must have a 'kind' field")
        end
        [[String, ["kind", "portable_data_hash", "uuid", "device_type",
                   "path", "commit", "repository_name", "git_url"]],
         [Integer, ["capacity"]]].each do |t, fields|
          fields.each do |f|
            if !v[f].nil? && !v[f].is_a?(t)
              errors.add(m, "#{k}: #{f} must be a #{t} but is #{v[f].class}")
            end
          end
        end
        ["writable", "exclude_from_output"].each do |f|
          if !v[f].nil? && !v[f].is_a?(TrueClass) && !v[f].is_a?(FalseClass)
            errors.add(m, "#{k}: #{f} must be a #{t} but is #{v[f].class}")
          end
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
      if scheduling_parameters['preemptible'] &&
         (new_record? || state_changed?) &&
         !self.class.any_preemptible_instances?
        errors.add :scheduling_parameters, "preemptible instances are not configured in InstanceTypes"
      end
      if scheduling_parameters.include? 'max_run_time' and
        (!scheduling_parameters['max_run_time'].is_a?(Integer) ||
          scheduling_parameters['max_run_time'] < 0)
          errors.add :scheduling_parameters, "max_run_time must be positive integer"
      end
    end
  end

  def check_update_whitelist
    permitted = AttrsPermittedAlways.dup

    if self.new_record? || self.state_was == Uncommitted
      # Allow create-and-commit in a single operation.
      permitted.push(*AttrsPermittedBeforeCommit)
    elsif mounts_changed? && mounts_was.keys.sort == mounts.keys.sort
      # Ignore the updated mounts if the only changes are default/zero
      # values as added by controller, see 17774
      only_defaults = true
      mounts.each do |path, mount|
        (mount.to_a - mounts_was[path].to_a).each do |k, v|
          if ![0, "", false, nil].index(v)
            only_defaults = false
          end
        end
      end
      if only_defaults
        clear_attribute_change("mounts")
      end
    end

    case self.state
    when Committed
      permitted.push :priority, :container_count_max, :container_uuid

      if self.priority.nil?
        self.errors.add :priority, "cannot be nil"
      end

      # Allow container count to increment (not by client, only by us
      # -- see set_container)
      permitted.push :container_count

      if current_user.andand.is_admin
        permitted.push :log_uuid
      end

    when Final
      if self.state_was == Committed
        # "Cancel" means setting priority=0, state=Committed
        permitted.push :priority

        if current_user.andand.is_admin
          permitted.push :output_uuid, :log_uuid
        end
      end

    end

    super(permitted)
  end

  def secret_mounts_key_conflict
    secret_mounts.each do |k, v|
      if mounts.has_key?(k)
        errors.add(:secret_mounts, 'conflict with non-secret mounts')
        return false
      end
    end
  end

  def validate_runtime_token
    if !self.runtime_token.nil? && self.runtime_token_changed?
      if !runtime_token[0..2] == "v2/"
        errors.add :runtime_token, "not a v2 token"
        return
      end
      if ApiClientAuthorization.validate(token: runtime_token).nil?
        errors.add :runtime_token, "failed validation"
      end
    end
  end

  def scrub_secrets
    if self.state == Final
      self.secret_mounts = {}
      self.runtime_token = nil
    end
  end

  def update_priority
    return unless saved_change_to_state? || saved_change_to_priority? || saved_change_to_container_uuid?
    act_as_system_user do
      Container.
        where('uuid in (?)', [container_uuid_before_last_save, self.container_uuid].compact).
        map(&:update_priority!)
    end
  end

  def set_priority_zero
    self.update_attributes!(priority: 0) if self.state != Final
  end

  def set_requesting_container_uuid
    if (self.requesting_container_uuid = get_requesting_container_uuid())
      # Determine the priority of container request for the requesting
      # container.
      self.priority = ContainerRequest.where(container_uuid: self.requesting_container_uuid).maximum("priority") || 0
    end
  end

  def get_requesting_container_uuid
    return self.requesting_container_uuid || Container.for_current_token.andand.uuid
  end
end
