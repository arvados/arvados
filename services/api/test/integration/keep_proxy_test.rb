# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class KeepProxyTest < ActionDispatch::IntegrationTest
  test "request keep disks" do
    get "/arvados/v1/keep_services/accessible",
      params: {:format => :json},
      headers: auth(:active)
    assert_response :success
    services = json_response['items']

    assert_equal Rails.configuration.Services.Keepstore.InternalURLs.length, services.length
    services.each do |service|
      assert_equal 'disk', service['service_type']
    end
  end

  test "request keep proxy" do
    get "/arvados/v1/keep_services/accessible",
      params: {:format => :json},
      headers: auth(:active).merge({'HTTP_X_EXTERNAL_CLIENT' => '1'})
    assert_response :success
    services = json_response['items']

    assert_equal 1, services.length

    scheme = "http#{'s' if services[0]['service_ssl_flag']}"
    host = services[0]['service_host']
    port = services[0]['service_port']
    assert_equal Rails.configuration.Services.Keepproxy.ExternalURL.to_s, "#{scheme}://#{host}:#{port}/"
    assert_equal 'proxy', services[0]['service_type']
  end
end
