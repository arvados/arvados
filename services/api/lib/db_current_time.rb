module DbCurrentTime
  CURRENT_TIME_SQL = "SELECT clock_timestamp()"

  def db_current_time
    Time.parse(ActiveRecord::Base.connection.select_value(CURRENT_TIME_SQL)).to_time
  end
end
