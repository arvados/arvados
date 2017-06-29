# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class LoginWorkflowTest < ActionDispatch::IntegrationTest
  test "default prompt to login is JSON" do
    post('/arvados/v1/specimens', {specimen: {}},
         {'HTTP_ACCEPT' => ''})
    assert_response 401
    assert_includes(json_response['errors'], "Not logged in")
  end

  test "login prompt respects JSON Accept header" do
    post('/arvados/v1/specimens', {specimen: {}},
         {'HTTP_ACCEPT' => 'application/json'})
    assert_response 401
    assert_includes(json_response['errors'], "Not logged in")
  end

  test "login prompt respects HTML Accept header" do
    post('/arvados/v1/specimens', {specimen: {}},
         {'HTTP_ACCEPT' => 'text/html'})
    assert_response 302
    assert_match(%r{/auth/joshid$}, @response.headers['Location'],
                 "HTML login prompt did not include expected redirect")
  end
end
