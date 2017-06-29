# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class JobsControllerTest < ActionController::TestCase
  test "visit jobs index page" do
    get :index, {}, session_for(:active)
    assert_response :success
  end

  test "job page lists pipelines and jobs in which it is used" do
    get(:show,
        {id: api_fixture('jobs')['completed_job_in_publicly_accessible_project']['uuid']},
        session_for(:active))
    assert_response :success

    assert_select "div.used-in-pipelines" do
      assert_select "a[href=/pipeline_instances/zzzzz-d1hrv-n68vc490mloy4fi]"
    end

    assert_select "div.used-in-jobs" do
      assert_select "a[href=/jobs/zzzzz-8i9sb-with2components]"
    end
  end
end
