require 'test_helper'

class Arvados::V1::NodesControllerTest < ActionController::TestCase

  test "should get index" do
    authorize_with :admin
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
  end

  # inactive user should not see any nodes
  test "should get empty index" do
    authorize_with :inactive
    get :index
    assert_response :success
    assert_equal 0, JSON.parse(@response.body)['items'].size
  end

end
