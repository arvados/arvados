# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ActiveRecord::ConnectionAdapters::AbstractAdapter.set_callback :checkout, :before, ->(conn) do
  # If the database connection is in a time zone other than UTC,
  # "timestamp" values don't behave as desired.
  #
  # For example, ['select now() > ?', Time.now] returns true in time
  # zones +0100 and UTC (which makes sense since Time.now is evaluated
  # before now()), but false in time zone -0100 (now() returns an
  # earlier clock time, and its time zone is dropped when comparing to
  # a "timestamp without time zone").
  conn.execute("SET TIME ZONE 'UTC'")
end
