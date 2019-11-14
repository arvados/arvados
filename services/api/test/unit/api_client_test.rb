# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ApiClientTest < ActiveSupport::TestCase
  include CurrentApiClient

  test "configured workbench is trusted" do
    Rails.configuration.Services.Workbench1.ExternalURL = URI("http://wb1.example.com")
    Rails.configuration.Services.Workbench2.ExternalURL = URI("https://wb2.example.com:443")

    act_as_system_user do
      [["http://wb0.example.com", false],
       ["http://wb1.example.com", true],
       ["http://wb2.example.com", false],
       ["https://wb2.example.com", true],
       ["https://wb2.example.com/", true],
      ].each do |pfx, result|
        a = ApiClient.create(url_prefix: pfx, is_trusted: false)
        assert_equal result, a.is_trusted
      end

      a = ApiClient.create(url_prefix: "http://example.com", is_trusted: true)
      a.save!
      a.reload
      assert a.is_trusted
    end
  end
end
