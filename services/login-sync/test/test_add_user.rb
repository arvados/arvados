# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'minitest/autorun'

require 'stubs'

class TestAddUser < Minitest::Test
  include Stubs

  def test_useradd_error
    valid_groups = %w(docker admin fuse).select { |g| Etc.getgrnam(g) rescue false }
    # binstub_new_user/useradd will exit non-zero because its args
    # won't match any line in this empty file:
    File.open(@tmpdir+'/succeed', 'w') do |f| end
    invoke_sync binstubs: ['new_user']
    spied = File.read(@tmpdir+'/spy')
    assert_match %r{useradd -m -c active -s /bin/bash active}, spied
    assert_match %r{useradd -m -c adminroot -s /bin/bash adminroot}, spied
  end

  def test_useradd_success
    # binstub_new_user/useradd will succeed.
    File.open(@tmpdir+'/succeed', 'w') do |f|
      f.puts 'useradd -m -c active -s /bin/bash -G active'
      f.puts 'useradd -m -c adminroot -s /bin/bash adminroot'
    end
    $stderr.puts "*** Expect crash after getpwnam() fails:"
    invoke_sync binstubs: ['new_user']
    assert !$?.success?
    spied = File.read(@tmpdir+'/spy')
    # Expect a crash after adding one user, because Dir.mkdir({home}) fails.
    assert_match %r{^useradd -m -c [^\n]+\n$}s, spied
  end
end
