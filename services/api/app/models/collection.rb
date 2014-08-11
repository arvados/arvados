class Collection < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :user, extend: :common do |t|
    t.add :data_size
    t.add :files
    t.add :manifest_text
  end

  def self.attributes_required_columns
    super.merge({ "data_size" => ["manifest_text"],
                  "files" => ["manifest_text"],
                })
  end

  def redundancy_status
    if redundancy_confirmed_as.nil?
      'unconfirmed'
    elsif redundancy_confirmed_as < redundancy
      'degraded'
    else
      if redundancy_confirmed_at.nil?
        'unconfirmed'
      elsif Time.now - redundancy_confirmed_at < 7.days
        'OK'
      else
        'stale'
      end
    end
  end

  def assign_uuid
    if not self.manifest_text
      errors.add :manifest_text, 'not supplied'
      return false
    end
    expect_uuid = Digest::MD5.hexdigest(self.manifest_text)
    if self.uuid
      self.uuid.gsub! /\+.*/, ''
      if self.uuid != expect_uuid
        errors.add :uuid, 'must match checksum of manifest_text'
        return false
      end
    else
      self.uuid = expect_uuid
    end
    self.uuid.gsub! /$/, '+' + self.manifest_text.length.to_s
    true
  end

  # TODO (#3036/tom) replace above assign_uuid method with below assign_uuid and self.generate_uuid
  # def assign_uuid
  #   # Even admins cannot assign collection uuids.
  #   self.uuid = self.class.generate_uuid
  # end
  # def self.generate_uuid
  #   # The last 10 characters of a collection uuid are the last 10
  #   # characters of the base-36 SHA256 digest of manifest_text.
  #   [Server::Application.config.uuid_prefix,
  #    self.uuid_prefix,
  #    rand(2**256).to_s(36)[-5..-1] + Digest::SHA256.hexdigest(self.manifest_text).to_i(16).to_s(36)[-10..-1],
  #   ].join '-'
  # end

  def data_size
    inspect_manifest_text if @data_size.nil? or manifest_text_changed?
    @data_size
  end

  def files
    inspect_manifest_text if @files.nil? or manifest_text_changed?
    @files
  end

  def inspect_manifest_text
    if !manifest_text
      @data_size = false
      @files = []
      return
    end

    @data_size = 0
    tmp = {}

    manifest_text.split("\n").each do |stream|
      toks = stream.split(" ")

      stream = toks[0].gsub /\\(\\|[0-7]{3})/ do |escape_sequence|
        case $1
        when '\\' '\\'
        else $1.to_i(8).chr
        end
      end

      toks[1..-1].each do |tok|
        if (re = tok.match /^[0-9a-f]{32}/)
          blocksize = nil
          tok.split('+')[1..-1].each do |hint|
            if !blocksize and hint.match /^\d+$/
              blocksize = hint.to_i
            end
            if (re = hint.match /^GS(\d+)$/)
              blocksize = re[1].to_i
            end
          end
          @data_size = false if !blocksize
          @data_size += blocksize if @data_size
        else
          if (re = tok.match /^(\d+):(\d+):(\S+)$/)
            filename = re[3].gsub /\\(\\|[0-7]{3})/ do |escape_sequence|
              case $1
              when '\\' '\\'
              else $1.to_i(8).chr
              end
            end
            fn = stream + '/' + filename
            i = re[2].to_i
            if tmp[fn]
              tmp[fn] += i
            else
              tmp[fn] = i
            end
          end
        end
      end
    end

    @files = []
    tmp.each do |k, v|
      re = k.match(/^(.+)\/(.+)/)
      @files << [re[1], re[2], v]
    end
  end

  def self.uuid_like_pattern
    "________________________________+%"
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

  def self.uuids_for_docker_image(search_term, search_tag=nil, readers=nil)
    readers ||= [Thread.current[:user]]
    base_search = Link.
      readable_by(*readers).
      readable_by(*readers, table_name: "collections").
      joins("JOIN collections ON links.head_uuid = collections.uuid").
      order("links.created_at DESC")

    # If the search term is a Collection locator that contains one file
    # that looks like a Docker image, return it.
    if loc = Locator.parse(search_term)
      loc.strip_hints!
      coll_match = readable_by(*readers).where(uuid: loc.to_s).first
      if coll_match and (coll_match.files.size == 1) and
          (coll_match.files[0][1] =~ /^[0-9A-Fa-f]{64}\.tar$/)
        return [loc.to_s]
      end
    end

    # Find Collections with matching Docker image repository+tag pairs.
    matches = base_search.
      where(link_class: "docker_image_repo+tag",
            name: "#{search_term}:#{search_tag || 'latest'}")

    # If that didn't work, find Collections with matching Docker image hashes.
    if matches.empty?
      matches = base_search.
        where("link_class = ? and name LIKE ?",
              "docker_image_hash", "#{search_term}%")
    end

    # Generate an order key for each result.  We want to order the results
    # so that anything with an image timestamp is considered more recent than
    # anything without; then we use the link's created_at as a tiebreaker.
    uuid_timestamps = {}
    matches.find_each do |link|
      uuid_timestamps[link.head_uuid] =
        [(-link.properties["image_timestamp"].to_datetime.to_i rescue 0),
         -link.created_at.to_i]
    end
    uuid_timestamps.keys.sort_by { |uuid| uuid_timestamps[uuid] }
  end

  def self.for_latest_docker_image(search_term, search_tag=nil, readers=nil)
    image_uuid = uuids_for_docker_image(search_term, search_tag, readers).first
    if image_uuid.nil?
      nil
    else
      find_by_uuid(image_uuid)
    end
  end
end
