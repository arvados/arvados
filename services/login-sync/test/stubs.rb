# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'etc'
require 'mocha/mini_test'
require 'ostruct'

module Stubs
  # These Etc mocks help only when we run arvados-login-sync in-process.

  def setup
    super
    ENV['ARVADOS_VIRTUAL_MACHINE_UUID'] = 'testvm2.shell'
    Etc.stubs(:to_enum).with(:passwd).returns stubpasswd.map { |x| OpenStruct.new x }
    Etc.stubs(:to_enum).with(:group).returns stubgroup.map { |x| OpenStruct.new x }
  end

  def stubpasswd
    [{name: 'root', uid: 0}]
  end

  def stubgroup
    [{name: 'root', gid: 0}]
  end

  # These child-ENV tricks help only when we run arvados-login-sync as a subprocess.

  def setup
    super
    @env_was = Hash[ENV]
    @tmpdir = Dir.mktmpdir
  end

  def teardown
    FileUtils.remove_dir(@tmpdir)
    ENV.select! { |k| @env_was.has_key? k }
    @env_was.each do |k,v| ENV[k]=v end
    super
  end

  def stubenv opts={}
    # Use UUID of testvm2.shell fixture, unless otherwise specified by test case.
    Hash[ENV].merge('ARVADOS_VIRTUAL_MACHINE_UUID' => 'zzzzz-2x53u-382brsig8rp3065',
                    'ARVADOS_LOGIN_SYNC_TMPDIR' => @tmpdir)
  end

  def invoke_sync opts={}
    env = stubenv.merge(opts[:env] || {})
    (opts[:binstubs] || []).each do |binstub|
      env['PATH'] = File.absolute_path('../binstub_'+binstub, __FILE__) + ':' + env['PATH']
    end
    login_sync_path = File.absolute_path '../../bin/arvados-login-sync', __FILE__
    system env, login_sync_path
  end
end
