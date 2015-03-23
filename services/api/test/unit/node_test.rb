require 'test_helper'

class NodeTest < ActiveSupport::TestCase
  def ping_node(node_name, ping_data)
    set_user_from_auth :admin
    node = nodes(node_name)
    node.ping({ping_secret: node.info['ping_secret'],
                ip: node.ip_address}.merge(ping_data))
    node
  end

  test "pinging a node can add and update stats" do
    node = ping_node(:idle, {total_cpu_cores: '12', total_ram_mb: '512'})
    assert_equal(12, node.properties['total_cpu_cores'])
    assert_equal(512, node.properties['total_ram_mb'])
  end

  test "stats disappear if not in a ping" do
    node = ping_node(:idle, {total_ram_mb: '256'})
    refute_includes(node.properties, 'total_cpu_cores')
    assert_equal(256, node.properties['total_ram_mb'])
  end

  test "worker state is down for node with no slot" do
    node = nodes(:was_idle_now_down)
    assert_nil node.slot_number, "fixture is not what I expected"
    assert_equal 'down', node.crunch_worker_state, "wrong worker state"
  end
end
