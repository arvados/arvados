# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class GemfileLockTest < ActionDispatch::IntegrationTest
  # Like the assertion message says, refer to Gemfile for this test's
  # rationale. This test can go away once we start supporting Ruby 3.4+.
  test "base64 gem is not locked to a specific version" do
    gemfile_lock_path = Rails.root.join("Gemfile.lock")
    File.open(gemfile_lock_path) do |f|
      assert_equal(
        f.each_line.any?(/^\s*base64\s+\(/),
        false,
        "Gemfile.lock includes a specific version of base64 - revert and refer to the comments in Gemfile",
      )
    end
  end
end
