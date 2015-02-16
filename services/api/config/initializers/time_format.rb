class ActiveSupport::TimeWithZone
  def as_json
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end
