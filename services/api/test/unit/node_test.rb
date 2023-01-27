# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'tmpdir'
require 'tempfile'

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
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir = Rails.root.join 'tmp'
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate = Rails.root.join 'config', 'unbound.template'
    conffile = Rails.root.join 'tmp', 'compute65535.conf'
    File.unlink conffile rescue nil
    assert Node.dns_server_update 'compute65535', '127.0.0.1'
    assert_match(/\"1\.0\.0\.127\.in-addr\.arpa\. IN PTR compute65535\.zzzzz\.arvadosapi\.com\"/, IO.read(conffile))
    File.unlink conffile
  end

  test "dns_server_restart_command" do
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir = Rails.root.join 'tmp'
    Rails.configuration.Containers.SLURM.Managed.DNSServerReloadCommand = 'foobar'
    restartfile = Rails.root.join 'tmp', 'restart.txt'
    File.unlink restartfile rescue nil
    assert Node.dns_server_update 'compute65535', '127.0.0.127'
    assert_equal "foobar\n", IO.read(restartfile)
    File.unlink restartfile
  end

  test "dns_server_restart_command fail" do
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir = Rails.root.join 'tmp', 'bogusdir'
    Rails.configuration.Containers.SLURM.Managed.DNSServerReloadCommand = 'foobar'
    refute Node.dns_server_update 'compute65535', '127.0.0.127'
  end

  test "dns_server_update_command with valid command" do
    testfile = Rails.root.join('tmp', 'node_test_dns_server_update_command.txt')
    Rails.configuration.Containers.SLURM.Managed.DNSServerUpdateCommand =
      ('echo -n "%{hostname} == %{ip_address}" >' +
       testfile.to_s.shellescape)
    assert Node.dns_server_update 'compute65535', '127.0.0.1'
    assert_equal 'compute65535 == 127.0.0.1', IO.read(testfile)
    File.unlink testfile
  end

  test "dns_server_update_command with failing command" do
    Rails.configuration.Containers.SLURM.Managed.DNSServerUpdateCommand = 'false %{hostname}'
    refute Node.dns_server_update 'compute65535', '127.0.0.1'
  end

  test "dns update with no commands/dirs configured" do
    Rails.configuration.Containers.SLURM.Managed.DNSServerUpdateCommand = ""
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir = ""
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate = 'ignored!'
    Rails.configuration.Containers.SLURM.Managed.DNSServerReloadCommand = 'ignored!'
    assert Node.dns_server_update 'compute65535', '127.0.0.127'
  end

  test "don't leave temp files behind if there's an error writing them" do
    Rails.configuration.Containers.SLURM.Managed.DNSServerConfTemplate = Rails.root.join 'config', 'unbound.template'
    Tempfile.any_instance.stubs(:puts).raises(IOError)
    Dir.mktmpdir do |tmpdir|
      Rails.configuration.Containers.SLURM.Managed.DNSServerConfDir = tmpdir
      refute Node.dns_server_update 'compute65535', '127.0.0.127'
      assert_empty Dir.entries(tmpdir).select{|f| File.file? f}
    end
  end

  test "ping new node with no hostname and default config" do
    node = ping_node(:new_with_no_hostname, {})
    slot_number = node.slot_number
    refute_nil slot_number
    assert_equal("compute#{slot_number}", node.hostname)
  end

  test "ping new node with no hostname and no config" do
    Rails.configuration.Containers.SLURM.Managed.AssignNodeHostname = false
    node = ping_node(:new_with_no_hostname, {})
    refute_nil node.slot_number
    assert_nil node.hostname
  end

  test "ping new node with zero padding config" do
    Rails.configuration.Containers.SLURM.Managed.AssignNodeHostname = 'compute%<slot_number>04d'
    node = ping_node(:new_with_no_hostname, {})
    slot_number = node.slot_number
    refute_nil slot_number
    assert_equal("compute000#{slot_number}", node.hostname)
  end

  test "ping node with hostname and config and expect hostname unchanged" do
    node = ping_node(:new_with_custom_hostname, {})
    assert_equal(23, node.slot_number)
    assert_equal("custom1", node.hostname)
  end

  test "ping node with hostname and no config and expect hostname unchanged" do
    Rails.configuration.Containers.SLURM.Managed.AssignNodeHostname = false
    node = ping_node(:new_with_custom_hostname, {})
    assert_equal(23, node.slot_number)
    assert_equal("custom1", node.hostname)
  end

  # Ping two nodes: one without a hostname and the other with a hostname.
  # Verify that the first one gets a hostname and second one is unchanged.
  test "ping two nodes one with no hostname and one with hostname and check hostnames" do
    # ping node with no hostname and expect it set with config format
    node = ping_node(:new_with_no_hostname, {})
    refute_nil node.slot_number
    assert_equal "compute#{node.slot_number}", node.hostname

    # ping node with a hostname and expect it to be unchanged
    node2 = ping_node(:new_with_custom_hostname, {})
    refute_nil node2.slot_number
    assert_equal "custom1", node2.hostname
  end

  test "update dns when hostname and ip_address are cleared" do
    act_as_system_user do
      node = ping_node(:new_with_custom_hostname, {})
      Node.expects(:dns_server_update).with(node.hostname, Node::UNUSED_NODE_IP)
      node.update_attributes(hostname: nil, ip_address: nil)
    end
  end

  test "update dns when hostname changes" do
    act_as_system_user do
      node = ping_node(:new_with_custom_hostname, {})

      Node.expects(:dns_server_update).with(node.hostname, Node::UNUSED_NODE_IP)
      Node.expects(:dns_server_update).with('foo0', node.ip_address)
      node.update_attributes!(hostname: 'foo0')

      Node.expects(:dns_server_update).with('foo0', Node::UNUSED_NODE_IP)
      node.update_attributes!(hostname: nil, ip_address: nil)

      Node.expects(:dns_server_update).with('foo0', '10.11.12.13')
      node.update_attributes!(hostname: 'foo0', ip_address: '10.11.12.13')

      Node.expects(:dns_server_update).with('foo0', '10.11.12.14')
      node.update_attributes!(hostname: 'foo0', ip_address: '10.11.12.14')
    end
  end

  test 'newest ping wins IP address conflict' do
    act_as_system_user do
      n1, n2 = Node.create!, Node.create!

      n1.ping(ip: '10.5.5.5', ping_secret: n1.info['ping_secret'])
      n1.reload

      Node.expects(:dns_server_update).with(n1.hostname, Node::UNUSED_NODE_IP)
      Node.expects(:dns_server_update).with(Not(equals(n1.hostname)), '10.5.5.5')
      n2.ping(ip: '10.5.5.5', ping_secret: n2.info['ping_secret'])

      n1.reload
      n2.reload
      assert_nil n1.ip_address
      assert_equal '10.5.5.5', n2.ip_address

      Node.expects(:dns_server_update).with(n2.hostname, Node::UNUSED_NODE_IP)
      Node.expects(:dns_server_update).with(n1.hostname, '10.5.5.5')
      n1.ping(ip: '10.5.5.5', ping_secret: n1.info['ping_secret'])

      n1.reload
      n2.reload
      assert_nil n2.ip_address
      assert_equal '10.5.5.5', n1.ip_address
    end
  end

  test 'run out of slots' do
    act_as_system_user do
      Node.destroy_all
      (1..4).each do |i|
        n = Node.create!
        args = { ip: "10.0.0.#{i}", ping_secret: n.info['ping_secret'] }
        if i <= 3 # MAX_VMS
          n.ping(args)
        else
          assert_raises do
            n.ping(args)
          end
        end
      end
    end
  end
end
