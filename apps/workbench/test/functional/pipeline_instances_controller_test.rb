require 'test_helper'

class PipelineInstancesControllerTest < ActionController::TestCase
  def create_instance_long_enough_to
    pt_fixture = api_fixture('pipeline_templates')['two_part']
    post :create, {
      pipeline_instance: {
        pipeline_template_uuid: pt_fixture['uuid']
      },
      format: :json
    }, session_for(:active)
    assert_response :success
    pi_uuid = assigns(:object).uuid
    assert_not_nil assigns(:object)
    yield pi_uuid, pt_fixture
    post :destroy, {
      id: pi_uuid,
      format: :json
    }
    assert_response :success
  end

  test "pipeline instance components populated after create" do
    create_instance_long_enough_to do |new_instance_uuid, template_fixture|
      assert_equal(template_fixture['components'].to_json,
                   assigns(:object).components.to_json)
    end
  end

  test "update script_parameters one at a time using merge param" do
    create_instance_long_enough_to do |new_instance_uuid, template_fixture|
      post :update, {
        id: new_instance_uuid,
        pipeline_instance: {
          components: {
            "part-two" => {
              script_parameters: {
                integer_with_value: {
                  value: 9
                },
                plain_string: {
                  value: 'quux'
                },
              }
            }
          }
        },
        merge: true,
        format: :json
      }, session_for(:active)
      assert_response :success
      assert_not_nil assigns(:object)
      orig_params = template_fixture['components']['part-two']['script_parameters']
      new_params = assigns(:object).components[:'part-two'][:script_parameters]
      orig_params.keys.each do |k|
        unless %w(integer_with_value plain_string).index(k)
          assert_equal orig_params[k].to_json, new_params[k.to_sym].to_json
        end
      end
    end
  end
end
