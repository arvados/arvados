class ActiveSupport::TimeWithZone
  def as_json *args
    strftime "%Y-%m-%dT%H:%M:%S.%NZ"
  end
end
