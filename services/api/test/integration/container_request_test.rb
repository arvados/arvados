# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ContainerRequestIntegrationTest < ActionDispatch::IntegrationTest

  test "test colon in input" do
    # Tests for bug #15311 where strings with leading colons get
    # corrupted when the leading ":" is stripped.
    val = {"itemSeparator" => ":"}
    post "/arvados/v1/container_requests",
      params: {
        :container_request => {
          :name => "workflow",
          :state => "Committed",
          :command => ["echo"],
          :container_image => "fa3c1a9cb6783f85f2ecda037e07b8c3+167",
          :output_path => "/",
          :priority => 1,
          :runtime_constraints => {"vcpus" => 1, "ram" => 1},
          :mounts => {
            :foo => {
              :kind => "json",
              :content => JSON.parse(SafeJSON.dump(val)),
            }
          }
        }
      }.to_json,
      headers: {
        'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:active).api_token}",
        'CONTENT_TYPE' => 'application/json'
      }
    assert_response :success
    assert_equal "arvados#containerRequest", json_response['kind']
    assert_equal val, json_response['mounts']['foo']['content']
  end
end
