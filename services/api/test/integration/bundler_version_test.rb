# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class BundlerVersionTest < ActionDispatch::IntegrationTest
  test "Bundler version matches expectations" do
    # The expected version range should be the latest that supports all the
    # versions of Ruby we intend to support. This test checks that a developer
    # doesn't accidentally update Bundler past that point.
    expected = Gem::Dependency.new("", "~> 2.4.22")
    actual = Bundler.gem_version
    assert(
      expected.match?("", actual),
      "Bundler version #{actual} did not match #{expected}",
    )
  end
end
