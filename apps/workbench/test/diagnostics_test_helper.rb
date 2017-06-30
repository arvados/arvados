# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'
require 'yaml'

# Diagnostics tests are executed when "RAILS_ENV=diagnostics" is used.
# When "RAILS_ENV=test" is used, tests in the "diagnostics" directory
# will not be executed.

# Command to run diagnostics tests:
#   RAILS_ENV=diagnostics bundle exec rake TEST=test/diagnostics/**/*.rb

class DiagnosticsTest < ActionDispatch::IntegrationTest

  # Prepends workbench URL to the path provided and visits that page
  # Expects path parameters such as "/collections/<uuid>"
  def visit_page_with_token token_name, path='/'
    workbench_url = Rails.configuration.arvados_workbench_url
    if workbench_url.end_with? '/'
      workbench_url = workbench_url[0, workbench_url.size-1]
    end
    tokens = Rails.configuration.user_tokens
    visit page_with_token(tokens[token_name], (workbench_url + path))
  end

  def select_input look_for
    inputs_needed = page.all('.btn', text: 'Choose')
    return if (!inputs_needed || !inputs_needed.any?)

    look_for_uuid = nil
    look_for_file = nil
    if look_for.andand.index('/').andand.>0
      partitions = look_for.partition('/')
      look_for_uuid = partitions[0]
      look_for_file = partitions[2]
    else
      look_for_uuid = look_for
      look_for_file = nil
    end

    assert_triggers_dom_event 'shown.bs.modal' do
      inputs_needed[0].click
    end

    within('.modal-dialog') do
      if look_for_uuid
        fill_in('Search', with: look_for_uuid, exact: true)
        wait_for_ajax
      end

      page.all('.selectable').first.click
      wait_for_ajax
      # ajax reload is wiping out input selection after search results; so, select again.
      page.all('.selectable').first.click
      wait_for_ajax

      if look_for_file
        wait_for_ajax
        within('.collection_files_name', text: look_for_file) do
          find('.fa-file').click
        end
      end

      find('button', text: 'OK').click
      wait_for_ajax
    end
  end

  # Looks for the text_to_look_for for up to the max_time provided
  def wait_until_page_has text_to_look_for, max_time=30
    max_time = 30 if (!max_time || (max_time.to_s != max_time.to_i.to_s))
    text_found = false
    Timeout.timeout(max_time) do
      until text_found do
        visit_page_with_token 'active', current_path
        text_found = has_text?(text_to_look_for)
      end
    end
  end
end
