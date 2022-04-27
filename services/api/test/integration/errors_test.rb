# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ErrorsTest < ActionDispatch::IntegrationTest
  fixtures :api_client_authorizations

  %w(/arvados/v1/shoes /arvados/shoes /shoes /nodes /users).each do |path|
    test "non-existent route #{path}" do
      get path, params: {:format => :json}, headers: auth(:active)
      assert_nil assigns(:objects)
      assert_nil assigns(:object)
      assert_not_nil json_response['errors']
      assert_response 404
      assert_match /^req-[0-9a-zA-Z]{20}$/, response.headers['X-Request-Id']
    end
  end

  n=0
  Rails.application.routes.routes.each do |route|
    test "route #{n += 1} '#{route.path.spec.to_s}' is not an accident" do
      # Generally, new routes should appear under /arvados/v1/. If
      # they appear elsewhere, that might have been caused by default
      # rails generator behavior that we don't want.
      assert_match(/^\/(|\*a|arvados\/v1\/.*|auth\/.*|login|logout|database\/reset|discovery\/.*|static\/.*|sys\/trash_sweep|themes\/.*|assets|_health\/.*|metrics)(\(\.:format\))?$/,
                   route.path.spec.to_s,
                   "Unexpected new route: #{route.path.spec}")
    end
  end

  test "X-Request-Id header" do
    get "/", headers: auth(:spectator)
    assert_match /^req-[0-9a-zA-Z]{20}$/, response.headers['X-Request-Id']
  end

  test "X-Request-Id header on non-existant object URL" do
    get "/arvados/v1/container_requests/invalid",
      params: {:format => :json}, headers: auth(:active)
    assert_response 404
    assert_match /^req-[0-9a-zA-Z]{20}$/, response.headers['X-Request-Id']
  end

  # The response header is the one that gets logged, so this test also
  # ensures we log the ID supplied in the request, if any.
  test "X-Request-Id given by client" do
    get "/", headers: auth(:spectator).merge({'X-Request-Id': 'abcdefG'})
    assert_equal 'abcdefG', response.headers['X-Request-Id']
  end

  test "X-Request-Id given by client is ignored if too long" do
    authorize_with :spectator
    long_reqId = 'abcdefG' * 1000
    get "/", headers: auth(:spectator).merge({'X-Request-Id': long_reqId})
    assert_match /^req-[0-9a-zA-Z]{20}$/, response.headers['X-Request-Id']
  end
end
