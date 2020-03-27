# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

# Tests XSS vulnerability monkeypatch
# See: https://github.com/advisories/GHSA-65cv-r6x7-79hv
class JavascriptHelperTest < ActionView::TestCase
  def test_escape_backtick
    assert_equal "\\`", escape_javascript("`")
  end

  def test_escape_dollar_sign
    assert_equal "\\$", escape_javascript("$")
  end
end
