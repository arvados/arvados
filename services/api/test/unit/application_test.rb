# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ApplicationTest < ActiveSupport::TestCase
  include CurrentApiClient

  test "act_as_system_user" do
    Thread.current[:user] = users(:active)
    assert_equal users(:active), Thread.current[:user]
    act_as_system_user do
      assert_not_equal users(:active), Thread.current[:user]
      assert_equal system_user, Thread.current[:user]
    end
    assert_equal users(:active), Thread.current[:user]
  end

  test "act_as_system_user is exception safe" do
    Thread.current[:user] = users(:active)
    assert_equal users(:active), Thread.current[:user]
    caught = false
    begin
      act_as_system_user do
        assert_not_equal users(:active), Thread.current[:user]
        assert_equal system_user, Thread.current[:user]
        raise "Fail"
      end
    rescue
      caught = true
    end
    assert caught
    assert_equal users(:active), Thread.current[:user]
  end

  test "config maps' keys are returned as symbols" do
    assert Rails.configuration.Users.AutoSetupUsernameBlacklist.is_a? ActiveSupport::OrderedOptions
    assert Rails.configuration.Users.AutoSetupUsernameBlacklist.keys.size > 0
    Rails.configuration.Users.AutoSetupUsernameBlacklist.keys.each do |k|
      assert k.is_a? Symbol
    end
  end
end
