# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'fileutils'
require 'tmpdir'

require 'integration_helper'

class JobsTest < ActionDispatch::IntegrationTest
  setup do
      need_javascript
  end

  def fakepipe_with_log_data
    content =
      "2014-01-01_12:00:01 zzzzz-8i9sb-0vsrcqi7whchuil 0  log message 1\n" +
      "2014-01-01_12:00:02 zzzzz-8i9sb-0vsrcqi7whchuil 0  log message 2\n" +
      "2014-01-01_12:00:03 zzzzz-8i9sb-0vsrcqi7whchuil 0  log message 3\n"
    StringIO.new content, 'r'
  end

  [
    ['active', true],
    ['job_reader2', false],
  ].each do |user, readable|
    test "view job with components as #{user} user" do
      job = api_fixture('jobs')['running_job_with_components']
      component1 = api_fixture('jobs')['completed_job_in_publicly_accessible_project']
      component2 = api_fixture('pipeline_instances')['running_pipeline_with_complete_job']
      component2_child1 = api_fixture('jobs')['previous_job_run']
      component2_child2 = api_fixture('jobs')['running']

      visit page_with_token(user, "/jobs/#{job['uuid']}")
      assert page.has_text? job['script_version']
      assert page.has_no_text? 'script_parameters'

      # The job_reader2 is allowed to read job, component2, and component2_child1,
      # and component2_child2 only as a component of the pipeline component2
      if readable
        assert page.has_link? 'component1'
        assert page.has_link? 'component2'
      else
        assert page.has_no_link? 'component1'
        assert page.has_link? 'component2'
      end

      if readable
        click_link('component1')
        within('.panel-collapse') do
          assert(has_text? component1['uuid'])
          assert(has_text? component1['script_version'])
          assert(has_text? 'script_parameters')
        end
        click_link('component1')
      end

      click_link('component2')
      within('.panel-collapse') do
        assert(has_text? component2['uuid'])
        assert(has_text? component2['script_version'])
        assert(has_no_text? 'script_parameters')
        assert(has_link? 'previous')
        assert(has_link? 'running')

        click_link('previous')
        within('.panel-collapse') do
          assert(has_text? component2_child1['uuid'])
          assert(has_text? component2_child1['script_version'])
        end
        click_link('previous')

        click_link('running')
        within('.panel-collapse') do
          assert(has_text? component2_child2['uuid'])
          if readable
            assert(has_text? component2_child2['script_version'])
          else
            assert(has_no_text? component2_child2['script_version'])
          end
        end
      end
    end
  end
end
