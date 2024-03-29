# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ApiClientTest < ActiveSupport::TestCase
  include CurrentApiClient

  [true, false].each do |token_lifetime_enabled|
    test "configured workbench is trusted when token lifetime is#{token_lifetime_enabled ? '': ' not'} enabled" do
      Rails.configuration.Login.TokenLifetime = token_lifetime_enabled ? 8.hours : 0
      Rails.configuration.Login.IssueTrustedTokens = !token_lifetime_enabled;
      Rails.configuration.Services.Workbench1.ExternalURL = URI("http://wb1.example.com")
      Rails.configuration.Services.Workbench2.ExternalURL = URI("https://wb2.example.com:443")
      Rails.configuration.Login.TrustedClients = ActiveSupport::OrderedOptions.new
      Rails.configuration.Login.TrustedClients[:"https://wb3.example.com"] = ActiveSupport::OrderedOptions.new

      act_as_system_user do
        [["http://wb0.example.com", false],
        ["http://wb1.example.com", true],
        ["http://wb2.example.com", false],
        ["https://wb2.example.com", true],
        ["https://wb2.example.com/", true],
        ["https://wb3.example.com/", true],
        ["https://wb4.example.com/", false],
        ].each do |pfx, result|
          a = ApiClient.create(url_prefix: pfx, is_trusted: false)
          if token_lifetime_enabled
            assert_equal false, a.is_trusted, "API client with url prefix '#{pfx}' shouldn't be trusted"
          else
            assert_equal result, a.is_trusted
          end
        end

        a = ApiClient.create(url_prefix: "http://example.com", is_trusted: true)
        a.save!
        a.reload
        assert a.is_trusted
      end
    end
  end

  [
    [true, "https://ok.example", "https://ok.example"],
    [true, "https://ok.example:443/", "https://ok.example"],
    [true, "https://ok.example", "https://ok.example:443/"],
    [true, "https://ok.example", "https://ok.example/foo/bar"],
    [true, "https://ok.example", "https://ok.example?foo/bar"],
    [true, "https://ok.example/waz?quux", "https://ok.example/foo?bar#baz"],
    [false, "https://ok.example", "http://ok.example"],
    [false, "https://ok.example", "http://ok.example:443"],

    [true, "https://*.wildcard.example", "https://ok.wildcard.example"],
    [true, "https://*.wildcard.example", "https://ok.ok.ok.wildcard.example"],
    [false, "https://*.wildcard.example", "http://wrongscheme.wildcard.example"],
    [false, "https://*.wildcard.example", "https://wrongport.wildcard.example:80"],
    [false, "https://*.wildcard.example", "https://ok.wildcard.example.attacker.example/"],
    [false, "https://*.wildcard.example", "https://attacker.example/https://ok.wildcard.example/"],
    [false, "https://*.wildcard.example", "https://attacker.example/?https://ok.wildcard.example/"],
    [false, "https://*.wildcard.example", "https://attacker.example/#https://ok.wildcard.example/"],
    [false, "https://*-wildcard.example", "https://notsupported-wildcard.example"],
  ].each do |pass, trusted, current|
    test "is_trusted(#{current}) returns #{pass} based on #{trusted} in TrustedClients" do
      Rails.configuration.Login.TrustedClients = ActiveSupport::OrderedOptions.new
      Rails.configuration.Login.TrustedClients[trusted.to_sym] = ActiveSupport::OrderedOptions.new
      assert_equal pass, ApiClient.new(url_prefix: current).is_trusted
    end
  end
end
