# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'arvados/keep'
require 'sweep_trashed_objects'
require 'trashable'

class Collection < ArvadosModel
  extend CurrentApiClient
  extend DbCurrentTime
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  include Trashable

  # Posgresql JSONB columns should NOT be declared as serialized, Rails 5
  # already know how to properly treat them.
  attribute :properties, :jsonbHash, default: {}
  attribute :storage_classes_desired, :jsonbArray, default: ["default"]
  attribute :storage_classes_confirmed, :jsonbArray, default: []

  before_validation :default_empty_manifest
  before_validation :default_storage_classes, on: :create
  before_validation :managed_properties, on: :create
  before_validation :check_encoding
  before_validation :check_manifest_validity
  before_validation :check_signatures
  before_validation :strip_signatures_and_update_replication_confirmed
  before_validation :name_null_if_empty
  validate :ensure_filesystem_compatible_name
  validate :ensure_pdh_matches_manifest_text
  validate :ensure_storage_classes_desired_is_not_empty
  validate :ensure_storage_classes_contain_non_empty_strings
  validate :versioning_metadata_updates, on: :update
  validate :past_versions_cannot_be_updated, on: :update
  validate :protected_managed_properties_updates, on: :update
  after_validation :set_file_count_and_total_size
  before_save :set_file_names
  around_update :manage_versioning, unless: :is_past_version?

  api_accessible :user, extend: :common do |t|
    t.add lambda { |x| x.name || "" }, as: :name
    t.add :description
    t.add :properties
    t.add :portable_data_hash
    t.add :signed_manifest_text, as: :manifest_text
    t.add :manifest_text, as: :unsigned_manifest_text
    t.add :replication_desired
    t.add :replication_confirmed
    t.add :replication_confirmed_at
    t.add :storage_classes_desired
    t.add :storage_classes_confirmed
    t.add :storage_classes_confirmed_at
    t.add :delete_at
    t.add :trash_at
    t.add :is_trashed
    t.add :version
    t.add :current_version_uuid
    t.add :preserve_version
    t.add :file_count
    t.add :file_size_total
  end

  UNLOGGED_CHANGES = ['preserve_version', 'updated_at']

  after_initialize do
    @signatures_checked = false
    @computed_pdh_for_manifest_text = false
  end

  def self.attributes_required_columns
    super.merge(
                # If we don't list manifest_text explicitly, the
                # params[:select] code gets confused by the way we
                # expose signed_manifest_text as manifest_text in the
                # API response, and never let clients select the
                # manifest_text column.
                #
                # We need trash_at and is_trashed to determine the
                # correct timestamp in signed_manifest_text.
                'manifest_text' => ['manifest_text', 'trash_at', 'is_trashed'],
                'unsigned_manifest_text' => ['manifest_text'],
                'name' => ['name'],
                )
  end

  def self.ignored_select_attributes
    super + ["updated_at", "file_names"]
  end

  def self.limit_index_columns_read
    ["manifest_text"]
  end

  FILE_TOKEN = /^[[:digit:]]+:[[:digit:]]+:/
  def check_signatures
    throw(:abort) if self.manifest_text.nil?

    return true if current_user.andand.is_admin

    # Provided the manifest_text hasn't changed materially since an
    # earlier validation, it's safe to pass this validation on
    # subsequent passes without checking any signatures. This is
    # important because the signatures have probably been stripped off
    # by the time we get to a second validation pass!
    if @signatures_checked && @signatures_checked == computed_pdh
      return true
    end

    if self.manifest_text_changed?
      # Check permissions on the collection manifest.
      # If any signature cannot be verified, raise PermissionDeniedError
      # which will return 403 Permission denied to the client.
      api_token = Thread.current[:token]
      signing_opts = {
        api_token: api_token,
        now: @validation_timestamp.to_i,
      }
      self.manifest_text.each_line do |entry|
        entry.split.each do |tok|
          if tok == '.' or tok.starts_with? './'
            # Stream name token.
          elsif tok =~ FILE_TOKEN
            # This is a filename token, not a blob locator. Note that we
            # keep checking tokens after this, even though manifest
            # format dictates that all subsequent tokens will also be
            # filenames. Safety first!
          elsif Blob.verify_signature tok, signing_opts
            # OK.
          elsif Keep::Locator.parse(tok).andand.signature
            # Signature provided, but verify_signature did not like it.
            logger.warn "Invalid signature on locator #{tok}"
            raise ArvadosModel::PermissionDeniedError
          elsif !Rails.configuration.Collections.BlobSigning
            # No signature provided, but we are running in insecure mode.
            logger.debug "Missing signature on locator #{tok} ignored"
          elsif Blob.new(tok).empty?
            # No signature provided -- but no data to protect, either.
          else
            logger.warn "Missing signature on locator #{tok}"
            raise ArvadosModel::PermissionDeniedError
          end
        end
      end
    end
    @signatures_checked = computed_pdh
  end

  def strip_signatures_and_update_replication_confirmed
    if self.manifest_text_changed?
      in_old_manifest = {}
      # manifest_text_was could be nil when dealing with a freshly created snapshot,
      # so we skip this case because there was no real manifest change. (Bug #18005)
      if (not self.replication_confirmed.nil?) and (not self.manifest_text_was.nil?)
        self.class.each_manifest_locator(manifest_text_was) do |match|
          in_old_manifest[match[1]] = true
        end
      end

      stripped_manifest = self.class.munge_manifest_locators(manifest_text) do |match|
        if not self.replication_confirmed.nil? and not in_old_manifest[match[1]]
          # If the new manifest_text contains locators whose hashes
          # weren't in the old manifest_text, storage replication is no
          # longer confirmed.
          self.replication_confirmed_at = nil
          self.replication_confirmed = nil
        end

        # Return the locator with all permission signatures removed,
        # but otherwise intact.
        match[0].gsub(/\+A[^+]*/, '')
      end

      if @computed_pdh_for_manifest_text == manifest_text
        # If the cached PDH was valid before stripping, it is still
        # valid after stripping.
        @computed_pdh_for_manifest_text = stripped_manifest.dup
      end

      self[:manifest_text] = stripped_manifest
    end
    true
  end

  def ensure_pdh_matches_manifest_text
    if not manifest_text_changed? and not portable_data_hash_changed?
      true
    elsif portable_data_hash.nil? or not portable_data_hash_changed?
      self.portable_data_hash = computed_pdh
    elsif portable_data_hash !~ Keep::Locator::LOCATOR_REGEXP
      errors.add(:portable_data_hash, "is not a valid locator")
      false
    elsif portable_data_hash[0..31] != computed_pdh[0..31]
      errors.add(:portable_data_hash,
                 "'#{portable_data_hash}' does not match computed hash '#{computed_pdh}'")
      false
    else
      # Ignore the client-provided size part: always store
      # computed_pdh in the database.
      self.portable_data_hash = computed_pdh
    end
  end

  def name_null_if_empty
    if name == ""
      self.name = nil
    end
  end

  def set_file_names
    if self.manifest_text_changed?
      self.file_names = manifest_files
    end
    true
  end

  def set_file_count_and_total_size
    # Only update the file stats if the manifest changed
    if self.manifest_text_changed?
      m = Keep::Manifest.new(self.manifest_text)
      self.file_size_total = m.files_size
      self.file_count = m.files_count
    # If the manifest didn't change but the attributes did, ignore the changes
    elsif self.file_count_changed? || self.file_size_total_changed?
      self.file_count = self.file_count_was
      self.file_size_total = self.file_size_total_was
    end
    true
  end

  def manifest_files
    return '' if !self.manifest_text

    done = {}
    names = ''
    self.manifest_text.scan(/ \d+:\d+:(\S+)/) do |name|
      next if done[name]
      done[name] = true
      names << name.first.gsub('\040',' ') + "\n"
    end
    self.manifest_text.scan(/^\.\/(\S+)/m) do |stream_name|
      next if done[stream_name]
      done[stream_name] = true
      names << stream_name.first.gsub('\040',' ') + "\n"
    end
    names
  end

  def default_empty_manifest
    self.manifest_text ||= ''
  end

  def skip_uuid_existence_check
    # Avoid checking the existence of current_version_uuid, as it's
    # assigned on creation of a new 'current version' collection, so
    # the collection's UUID only lives on memory when the validation check
    # is performed.
    ['current_version_uuid']
  end

  def manage_versioning
    should_preserve_version = should_preserve_version? # Time sensitive, cache value
    return(yield) unless (should_preserve_version || syncable_updates.any?)

    # Put aside the changes because with_lock does a record reload
    changes = self.changes
    snapshot = nil
    restore_attributes
    with_lock do
      # Copy the original state to save it as old version
      if should_preserve_version
        snapshot = self.dup
        snapshot.uuid = nil # Reset UUID so it's created as a new record
        snapshot.created_at = self.created_at
        snapshot.modified_at = self.modified_at_was
      end

      # Restore requested changes on the current version
      changes.keys.each do |attr|
        if attr == 'preserve_version' && changes[attr].last == false && !should_preserve_version
          next # Ignore false assignment, once true it'll be true until next version
        end
        self.attributes = {attr => changes[attr].last}
        if attr == 'uuid'
          # Also update the current version reference
          self.attributes = {'current_version_uuid' => changes[attr].last}
        end
      end

      if should_preserve_version
        self.version += 1
      end

      yield

      sync_past_versions if syncable_updates.any?
      if snapshot
        snapshot.attributes = self.syncable_updates
        leave_modified_by_user_alone do
          leave_modified_at_alone do
            act_as_system_user do
              snapshot.save
            end
          end
        end
      end
    end
  end

  def maybe_update_modified_by_fields
    if !(self.changes.keys - ['updated_at', 'preserve_version']).empty?
      super
    end
  end

  def syncable_updates
    updates = {}
    if self.changes.any?
      changes = self.changes
    else
      # If called after save...
      changes = self.saved_changes
    end
    (syncable_attrs & changes.keys).each do |attr|
      if attr == 'uuid'
        # Point old versions to current version's new UUID
        updates['current_version_uuid'] = changes[attr].last
      else
        updates[attr] = changes[attr].last
      end
    end
    return updates
  end

  def sync_past_versions
    updates = self.syncable_updates
    Collection.where('current_version_uuid = ? AND uuid != ?', self.uuid_before_last_save, self.uuid_before_last_save).each do |c|
      c.attributes = updates
      # Use a different validation context to skip the 'past_versions_cannot_be_updated'
      # validator, as on this case it is legal to update some fields.
      leave_modified_by_user_alone do
        leave_modified_at_alone do
          c.save(context: :update_old_versions)
        end
      end
    end
  end

  def versionable_updates?(attrs)
    (['manifest_text', 'description', 'properties', 'name'] & attrs).any?
  end

  def syncable_attrs
    ['uuid', 'owner_uuid', 'delete_at', 'trash_at', 'is_trashed', 'replication_desired', 'storage_classes_desired']
  end

  def is_past_version?
    # Check for the '_was' values just in case the update operation
    # includes a change on current_version_uuid or uuid.
    !(new_record? || self.current_version_uuid_was == self.uuid_was)
  end

  def should_preserve_version?
    return false unless (Rails.configuration.Collections.CollectionVersioning && versionable_updates?(self.changes.keys))

    return false if self.is_trashed

    idle_threshold = Rails.configuration.Collections.PreserveVersionIfIdle
    if !self.preserve_version_was &&
      !self.preserve_version &&
      (idle_threshold < 0 ||
        (idle_threshold > 0 && self.modified_at_was > db_current_time-idle_threshold.seconds))
      return false
    end
    return true
  end

  def check_encoding
    if !(manifest_text.encoding.name == 'UTF-8' and manifest_text.valid_encoding?)
      begin
        # If Ruby thinks the encoding is something else, like 7-bit
        # ASCII, but its stored bytes are equal to the (valid) UTF-8
        # encoding of the same string, we declare it to be a UTF-8
        # string.
        utf8 = manifest_text
        utf8.force_encoding Encoding::UTF_8
        if utf8.valid_encoding? and utf8 == manifest_text.encode(Encoding::UTF_8)
          self.manifest_text = utf8
          return true
        end
      rescue
      end
      errors.add :manifest_text, "must use UTF-8 encoding"
      throw(:abort)
    end
  end

  def check_manifest_validity
    begin
      Keep::Manifest.validate! manifest_text
      true
    rescue ArgumentError => e
      errors.add :manifest_text, e.message
      throw(:abort)
    end
  end

  def signed_manifest_text
    if !has_attribute? :manifest_text
      return nil
    elsif is_trashed
      return manifest_text
    else
      token = Thread.current[:token]
      exp = [db_current_time.to_i + Rails.configuration.Collections.BlobSigningTTL.to_i,
             trash_at].compact.map(&:to_i).min
      self.class.sign_manifest manifest_text, token, exp
    end
  end

  def self.sign_manifest manifest, token, exp=nil
    if exp.nil?
      exp = db_current_time.to_i + Rails.configuration.Collections.BlobSigningTTL.to_i
    end
    signing_opts = {
      api_token: token,
      expire: exp,
    }
    m = munge_manifest_locators(manifest) do |match|
      Blob.sign_locator(match[0], signing_opts)
    end
    return m
  end

  def self.munge_manifest_locators manifest
    # Given a manifest text and a block, yield the regexp MatchData
    # for each locator. Return a new manifest in which each locator
    # has been replaced by the block's return value.
    return nil if !manifest
    return '' if manifest == ''

    new_lines = []
    manifest.each_line do |line|
      line.rstrip!
      new_words = []
      line.split(' ').each do |word|
        if new_words.empty?
          new_words << word
        elsif match = Keep::Locator::LOCATOR_REGEXP.match(word)
          new_words << yield(match)
        else
          new_words << word
        end
      end
      new_lines << new_words.join(' ')
    end
    new_lines.join("\n") + "\n"
  end

  def self.each_manifest_locator manifest
    # Given a manifest text and a block, yield the regexp match object
    # for each locator.
    manifest.each_line do |line|
      # line will have a trailing newline, but the last token is never
      # a locator, so it's harmless here.
      line.split(' ').each do |word|
        if match = Keep::Locator::LOCATOR_REGEXP.match(word)
          yield(match)
        end
      end
    end
  end

  def self.normalize_uuid uuid
    hash_part = nil
    size_part = nil
    uuid.split('+').each do |token|
      if token.match(/^[0-9a-f]{32,}$/)
        raise "uuid #{uuid} has multiple hash parts" if hash_part
        hash_part = token
      elsif token.match(/^\d+$/)
        raise "uuid #{uuid} has multiple size parts" if size_part
        size_part = token
      end
    end
    raise "uuid #{uuid} has no hash part" if !hash_part
    [hash_part, size_part].compact.join '+'
  end

  def self.get_compatible_images(readers, pattern, collections)
    if collections.empty?
      return []
    end

    migrations = Hash[
      Link.where('tail_uuid in (?) AND link_class=? AND links.owner_uuid=?',
                 collections.map(&:portable_data_hash),
                 'docker_image_migration',
                 system_user_uuid).
      order('links.created_at asc').
      map { |l|
        [l.tail_uuid, l.head_uuid]
      }]

    migrated_collections = Hash[
      Collection.readable_by(*readers).
      where('portable_data_hash in (?)', migrations.values).
      map { |c|
        [c.portable_data_hash, c]
      }]

    collections.map { |c|
      # Check if the listed image is compatible first, if not, then try the
      # migration link.
      manifest = Keep::Manifest.new(c.manifest_text)
      if manifest.exact_file_count?(1) and manifest.files[0][1] =~ pattern
        c
      elsif m = migrated_collections[migrations[c.portable_data_hash]]
        manifest = Keep::Manifest.new(m.manifest_text)
        if manifest.exact_file_count?(1) and manifest.files[0][1] =~ pattern
          m
        end
      end
    }.compact
  end

  # Resolve a Docker repo+tag, hash, or collection PDH to an array of
  # Collection objects, sorted by timestamp starting with the most recent
  # match.
  #
  # If filter_compatible_format is true (the default), only return image
  # collections which are support by the installation as indicated by
  # Rails.configuration.Containers.SupportedDockerImageFormats.  Will follow
  # 'docker_image_migration' links if search_term resolves to an incompatible
  # image, but an equivalent compatible image is available.
  def self.find_all_for_docker_image(search_term, search_tag=nil, readers=nil, filter_compatible_format: true)
    readers ||= [Thread.current[:user]]
    base_search = Link.
      readable_by(*readers).
      readable_by(*readers, table_name: "collections").
      joins("JOIN collections ON links.head_uuid = collections.uuid").
      order("links.created_at DESC")

    docker_image_formats = Rails.configuration.Containers.SupportedDockerImageFormats.keys.map(&:to_s)

    if (docker_image_formats.include? 'v1' and
        docker_image_formats.include? 'v2') or filter_compatible_format == false
      pattern = /^(sha256:)?[0-9A-Fa-f]{64}\.tar$/
    elsif docker_image_formats.include? 'v2'
      pattern = /^(sha256:)[0-9A-Fa-f]{64}\.tar$/
    elsif docker_image_formats.include? 'v1'
      pattern = /^[0-9A-Fa-f]{64}\.tar$/
    else
      raise "Unrecognized configuration for docker_image_formats #{docker_image_formats}"
    end

    # If the search term is a Collection locator that contains one file
    # that looks like a Docker image, return it.
    if loc = Keep::Locator.parse(search_term)
      loc.strip_hints!
      coll_match = readable_by(*readers).where(portable_data_hash: loc.to_s).limit(1)
      rc = Rails.configuration.RemoteClusters.select{ |k|
        k != :"*" && k != Rails.configuration.ClusterID}
      if coll_match.any? or rc.length == 0
        return get_compatible_images(readers, pattern, coll_match)
      else
        # Allow bare pdh that doesn't exist in the local database so
        # that federated container requests which refer to remotely
        # stored containers will validate.
        return [Collection.new(portable_data_hash: loc.to_s)]
      end
    end

    if search_tag.nil? and (n = search_term.index(":"))
      search_tag = search_term[n+1..-1]
      search_term = search_term[0..n-1]
    end

    # Find Collections with matching Docker image repository+tag pairs.
    matches = base_search.
      where(link_class: "docker_image_repo+tag",
            name: "#{search_term}:#{search_tag || 'latest'}")

    # If that didn't work, find Collections with matching Docker image hashes.
    if matches.empty?
      matches = base_search.
        where("link_class = ? and links.name LIKE ?",
              "docker_image_hash", "#{search_term}%")
    end

    # Generate an order key for each result.  We want to order the results
    # so that anything with an image timestamp is considered more recent than
    # anything without; then we use the link's created_at as a tiebreaker.
    uuid_timestamps = {}
    matches.each do |link|
      uuid_timestamps[link.head_uuid] = [(-link.properties["image_timestamp"].to_datetime.to_i rescue 0),
       -link.created_at.to_i]
     end

    sorted = Collection.where('uuid in (?)', uuid_timestamps.keys).sort_by { |c|
      uuid_timestamps[c.uuid]
    }
    compatible = get_compatible_images(readers, pattern, sorted)
    if sorted.length > 0 and compatible.empty?
      raise ArvadosModel::UnresolvableContainerError.new "Matching Docker image is incompatible with 'docker_image_formats' configuration."
    end
    compatible
  end

  def self.for_latest_docker_image(search_term, search_tag=nil, readers=nil)
    find_all_for_docker_image(search_term, search_tag, readers).first
  end

  def self.searchable_columns operator
    super - ["manifest_text"]
  end

  def self.full_text_searchable_columns
    super - ["manifest_text", "storage_classes_desired", "storage_classes_confirmed", "current_version_uuid"]
  end

  def self.where *args
    SweepTrashedObjects.sweep_if_stale
    super
  end

  protected

  # Although the defaults for these columns is already set up on the schema,
  # collection creation from an API client seems to ignore them, making the
  # validation on empty desired storage classes return an error.
  def default_storage_classes
    if self.storage_classes_desired.nil? || self.storage_classes_desired.empty?
      self.storage_classes_desired = ["default"]
    end
    self.storage_classes_confirmed ||= []
  end

  # Sets managed properties at creation time
  def managed_properties
    managed_props = Rails.configuration.Collections.ManagedProperties.with_indifferent_access
    if managed_props.empty?
      return
    end
    (managed_props.keys - self.properties.keys).each do |key|
      if managed_props[key]['Function'] == 'original_owner'
        self.properties[key] = self.user_owner_uuid
      elsif managed_props[key]['Value']
        self.properties[key] = managed_props[key]['Value']
      else
        logger.warn "Unidentified default property definition '#{key}': #{managed_props[key].inspect}"
      end
    end
  end

  def portable_manifest_text
    self.class.munge_manifest_locators(manifest_text) do |match|
      if match[2] # size
        match[1] + match[2]
      else
        match[1]
      end
    end
  end

  def compute_pdh
    portable_manifest = portable_manifest_text
    (Digest::MD5.hexdigest(portable_manifest) +
     '+' +
     portable_manifest.bytesize.to_s)
  end

  def computed_pdh
    if @computed_pdh_for_manifest_text == manifest_text
      return @computed_pdh
    end
    @computed_pdh = compute_pdh
    @computed_pdh_for_manifest_text = manifest_text.dup
    @computed_pdh
  end

  def ensure_permission_to_save
    if (not current_user.andand.is_admin)
      if (replication_confirmed_at_changed? or replication_confirmed_changed?) and
        not (replication_confirmed_at.nil? and replication_confirmed.nil?)
        raise ArvadosModel::PermissionDeniedError.new("replication_confirmed and replication_confirmed_at attributes cannot be changed, except by setting both to nil")
      end
      if (storage_classes_confirmed_changed? or storage_classes_confirmed_at_changed?) and
        not (storage_classes_confirmed == [] and storage_classes_confirmed_at.nil?)
        raise ArvadosModel::PermissionDeniedError.new("storage_classes_confirmed and storage_classes_confirmed_at attributes cannot be changed, except by setting them to [] and nil respectively")
      end
    end
    super
  end

  def ensure_storage_classes_desired_is_not_empty
    if self.storage_classes_desired.empty?
      raise ArvadosModel::InvalidStateTransitionError.new("storage_classes_desired shouldn't be empty")
    end
  end

  def ensure_storage_classes_contain_non_empty_strings
    (self.storage_classes_desired + self.storage_classes_confirmed).each do |c|
      if !c.is_a?(String) || c == ''
        raise ArvadosModel::InvalidStateTransitionError.new("storage classes should only be non-empty strings")
      end
    end
  end

  def past_versions_cannot_be_updated
    if is_past_version?
      errors.add(:base, "past versions cannot be updated")
      false
    end
  end

  def protected_managed_properties_updates
    managed_properties = Rails.configuration.Collections.ManagedProperties.with_indifferent_access
    if managed_properties.empty? || !properties_changed? || current_user.is_admin
      return true
    end
    protected_props = managed_properties.keys.select do |p|
      Rails.configuration.Collections.ManagedProperties[p]['Protected']
    end
    # Pre-existent protected properties can't be updated
    invalid_updates = properties_was.keys.select{|p| properties_was[p] != properties[p]} & protected_props
    if !invalid_updates.empty?
      invalid_updates.each do |p|
        errors.add("protected property cannot be updated:", p)
      end
      raise PermissionDeniedError.new
    end
    true
  end

  def versioning_metadata_updates
    valid = true
    if !is_past_version? && current_version_uuid_changed?
      errors.add(:current_version_uuid, "cannot be updated")
      valid = false
    end
    if version_changed?
      errors.add(:version, "cannot be updated")
      valid = false
    end
    valid
  end

  def assign_uuid
    super
    self.current_version_uuid ||= self.uuid
    true
  end

  def log_update
    super unless (saved_changes.keys - UNLOGGED_CHANGES).empty?
  end
end
