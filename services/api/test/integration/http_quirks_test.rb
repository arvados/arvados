# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class HttpQuirksTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "GET request with empty Content-Type header" do
    authorize_with :active
    get "/arvados/v1/collections",
        headers: auth(:active).merge("Content-Type" => "")
    assert_response :success
  end
end
