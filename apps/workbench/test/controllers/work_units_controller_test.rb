# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class WorkUnitsControllerTest < ActionController::TestCase
  # These tests don't do state-changing API calls.
  # Save some time by skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  [
    ['foo', 10, 25,
      ['/pipeline_instances/zzzzz-d1hrv-1xfj6xkicf2muk2',
       '/pipeline_instances/zzzzz-d1hrv-jobspeccomponts',
       '/jobs/zzzzz-8i9sb-grx15v5mjnsyxk7'],
      ['/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk3',
       '/jobs/zzzzz-8i9sb-n7omg50bvt0m1nf',
       '/container_requests/zzzzz-xvhdp-cr4completedcr2']],
    ['pipeline_with_tagged_collection_input', 1, 1,
      ['/pipeline_instances/zzzzz-d1hrv-1yfj61234abcdk3'],
      ['/pipeline_instances/zzzzz-d1hrv-jobspeccomponts',
       '/jobs/zzzzz-8i9sb-pshmckwoma9plh7',
       '/jobs/zzzzz-8i9sb-n7omg50bvt0m1nf',
       '/container_requests/zzzzz-xvhdp-cr4completedcr2']],
    ['no_such_match', 0, 0,
      [],
      ['/pipeline_instances/zzzzz-d1hrv-jobspeccomponts',
       '/jobs/zzzzz-8i9sb-pshmckwoma9plh7',
       '/jobs/zzzzz-8i9sb-n7omg50bvt0m1nf',
       '/container_requests/zzzzz-xvhdp-cr4completedcr2']],
  ].each do |search_filter, expected_min, expected_max, expected, not_expected|
    test "all_processes page for search filter '#{search_filter}'" do
      work_units_index(filters: [['any','ilike', "%#{search_filter}%"]], show_children: true)
      assert_response :success

      # Verify that expected number of processes are found
      found_count = json_response['content'].scan('<tr').count
      if expected_min == expected_max
        assert_equal(true, found_count == expected_min,
          "Not found expected number of items. Expected #{expected_min} and found #{found_count}")
      else
        assert_equal(true, found_count>=expected_min,
          "Found too few items. Expected at least #{expected_min} and found #{found_count}")
        assert_equal(true, found_count<=expected_max,
          "Found too many items. Expected at most #{expected_max} and found #{found_count}")
      end

      # verify that all expected uuid links are found
      expected.each do |link|
        assert_match /href="#{link}"/, json_response['content']
      end

      # verify that none of the not_expected uuid links are found
      not_expected.each do |link|
        assert_no_match /href="#{link}"/, json_response['content']
      end
    end
  end

  def work_units_index params
    params = {
      partial: :all_processes_rows,
      format: :json,
    }.merge(params)
    encoded_params = Hash[params.map { |k,v|
                            [k, (v.is_a?(Array) || v.is_a?(Hash)) ? v.to_json : v]
                          }]
    get :index, params: encoded_params, session: session_for(:active)
  end
end
