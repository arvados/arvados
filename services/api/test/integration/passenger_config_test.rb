# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class PassengerConfigTest < ActionDispatch::IntegrationTest
  def setup
    super
    @passenger_config ||= File.open(Rails.root.join("Passengerfile.json")) do |f|
      JSON.parse(f)
    end
  end

  test "Passenger disables exception extension gems" do
    # For security, consistency, and performance reasons, we do not want these
    # gems to extend exception messages included in API error responses.
    begin
      rubyopt = @passenger_config["envvars"]["RUBYOPT"].split
    rescue NoMethodError, TypeError
      rubyopt = ["<RUBYOPT not configured>"]
    end
    assert_includes(rubyopt, "--disable-did_you_mean")
    assert_includes(rubyopt, "--disable-error_highlight")
    assert_includes(rubyopt, "--disable-syntax_suggest")
  end
end
