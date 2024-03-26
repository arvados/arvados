# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module DbCurrentTime
  CURRENT_TIME_SQL = "SELECT clock_timestamp() AT TIME ZONE 'UTC'"

  def db_current_time
    ActiveRecord::Base.connection.select_value(CURRENT_TIME_SQL)
  end

  def db_transaction_time
    ActiveRecord::Base.connection.select_value("SELECT current_timestamp AT TIME ZONE 'UTC'")
  end
end
