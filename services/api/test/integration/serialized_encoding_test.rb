# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/git_test_helper'

class SerializedEncodingTest < ActionDispatch::IntegrationTest
  include GitTestHelper

  fixtures :all

  {
    api_client_authorization: {scopes: []},

    human: {properties: {eye_color: 'gray'}},

    link: {link_class: 'test', name: 'test', properties: {foo: :bar}},

    node: {info: {uptime: 1234}},

    specimen: {properties: {eye_color: 'meringue'}},

    trait: {properties: {eye_color: 'brown'}},

    user: {prefs: {cookies: 'thin mint'}},
  }.each_pair do |resource, postdata|
    test "create json-encoded #{resource.to_s}" do
      post("/arvados/v1/#{resource.to_s.pluralize}",
        params: {resource => postdata.to_json},
        headers: auth(:admin_trustedclient))
      assert_response :success
    end
  end
end
