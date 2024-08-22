# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class KeepServiceTest < ActiveSupport::TestCase
  test "non-admins cannot create services" do
    set_user_from_auth :active
    ks = KeepService.new
    assert_not_allowed do
      ks.save
    end
  end

  test "non-admins cannot update services" do
    set_user_from_auth :admin
    ks = KeepService.new
    assert(ks.save, "saving new service failed")
    set_user_from_auth :active
    ks.service_port = 64434
    assert_not_allowed do
      ks.save
    end
  end

  test "admins can create and update services" do
    set_user_from_auth :admin
    ks = KeepService.new
    assert(ks.save, "saving new service failed")
    ks.service_port = 64434
    assert(ks.save, "saving updated service failed")
  end
end
