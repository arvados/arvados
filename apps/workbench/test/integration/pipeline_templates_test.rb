# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'

class PipelineTemplatesTest < ActionDispatch::IntegrationTest
  test "JSON popup available for strange components" do
    need_javascript
    uuid = api_fixture("pipeline_templates")["components_is_jobspec"]["uuid"]
    visit page_with_token("active", "/pipeline_templates/#{uuid}")
    click_on "Components"
    assert(page.has_no_text?("script_parameters"),
           "components JSON visible without popup")
    click_on "Show components JSON"
    assert(page.has_text?("script_parameters"),
           "components JSON not found")
  end

end
