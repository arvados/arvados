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

  test "dns_server_conf_template" do
    Rails.configuration.dns_server_conf_dir = Rails.root.join 'tmp'
    Rails.configuration.dns_server_conf_template = Rails.root.join 'config', 'unbound.template'
    conffile = Rails.root.join 'tmp', 'compute65535.conf'
    File.unlink conffile rescue nil
    assert Node.dns_server_update 'compute65535', '127.0.0.1'
    assert_match /\"1\.0\.0\.127\.in-addr\.arpa\. IN PTR compute65535\.zzzzz\.arvadosapi\.com\"/, IO.read(conffile)
    File.unlink conffile
  end

  test "dns_server_restart_command" do
    Rails.configuration.dns_server_conf_dir = Rails.root.join 'tmp'
    Rails.configuration.dns_server_reload_command = 'foobar'
    restartfile = Rails.root.join 'tmp', 'restart.txt'
    File.unlink restartfile rescue nil
    assert Node.dns_server_update 'compute65535', '127.0.0.127'
    assert_equal "foobar\n", IO.read(restartfile)
    File.unlink restartfile
  end

  test "dns_server_restart_command fail" do
    Rails.configuration.dns_server_conf_dir = Rails.root.join 'tmp', 'bogusdir'
    Rails.configuration.dns_server_reload_command = 'foobar'
    refute Node.dns_server_update 'compute65535', '127.0.0.127'
  end

  test "dns_server_update_command with valid command" do
    testfile = Rails.root.join('tmp', 'node_test_dns_server_update_command.txt')
    Rails.configuration.dns_server_update_command =
      ('echo -n "%{hostname} == %{ip_address}" >' +
       testfile.to_s.shellescape)
    assert Node.dns_server_update 'compute65535', '127.0.0.1'
    assert_equal 'compute65535 == 127.0.0.1', IO.read(testfile)
    File.unlink testfile
  end

  test "dns_server_update_command with failing command" do
    Rails.configuration.dns_server_update_command = 'false %{hostname}'
    refute Node.dns_server_update 'compute65535', '127.0.0.1'
  end

  test "dns update with no commands/dirs configured" do
    Rails.configuration.dns_server_update_command = false
    Rails.configuration.dns_server_conf_dir = false
    Rails.configuration.dns_server_conf_template = 'ignored!'
    Rails.configuration.dns_server_reload_command = 'ignored!'
    assert Node.dns_server_update 'compute65535', '127.0.0.127'
  end
end
