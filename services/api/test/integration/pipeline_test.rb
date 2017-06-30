# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class PipelineIntegrationTest < ActionDispatch::IntegrationTest
  # These tests simulate the workflow of arv-run-pipeline-instance
  # and other pipeline-running code.

  def check_component_match(comp_key, comp_hash)
    assert_response :success
    built_json = json_response
    built_component = built_json["components"][comp_key]
    comp_hash.each_pair do |key, expected|
      assert_equal(expected, built_component[key.to_s],
                   "component's #{key} field changed")
    end
  end

  test "creating a pipeline instance preserves required component parameters" do
    comp_name = "test_component"
    component = {
      repository: "test_repo",
      script: "test_script",
      script_version: "test_refspec",
      script_parameters: {},
    }

    post("/arvados/v1/pipeline_instances",
         {pipeline_instance: {components: {comp_name => component}}.to_json},
         auth(:active))
    check_component_match(comp_name, component)
    pi_uuid = json_response["uuid"]

    @response = nil
    get("/arvados/v1/pipeline_instances/#{pi_uuid}", {}, auth(:active))
    check_component_match(comp_name, component)
  end
end
