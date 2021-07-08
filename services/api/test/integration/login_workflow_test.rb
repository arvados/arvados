# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class LoginWorkflowTest < ActionDispatch::IntegrationTest
  test "default prompt to login is JSON" do
    post('/arvados/v1/specimens',
      params: {specimen: {}},
      headers: {'HTTP_ACCEPT' => ''})
    assert_response 401
    json_response['errors'].each do |err|
      assert(err.include?("Not logged in"), "error message '#{err}' expected to include 'Not logged in'")
    end
  end

  test "login prompt respects JSON Accept header" do
    post('/arvados/v1/specimens',
      params: {specimen: {}},
      headers: {'HTTP_ACCEPT' => 'application/json'})
    assert_response 401
    json_response['errors'].each do |err|
      assert(err.include?("Not logged in"), "error message '#{err}' expected to include 'Not logged in'")
    end
  end

  test "login prompt respects HTML Accept header" do
    post('/arvados/v1/specimens',
      params: {specimen: {}},
      headers: {'HTTP_ACCEPT' => 'text/html'})
    assert_response 302
    assert_match(%r{http://www.example.com/login$}, @response.headers['Location'],
                 "HTML login prompt did not include expected redirect")
  end
end
