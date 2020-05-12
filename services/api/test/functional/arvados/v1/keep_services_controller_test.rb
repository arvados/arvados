# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::KeepServicesControllerTest < ActionController::TestCase

  test "search by service_port with < query" do
    authorize_with :active
    get :index, params: {
      filters: [['service_port', '<', 25107]]
    }
    assert_response :success
    assert_equal false, assigns(:objects).any?
  end

  test "search by service_port with >= query" do
    authorize_with :active
    get :index, params: {
      filters: [['service_port', '>=', 25107]]
    }
    assert_response :success
    assert_equal true, assigns(:objects).any?
  end

  [:admin, :active, :inactive, :anonymous, nil].each do |u|
    test "accessible to #{u.inspect} user" do
      authorize_with(u) if u
      get :accessible
      assert_response :success
      assert_not_empty json_response['items']
      json_response['items'].each do |ks|
        assert_not_equal ks['service_type'], 'proxy'
      end
    end
  end

  test "report configured servers if db is empty" do
    KeepService.unscoped.all.delete_all
    expect_rvz = {}
    n = 0
    Rails.configuration.Services.Keepstore.InternalURLs.each do |k,v|
      n += 1
      rvz = "%015x" % n
      expect_rvz[k.to_s] = rvz
      Rails.configuration.Services.Keepstore.InternalURLs[k].Rendezvous = rvz
    end
    expect_rvz[Rails.configuration.Services.Keepproxy.ExternalURL] = true
    refute_empty expect_rvz
    authorize_with :active
    get :index,
      params: {:format => :json},
      headers: auth(:active)
    assert_response :success
    json_response['items'].each do |svc|
      url = "#{svc['service_ssl_flag'] ? 'https' : 'http'}://#{svc['service_host']}:#{svc['service_port']}/"
      assert_equal true, expect_rvz.has_key?(url), "#{url} does not match any configured service: expecting #{expect_rvz}"
      rvz = expect_rvz[url]
      if rvz.is_a? String
        assert_equal "zzzzz-bi6l4-#{rvz}", svc['uuid'], "exported service UUID should match InternalURLs.*.Rendezvous value"
      end
      expect_rvz.delete(url)
    end
    assert_equal({}, expect_rvz, "all configured Keepstore and Keepproxy services should be returned")
  end

end
