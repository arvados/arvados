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

  test "node should ping with ping_secret and no token" do
    post :ping, {
      id: 'zzzzz-7ekkf-2z3mc76g2q73aio',
      instance_id: 'i-0000000',
      local_ipv4: '172.17.2.174',
      ping_secret: '69udawxvn3zzj45hs8bumvndricrha4lcpi23pd69e44soanc0'
    }
    assert_response :success
    response = JSON.parse(@response.body)
    assert_equal 'zzzzz-7ekkf-2z3mc76g2q73aio', response['uuid']
    # Ensure we are getting the "superuser" attributes, too
    assert_not_nil response['first_ping_at'], '"first_ping_at" attr missing'
    assert_not_nil response['info'], '"info" attr missing'
    assert_not_nil response['nameservers'], '"nameservers" attr missing'
  end

  test "node should fail ping with invalid ping_secret" do
    post :ping, {
      id: 'zzzzz-7ekkf-2z3mc76g2q73aio',
      instance_id: 'i-0000000',
      local_ipv4: '172.17.2.174',
      ping_secret: 'dricrha4lcpi23pd69e44soanc069udawxvn3zzj45hs8bumvn'
    }
    assert_response 401
  end

  test "create node" do
    authorize_with :admin
    post :create
    assert_response :success
    assert_not_nil json_response['uuid']
    assert_not_nil json_response['info'].is_a? Hash
    assert_not_nil json_response['info']['ping_secret']
  end

  test "ping adds node stats to info" do
    authorize_with :admin
    node = nodes(:idle)
    post :ping, {
      id: node.uuid,
      ping_secret: node.info['ping_secret'],
      total_cpu_cores: 32,
      total_ram_mb: 1024,
      total_scratch_mb: 2048
    }
    assert_response :success
    info = JSON.parse(@response.body)['info']
    assert_equal(node.info['ping_secret'], info['ping_secret'])
    assert_equal(32, info['total_cpu_cores'].to_i)
    assert_equal(1024, info['total_ram_mb'].to_i)
    assert_equal(2048, info['total_scratch_mb'].to_i)
  end
end
