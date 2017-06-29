# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::PipelineInstancesControllerTest < ActionController::TestCase

  test 'create pipeline with components copied from template' do
    authorize_with :active
    post :create, {
      pipeline_instance: {
        pipeline_template_uuid: pipeline_templates(:two_part).uuid
      }
    }
    assert_response :success
    assert_equal(pipeline_templates(:two_part).components.to_json,
                 assigns(:object).components.to_json)
  end

  test 'create pipeline with no template' do
    authorize_with :active
    post :create, {
      pipeline_instance: {
        components: {}
      }
    }
    assert_response :success
    assert_equal({}, assigns(:object).components)
  end

  [
    true,
    false
  ].each do |cascade|
    test "cancel a pipeline instance with cascade=#{cascade}" do
      authorize_with :active
      pi_uuid = pipeline_instances(:job_child_pipeline_with_components_at_level_2).uuid

      post :cancel, {id: pi_uuid, cascade: cascade}
      assert_response :success

      pi = PipelineInstance.where(uuid: pi_uuid).first
      assert_equal "Paused", pi.state

      children = Job.where(uuid: ['zzzzz-8i9sb-job1atlevel3noc', 'zzzzz-8i9sb-job2atlevel3noc'])
      children.each do |child|
        assert_equal ("Cancelled" == child.state), cascade
      end
    end
  end
end
