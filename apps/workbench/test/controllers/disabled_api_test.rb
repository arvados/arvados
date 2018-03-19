# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/share_object_helper'

class DisabledApiTest < ActionController::TestCase
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, false

  test "dashboard recent processes when pipeline_instance index API is disabled" do
    @controller = ProjectsController.new

    dd = ArvadosApiClient.new_or_current.discovery.deep_dup
    dd[:resources][:pipeline_instances][:methods].delete(:index)
    ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)

    get :index, {}, session_for(:active)
    assert_includes @response.body, "zzzzz-xvhdp-cr4runningcntnr" # expect crs
    assert_not_includes @response.body, "zzzzz-d1hrv-"   # expect no pipelines
    assert_includes @response.body, "Run a process"
  end

  test "dashboard compute node status not shown when pipeline_instance index API is disabled" do
    @controller = ProjectsController.new

    dd = ArvadosApiClient.new_or_current.discovery.deep_dup
    dd[:resources][:pipeline_instances][:methods].delete(:index)
    ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)

    get :index, {}, session_for(:active)
    assert_not_includes @response.body, "compute-node-summary-pane"
  end

  [
    [:jobs, JobsController.new],
    [:job_tasks, JobTasksController.new],
    [:pipeline_instances, PipelineInstancesController.new],
    [:pipeline_templates, PipelineTemplatesController.new],
  ].each do |ctrl_name, ctrl|
    test "#{ctrl_name} index page when API is disabled" do
      @controller = ctrl

      dd = ArvadosApiClient.new_or_current.discovery.deep_dup
      dd[:resources][ctrl_name][:methods].delete(:index)
      ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)

      get :index, {}, session_for(:active)
      assert_response 404
    end
  end

  [
    :admin,
    :active,
    nil,
  ].each do |user|
    test "project tabs as user #{user} when pipeline related index APIs are disabled" do
      @controller = ProjectsController.new

      Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']

      dd = ArvadosApiClient.new_or_current.discovery.deep_dup
      dd[:resources][:pipeline_templates][:methods].delete(:index)
      ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)

      proj_uuid = api_fixture('groups')['anonymously_accessible_project']['uuid']

      if user
        get(:show, {id: proj_uuid}, session_for(user))
      else
        get(:show, {id: proj_uuid})
      end

      resp = @response.body
      assert_includes resp, "href=\"#Data_collections\""
      assert_includes resp, "href=\"#Pipelines_and_processes\""
      assert_includes resp, "href=\"#Workflows\""
      assert_not_includes resp, "href=\"#Pipeline_templates\""
      assert_includes @response.body, "Run a process" if user == :admin
    end
  end
end
