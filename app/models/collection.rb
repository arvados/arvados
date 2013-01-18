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
end
