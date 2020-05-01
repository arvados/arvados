# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class TimeZoneTest < ActiveSupport::TestCase
  test "Database connection time zone" do
    # This is pointless if the testing host is already using the UTC
    # time zone.  But if not, the test confirms that
    # config/initializers/time_zone.rb has successfully changed the
    # database connection time zone to UTC.
    assert_equal('UTC', ActiveRecord::Base.connection.select_value("show timezone"))
  end
end
