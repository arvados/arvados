require 'test_helper'

class PipelineTemplatesControllerTest < ActionController::TestCase
  test "component rendering copes with unexpeceted components format" do
    get(:show,
        {id: api_fixture("pipeline_templates")["components_is_jobspec"]["uuid"]},
        session_for(:active))
    assert_response :success
  end
end
