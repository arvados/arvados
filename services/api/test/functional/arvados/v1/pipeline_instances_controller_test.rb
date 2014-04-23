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

end
