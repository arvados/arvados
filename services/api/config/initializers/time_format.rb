class ActiveSupport::TimeWithZone
  def as_json *args
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end

class Time
  def as_json *args
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end
