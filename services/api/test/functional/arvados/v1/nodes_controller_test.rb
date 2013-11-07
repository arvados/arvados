require 'test_helper'

class Arvados::V1::NodesControllerTest < ActionController::TestCase

  test "should get index with ping_secret" do
    authorize_with :admin
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
    node_items = JSON.parse(@response.body)['items']
    assert_not_equal 0, node_items.size
    assert_not_nil node_items[0]['info'].andand['ping_secret']
  end

  # inactive user should not see any nodes
  test "should get index without ping_secret" do
    authorize_with :inactive
    get :index
    assert_response :success
    node_items = JSON.parse(@response.body)['items']
    assert_not_equal 0, node_items.size
    assert_nil node_items[0]['info'].andand['ping_secret']
  end

end
