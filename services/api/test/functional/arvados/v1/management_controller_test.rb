# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::ManagementControllerTest < ActionController::TestCase
  [
    [false, nil, 404, 'disabled'],
    [true, nil, 401, 'authorization required'],
    [true, 'badformatwithnoBearer', 403, 'authorization error'],
    [true, 'Bearer wrongtoken', 403, 'authorization error'],
    [true, 'Bearer configuredmanagementtoken', 200, '{"health":"OK"}'],
  ].each do |enabled, header, error_code, error_msg|
    test "_health/ping when #{if enabled then 'enabled' else 'disabled' end} with header '#{header}'" do
      if enabled
        Rails.configuration.ManagementToken = 'configuredmanagementtoken'
      else
        Rails.configuration.ManagementToken = ""
      end

      @request.headers['Authorization'] = header
      get :health, params: {check: 'ping'}
      assert_response error_code

      resp = JSON.parse(@response.body)
      if error_code == 200
        assert_equal(JSON.load('{"health":"OK"}'), resp)
      else
        assert_equal(error_msg, resp['errors'])
      end
    end
  end

  test "metrics" do
    mtime = File.mtime(ENV["ARVADOS_CONFIG"])
    hash = Digest::SHA256.hexdigest(File.read(ENV["ARVADOS_CONFIG"]))
    Rails.configuration.ManagementToken = "configuredmanagementtoken"
    @request.headers['Authorization'] = "Bearer configuredmanagementtoken"
    get :metrics
    assert_response :success
    assert_equal 'text/plain', @response.content_type

    assert_match /\narvados_config_source_timestamp_seconds{sha256="#{hash}"} #{Regexp.escape mtime.utc.to_f.to_s}\n/, @response.body

    # Expect mtime < loadtime < now
    m = @response.body.match(/\narvados_config_load_timestamp_seconds{sha256="#{hash}"} (.*?)\n/)
    assert_operator m[1].to_f, :>, mtime.utc.to_f
    assert_operator m[1].to_f, :<, Time.now.utc.to_f
  end

  test "metrics disabled" do
    Rails.configuration.ManagementToken = ""
    @request.headers['Authorization'] = "Bearer configuredmanagementtoken"
    get :metrics
    assert_response 404
  end

  test "metrics bad token" do
    Rails.configuration.ManagementToken = "configuredmanagementtoken"
    @request.headers['Authorization'] = "Bearer asdf"
    get :metrics
    assert_response 403
  end

  test "metrics unauthorized" do
    Rails.configuration.ManagementToken = "configuredmanagementtoken"
    get :metrics
    assert_response 401
  end
end
