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

  # inactive user does not see any nodes
  test "inactive user should get empty index" do
    authorize_with :inactive
    get :index
    assert_response :success
    node_items = JSON.parse(@response.body)['items']
    assert_equal 0, node_items.size
  end

  # active user sees non-secret attributes of up and recently-up nodes
  test "active user should get non-empty index with no ping_secret" do
    authorize_with :active
    get :index
    assert_response :success
    node_items = JSON.parse(@response.body)['items']
    assert_not_equal 0, node_items.size
    found_busy_node = false
    node_items.each do |node|
      assert_nil node['info'].andand['ping_secret']
      assert_not_nil node['crunch_worker_state']
      if node['uuid'] == nodes(:busy).uuid
        found_busy_node = true
        assert_equal 'busy', node['crunch_worker_state']
      end
    end
    assert_equal true, found_busy_node
  end

end
