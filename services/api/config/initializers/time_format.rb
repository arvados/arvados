class ActiveSupport::TimeWithZone
  remove_method :as_json
  def as_json *args
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end

class Time
  remove_method :as_json
  def as_json *args
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end
