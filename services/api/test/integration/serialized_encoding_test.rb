# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class SerializedEncodingTest < ActionDispatch::IntegrationTest
  fixtures :all

  {
    api_client_authorization: {scopes: []},
    link: {link_class: 'test', name: 'test', properties: {foo: :bar}},
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
