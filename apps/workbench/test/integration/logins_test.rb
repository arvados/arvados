# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'

class LoginsTest < ActionDispatch::IntegrationTest
  setup do
    need_javascript
  end

  test "login with api_token works after redirect" do
    visit page_with_token('active_trustedclient')
    assert page.has_text?('Recent processes'), "Missing 'Recent processes' from page"
    assert_no_match(/\bapi_token=/, current_path)
  end

  test "trying to use expired token redirects to login page" do
    visit page_with_token('expired_trustedclient')
    buttons = all("button.btn", text: /Log in/)
    assert_equal(1, buttons.size, "Failed to find one login button")
  end
end
