# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module DbCurrentTime
  CURRENT_TIME_SQL = "SELECT clock_timestamp()"

  def db_current_time
    Time.parse(ActiveRecord::Base.connection.select_value(CURRENT_TIME_SQL)).to_time
  end
end
