require 'arvados/keep'

class Collection < ArvadosModel
  extend DbCurrentTime
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  serialize :properties, Hash

  before_validation :check_encoding
  before_validation :check_signatures
  before_validation :strip_manifest_text_and_clear_replication_confirmed
  before_validation :set_portable_data_hash
  validate :ensure_hash_matches_manifest_text
  before_save :set_file_names

  # Query only undeleted collections by default.
  default_scope where("expires_at IS NULL or expires_at > CURRENT_TIMESTAMP")

  api_accessible :user, extend: :common do |t|
    t.add :name
    t.add :description
    t.add :properties
    t.add :portable_data_hash
    t.add :signed_manifest_text, as: :manifest_text
    t.add :replication_desired
    t.add :replication_confirmed
    t.add :replication_confirmed_at
  end

  LOCATOR_REGEXP = /^([[:xdigit:]]{32})(\+([[:digit:]]+))?(\+([[:upper:]][[:alnum:]+@_-]*))?$/

  def self.attributes_required_columns
    super.merge(
                # If we don't list manifest_text explicitly, the
                # params[:select] code gets confused by the way we
                # expose signed_manifest_text as manifest_text in the
                # API response, and never let clients select the
                # manifest_text column.
                'manifest_text' => ['manifest_text'],
                )
  end

  FILE_TOKEN = /^[[:digit:]]+:[[:digit:]]+:/
  def check_signatures
    return false if self.manifest_text.nil?

    return true if current_user.andand.is_admin

    # Provided the manifest_text hasn't changed materially since an
    # earlier validation, it's safe to pass this validation on
    # subsequent passes without checking any signatures. This is
    # important because the signatures have probably been stripped off
    # by the time we get to a second validation pass!
    computed_pdh = compute_pdh
    return true if @signatures_checked and @signatures_checked == computed_pdh

    if self.manifest_text_changed?
      # Check permissions on the collection manifest.
      # If any signature cannot be verified, raise PermissionDeniedError
      # which will return 403 Permission denied to the client.
      api_token = current_api_client_authorization.andand.api_token
      signing_opts = {
        api_token: api_token,
        now: db_current_time.to_i,
      }
      self.manifest_text.lines.each do |entry|
        entry.split[1..-1].each do |tok|
          if tok =~ FILE_TOKEN
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
          elsif Rails.configuration.permit_create_collection_with_unsigned_manifest
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

  def strip_manifest_text_and_clear_replication_confirmed
    if self.manifest_text_changed?
      in_old_manifest = {}
      self.class.munge_manifest_locators!(manifest_text_was) do |match|
        in_old_manifest[match[1]] = true
      end

      cleared_replication_confirmed = false

      # Remove any permission signatures from the manifest.
      self[:manifest_text] = self.class.munge_manifest_locators!(self[:manifest_text]) do |match|
        if not cleared_replication_confirmed  and not in_old_manifest[match[1]]
          self.replication_confirmed_at = nil
          self.replication_confirmed = nil
          cleared_replication_confirmed = true
        end
        self.class.locator_without_signature(match)
      end
    end
    true
  end

  def self.locator_without_signature match
    without_signature = match[1]
    without_signature += match[2] if match[2]
    if match[4]
      hints = match[4].split('+').reject { |hint| hint.start_with?("A") }
      without_signature += hints.join('+')
    end
    without_signature
  end

  def set_portable_data_hash
    if (portable_data_hash.nil? or
        portable_data_hash == "" or
        (manifest_text_changed? and !portable_data_hash_changed?))
      @need_pdh_validation = false
      self.portable_data_hash = compute_pdh
    elsif portable_data_hash_changed?
      @need_pdh_validation = true
      begin
        loc = Keep::Locator.parse!(self.portable_data_hash)
        loc.strip_hints!
        if loc.size
          self.portable_data_hash = loc.to_s
        else
          self.portable_data_hash = "#{loc.hash}+#{portable_manifest_text.bytesize}"
        end
      rescue ArgumentError => e
        errors.add(:portable_data_hash, "#{e}")
        return false
      end
    end
    true
  end

  def ensure_hash_matches_manifest_text
    return true unless manifest_text_changed? or portable_data_hash_changed?
    # No need verify it if :set_portable_data_hash just computed it!
    return true if not @need_pdh_validation
    expect_pdh = compute_pdh
    if expect_pdh != portable_data_hash
      errors.add(:portable_data_hash,
                 "does not match computed hash #{expect_pdh}")
      return false
    end
  end

  def set_file_names
    if self.manifest_text_changed?
      self.file_names = manifest_files
    end
    true
  end

  def manifest_files
    names = ''
    if self.manifest_text
      self.manifest_text.scan(/ \d+:\d+:(\S+)/) do |name|
        names << name.first.gsub('\040',' ') + "\n"
        break if names.length > 2**12
      end
    end

    if self.manifest_text and names.length < 2**12
      self.manifest_text.scan(/^\.\/(\S+)/m) do |stream_name|
        names << stream_name.first.gsub('\040',' ') + "\n"
        break if names.length > 2**12
      end
    end

    names[0,2**12]
  end

  def check_encoding
    if manifest_text.encoding.name == 'UTF-8' and manifest_text.valid_encoding?
      true
    else
      begin
        # If Ruby thinks the encoding is something else, like 7-bit
        # ASCII, but its stored bytes are equal to the (valid) UTF-8
        # encoding of the same string, we declare it to be a UTF-8
        # string.
        utf8 = manifest_text
        utf8.force_encoding Encoding::UTF_8
        if utf8.valid_encoding? and utf8 == manifest_text.encode(Encoding::UTF_8)
          manifest_text = utf8
          return true
        end
      rescue
      end
      errors.add :manifest_text, "must use UTF-8 encoding"
      false
    end
  end

  def signed_manifest_text
    if has_attribute? :manifest_text
      token = current_api_client_authorization.andand.api_token
      @signed_manifest_text = self.class.sign_manifest manifest_text, token
    end
  end

  def self.sign_manifest manifest, token
    signing_opts = {
      api_token: token,
      expire: db_current_time.to_i + Rails.configuration.blob_signature_ttl,
    }
    m = manifest.dup
    m = munge_manifest_locators!(m) do |match|
      Blob.sign_locator(locator_without_signature(match), signing_opts)
    end
    return m
  end

  def self.munge_manifest_locators! manifest
    # Given a manifest text and a block, yield each locator,
    # and replace it with whatever the block returns.
    new_lines = []
    lines = manifest.andand.split("\n")
    lines.andand.each do |line|
      words = line.split(' ')
      new_words = []
      words.each do |word|
        if match = LOCATOR_REGEXP.match(word.strip)
          new_words << yield(match)
        else
          new_words << word.strip
        end
      end
      new_lines << new_words.join(' ')
    end

    if !new_lines.empty?
      ends_with_newline = manifest.end_with?("\n")
      manifest = new_lines.join("\n")
      manifest += "\n" if ends_with_newline
    end

    manifest
  end

  def self.normalize_uuid uuid
    hash_part = nil
    size_part = nil
    uuid.split('+').each do |token|
      if token.match /^[0-9a-f]{32,}$/
        raise "uuid #{uuid} has multiple hash parts" if hash_part
        hash_part = token
      elsif token.match /^\d+$/
        raise "uuid #{uuid} has multiple size parts" if size_part
        size_part = token
      end
    end
    raise "uuid #{uuid} has no hash part" if !hash_part
    [hash_part, size_part].compact.join '+'
  end

  # Return array of Collection objects
  def self.find_all_for_docker_image(search_term, search_tag=nil, readers=nil)
    readers ||= [Thread.current[:user]]
    base_search = Link.
      readable_by(*readers).
      readable_by(*readers, table_name: "collections").
      joins("JOIN collections ON links.head_uuid = collections.uuid").
      order("links.created_at DESC")

    # If the search term is a Collection locator that contains one file
    # that looks like a Docker image, return it.
    if loc = Keep::Locator.parse(search_term)
      loc.strip_hints!
      coll_match = readable_by(*readers).where(portable_data_hash: loc.to_s).limit(1).first
      if coll_match
        # Check if the Collection contains exactly one file whose name
        # looks like a saved Docker image.
        manifest = Keep::Manifest.new(coll_match.manifest_text)
        if manifest.exact_file_count?(1) and
            (manifest.files[0][1] =~ /^[0-9A-Fa-f]{64}\.tar$/)
          return [coll_match]
        end
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
    matches.all.map do |link|
      uuid_timestamps[link.head_uuid] = [(-link.properties["image_timestamp"].to_datetime.to_i rescue 0),
       -link.created_at.to_i]
    end
    Collection.where('uuid in (?)', uuid_timestamps.keys).sort_by { |c| uuid_timestamps[c.uuid] }
  end

  def self.for_latest_docker_image(search_term, search_tag=nil, readers=nil)
    find_all_for_docker_image(search_term, search_tag, readers).first
  end

  def self.searchable_columns operator
    super - ["manifest_text"]
  end

  def self.full_text_searchable_columns
    super - ["manifest_text"]
  end

  protected
  def portable_manifest_text
    portable_manifest = self[:manifest_text].dup
    portable_manifest = self.class.munge_manifest_locators!(portable_manifest) do |match|
      if match[2] # size
        match[1] + match[2]
      else
        match[1]
      end
    end
    portable_manifest
  end

  def compute_pdh
    return @computed_pdh if @computed_pdh
    portable_manifest = portable_manifest_text
    @computed_pdh = (Digest::MD5.hexdigest(portable_manifest) +
                     '+' +
                     portable_manifest.bytesize.to_s)
    @computed_pdh
  end

  def ensure_permission_to_save
    if (not current_user.andand.is_admin and
        (replication_confirmed_at_changed? or replication_confirmed_changed?) and
        not (replication_confirmed_at.nil? and replication_confirmed.nil?))
      raise ArvadosModel::PermissionDeniedError.new("replication_confirmed and replication_confirmed_at attributes cannot be changed, except by setting both to nil")
    end
    super
  end
end
