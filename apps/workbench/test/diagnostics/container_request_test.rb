# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'diagnostics_test_helper'

# This test assumes that the configured workflow_uuid corresponds to a cwl workflow.
# Ex: configure a workflow using the steps below and use the resulting workflow uuid:
#   > cd arvados/doc/user/cwl/bwa-mem
#   > arvados-cwl-runner --create-workflow bwa-mem.cwl bwa-mem-input.yml

class ContainerRequestTest < DiagnosticsTest
  crs_to_test = Rails.configuration.container_requests_to_test.andand.keys

  setup do
    need_selenium 'to make websockets work'
  end

  crs_to_test.andand.each do |cr_to_test|
    test "run container_request: #{cr_to_test}" do
      cr_config = Rails.configuration.container_requests_to_test[cr_to_test]

      visit_page_with_token 'active'

      find('.btn', text: 'Run a process').click

      within('.modal-dialog') do
        page.find_field('Search').set cr_config['workflow_uuid']
        wait_for_ajax
        find('.selectable', text: 'bwa-mem.cwl').click
        find('.btn', text: 'Next: choose inputs').click
      end

      page.assert_selector('a.disabled,button.disabled', text: 'Run') if cr_config['input_paths'].any?

      # Choose input for the workflow
      cr_config['input_paths'].each do |look_for|
        select_input look_for
      end
      wait_for_ajax

      # All needed input are already filled in. Run this workflow now
      page.assert_no_selector('a.disabled,button.disabled', text: 'Run')
      find('a,button', text: 'Run').click

      # container_request is running. Run button is no longer available.
      page.assert_no_selector('a', text: 'Run')

      # Wait for container_request run to complete
      wait_until_page_has 'completed', cr_config['max_wait_seconds']
    end
  end
end
