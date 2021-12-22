# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

ActiveRecord::ConnectionAdapters::AbstractAdapter.set_callback :checkout, :before, ->(conn) do
  ms = Rails.configuration.API.RequestTimeout.to_i * 1000
  conn.execute("SET statement_timeout = #{ms}")
  conn.execute("SET lock_timeout = #{ms}")
end
