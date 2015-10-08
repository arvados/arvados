require 'test_helper'

class AppVersionTest < ActiveSupport::TestCase

  setup do AppVersion.forget end

  teardown do AppVersion.forget end

  test 'invoke git processes only on first call' do
    AppVersion.expects(:git).
      with("status", "--porcelain").once.
      yields " M services/api/README\n"
    AppVersion.expects(:git).
      with("log", "-n1", "--format=%H").once.
      yields "da39a3ee5e6b4b0d3255bfef95601890afd80709\n"

    (0..4).each do
      v = AppVersion.hash
      assert_equal 'da39a3ee-modified', v
    end
  end

  test 'override with configuration' do
    Rails.configuration.source_version = 'foobar'
    assert_equal 'foobar', AppVersion.hash
    Rails.configuration.source_version = false
    assert_not_equal 'foobar', AppVersion.hash
  end

  test 'override with file' do
    path = Rails.root.join 'git-commit.version'
    assert(!File.exists?(path),
           "Packaged version file found in source tree: #{path}")
    begin
      File.open(path, 'w') do |f|
        f.write "0.1.abc123\n"
      end
      assert_equal "0.1.abc123", AppVersion.hash
    ensure
      File.unlink path
    end
  end
end
