require 'minitest/autorun'

require 'stubs'

class TestAddUser < Minitest::Test
  include Stubs

  def test_useradd_error
    # binstub_new_user/useradd will exit non-zero because its args
    # won't match any line in this empty file:
    File.open(@tmpdir+'/succeed', 'w') do |f| end
    invoke_sync binstubs: ['new_user']
    spied = File.read(@tmpdir+'/spy')
    assert_match %r{useradd -m -c active -s /bin/bash -G fuse active}, spied
    # BUG(TC): This assertion succeeds only if docker and fuse groups
    # exist on the host, but is insensitive to the admin group (groups
    # are quietly ignored by login-sync if they don't exist on the
    # current host).
    assert_match %r{useradd -m -c adminroot -s /bin/bash -G docker(,admin)?,fuse adminroot}, spied
  end

  def test_useradd_success
    # binstub_new_user/useradd will succeed.
    File.open(@tmpdir+'/succeed', 'w') do |f|
      f.puts 'useradd -m -c active -s /bin/bash -G fuse active'
      # Accept either form; see note about groups in test_useradd_error.
      f.puts 'useradd -m -c adminroot -s /bin/bash -G docker,fuse adminroot'
      f.puts 'useradd -m -c adminroot -s /bin/bash -G docker,admin,fuse adminroot'
    end
    $stderr.puts "*** Expect crash after getpwnam() fails:"
    invoke_sync binstubs: ['new_user']
    assert !$?.success?
    spied = File.read(@tmpdir+'/spy')
    # Expect a crash after adding one user, because Dir.mkdir({home}) fails.
    assert_match %r{^useradd -m -c [^\n]+\n$}s, spied
  end
end
