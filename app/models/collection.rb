class Collection < OrvosModel
  include AssignUuid
  include KindAndEtag
  include CommonApiTemplate

  api_accessible :superuser, :extend => :common do |t|
    t.add :locator
    t.add :portable_data_hash
    t.add :name
    t.add :redundancy
    t.add :redundancy_confirmed_by_client
    t.add :redundancy_confirmed_at
    t.add :redundancy_confirmed_as
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
    if self.manifest_text.nil? and self.uuid.nil?
      super
    elsif self.manifest_text and self.uuid
      if self.uuid == Digest::MD5.hexdigest(self.manifest_text)
        true
      else
        errors.add :uuid, 'uuid does not match checksum of manifest_text'
        false
      end
    elsif self.manifest_text
      errors.add :uuid, 'checksum for manifest_text not supplied in uuid'
      false
    else
      errors.add :manifest_text, 'manifest_text not supplied'
      false
    end
  end
end
