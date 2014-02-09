class Collection < ArvadosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :user, extend: :common do |t|
    t.add :data_size
    t.add :files
  end

  api_accessible :with_data, extend: :user do |t|
    t.add :portable_manifest_text
    t.add :manifest_text
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
    # The client may provide either a portable or a non-portable
    # manifest or both, as long as the given UUID matches one of
    # them. If only one is provided, we make up something reasonable
    # for the other. This behavior allows the client to expect all
    # three of "uuid given == uuid stored in database", "uuid is
    # stable when non-portable manifest changes, if md5(portable
    # manifest) given", and "other clients see the +K@xyzzy hints I
    # provide until the api server takes charge of them".
    self.manifest_text ||= portable_manifest_text
    self.portable_manifest_text ||= manifest_text.andand.gsub /\+K@[a-z0-9]+/, ''

    if self.manifest_text.nil? and self.uuid.nil?
      super
    elsif self.manifest_text and self.uuid
      self.uuid.gsub! /\+.*/, ''
      if self.uuid == Digest::MD5.hexdigest(self.manifest_text)
        self.uuid.gsub! /$/, '+' + self.manifest_text.length.to_s
        true
      elsif self.uuid == Digest::MD5.hexdigest(self.portable_manifest_text)
        self.uuid.gsub! /$/, '+' + self.portable_manifest_text.length.to_s
        true
      else
        errors.add :uuid, 'does not match checksum of manifest_text or portable_manifest_text'
        false
      end
    elsif self.manifest_text
      errors.add :uuid, 'not supplied to match manifest_text'
      false
    else
      errors.add :manifest_text, 'not supplied'
      false
    end
  end

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
    @files = []
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
            @files << [stream, filename, re[2].to_i]
          end
        end
      end
    end
  end
end
