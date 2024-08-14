# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class NoopDeepMungeTest < ActionDispatch::IntegrationTest
  test "empty array" do
    check({"foo" => []})
  end

  test "null in array" do
    check({"foo" => ["foo", nil]})
  end

  test "array of nulls" do
    check({"foo" => [nil, nil, nil]})
  end

  protected

  def check(val)
    post "/arvados/v1/container_requests",
      params: {
        :container_request => {
          :name => "workflow",
          :state => "Uncommitted",
          :command => ["echo"],
          :container_image => "arvados/jobs",
          :output_path => "/",
          :mounts => {
            :foo => {
              :kind => "json",
              :content => JSON.parse(SafeJSON.dump(val)),
            }
          }
        }
      }.to_json,
      headers: {
        'HTTP_AUTHORIZATION' => "Bearer #{api_client_authorizations(:admin).api_token}",
        'CONTENT_TYPE' => 'application/json'
      }
    assert_response :success
    assert_equal "arvados#containerRequest", json_response['kind']
    assert_equal val, json_response['mounts']['foo']['content']
  end
end
