require 'minitest/autorun'
require 'digest/md5'

class TestArvGet < Minitest::Test
  def setup
    begin
      Dir.mkdir './tmp'
    rescue Errno::EEXIST
    end
    @@foo_manifest_locator ||= `echo -n foo | ./bin/arv-put --filename foo --no-progress -`.strip
    @@baz_locator ||= `echo -n baz | ./bin/arv-put --as-raw --no-progress -`.strip
    @@multilevel_manifest_locator ||= `echo ./foo/bar #{@@baz_locator} 0:3:baz | ./bin/arv-put --as-raw --no-progress -`.strip
  end

  def test_no_args
    out, err = capture_subprocess_io do
      assert_arv_get false
    end
    assert_equal '', out
    assert_match /^usage:/, err
  end

  def test_help
    out, err = capture_subprocess_io do
      assert_arv_get '-h'
    end
    $stderr.write err
    assert_equal '', err
    assert_match /^usage:/, out
  end

  def test_file_to_dev_stdout
    test_file_to_stdout('/dev/stdout')
  end

  def test_file_to_stdout(specify_stdout_as='-')
    out, err = capture_subprocess_io do
      assert_arv_get @@foo_manifest_locator + '/foo', specify_stdout_as
    end
    assert_equal '', err
    assert_equal 'foo', out
  end

  def test_file_to_file
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get @@foo_manifest_locator + '/foo', 'tmp/foo'
    end
    assert_equal '', err
    assert_equal '', out
    assert_equal 'foo', IO.read('tmp/foo')
  end

  def test_file_to_file_no_overwrite_file
    File.open './tmp/foo', 'wb' do |f|
      f.write 'baz'
    end
    out, err = capture_subprocess_io do
      assert_arv_get false, @@foo_manifest_locator + '/foo', 'tmp/foo'
    end
    assert_match /Local file tmp\/foo already exists/, err
    assert_equal '', out
    assert_equal 'baz', IO.read('tmp/foo')
  end

  def test_file_to_file_no_overwrite_file_in_dir
    File.open './tmp/foo', 'wb' do |f|
      f.write 'baz'
    end
    out, err = capture_subprocess_io do
      assert_arv_get false, @@foo_manifest_locator + '/', 'tmp/'
    end
    assert_match /Local file tmp\/foo already exists/, err
    assert_equal '', out
    assert_equal 'baz', IO.read('tmp/foo')
  end

  def test_file_to_file_force_overwrite
    File.open './tmp/foo', 'wb' do |f|
      f.write 'baz'
    end
    assert_equal 'baz', IO.read('tmp/foo')
    out, err = capture_subprocess_io do
      assert_arv_get '-f', @@foo_manifest_locator + '/', 'tmp/'
    end
    assert_match '', err
    assert_equal '', out
    assert_equal 'foo', IO.read('tmp/foo')
  end

  def test_file_to_file_skip_existing
    File.open './tmp/foo', 'wb' do |f|
      f.write 'baz'
    end
    assert_equal 'baz', IO.read('tmp/foo')
    out, err = capture_subprocess_io do
      assert_arv_get '--skip-existing', @@foo_manifest_locator + '/', 'tmp/'
    end
    assert_match '', err
    assert_equal '', out
    assert_equal 'baz', IO.read('tmp/foo')
  end

  def test_file_to_dir
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get @@foo_manifest_locator + '/foo', 'tmp/'
    end
    assert_equal '', err
    assert_equal '', out
    assert_equal 'foo', IO.read('tmp/foo')
  end

  def test_dir_to_file
    out, err = capture_subprocess_io do
      assert_arv_get false, @@foo_manifest_locator + '/', 'tmp/foo'
    end
    assert_equal '', out
    assert_match /^usage:/, err
  end

  def test_dir_to_empty_string
    out, err = capture_subprocess_io do
      assert_arv_get false, @@foo_manifest_locator + '/', ''
    end
    assert_equal '', out
    assert_match /^usage:/, err
  end

  def test_nonexistent_block
    out, err = capture_subprocess_io do
      assert_arv_get false, 'e796ab2294f3e48ec709ffa8d6daf58c'
    end
    assert_equal '', out
    assert_match /Error:/, err
  end

  def test_nonexistent_manifest
    out, err = capture_subprocess_io do
      assert_arv_get false, 'acbd18db4cc2f85cedef654fccc4a4d8/', 'tmp/'
    end
    assert_equal '', out
    assert_match /Error:/, err
  end

  def test_manifest_root_to_dir
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get '-r', @@foo_manifest_locator + '/', 'tmp/'
    end
    assert_equal '', err
    assert_equal '', out
    assert_equal 'foo', IO.read('tmp/foo')
  end

  def test_manifest_root_to_dir_noslash
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get '-r', @@foo_manifest_locator + '/', 'tmp'
    end
    assert_equal '', err
    assert_equal '', out
    assert_equal 'foo', IO.read('tmp/foo')
  end

  def test_display_md5sum
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get '-r', '--md5sum', @@foo_manifest_locator + '/', 'tmp/'
    end
    assert_equal "#{Digest::MD5.hexdigest('foo')}  ./foo\n", err
    assert_equal '', out
    assert_equal 'foo', IO.read('tmp/foo')
  end

  def test_md5sum_nowrite
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get '-n', '--md5sum', @@foo_manifest_locator + '/', 'tmp/'
    end
    assert_equal "#{Digest::MD5.hexdigest('foo')}  ./foo\n", err
    assert_equal '', out
    assert_equal false, File.exists?('tmp/foo')
  end

  def test_sha1_nowrite
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get '-n', '-r', '--hash', 'sha1', @@foo_manifest_locator+'/', 'tmp/'
    end
    assert_equal "#{Digest::SHA1.hexdigest('foo')}  ./foo\n", err
    assert_equal '', out
    assert_equal false, File.exists?('tmp/foo')
  end

  def test_block_to_file
    remove_tmp_foo
    out, err = capture_subprocess_io do
      assert_arv_get @@foo_manifest_locator, 'tmp/foo'
    end
    assert_equal '', err
    assert_equal '', out

    digest = Digest::MD5.hexdigest('foo')
    !(IO.read('tmp/foo')).gsub!( /^(. #{digest}+3)(.*)( 0:3:foo)$/).nil?
  end

  def test_create_directory_tree
    `rm -rf ./tmp/arv-get-test/`
    Dir.mkdir './tmp/arv-get-test'
    out, err = capture_subprocess_io do
      assert_arv_get @@multilevel_manifest_locator + '/', 'tmp/arv-get-test/'
    end
    assert_equal '', err
    assert_equal '', out
    assert_equal 'baz', IO.read('tmp/arv-get-test/foo/bar/baz')
  end

  def test_create_partial_directory_tree
    `rm -rf ./tmp/arv-get-test/`
    Dir.mkdir './tmp/arv-get-test'
    out, err = capture_subprocess_io do
      assert_arv_get(@@multilevel_manifest_locator + '/foo/',
                     'tmp/arv-get-test/')
    end
    assert_equal '', err
    assert_equal '', out
    assert_equal 'baz', IO.read('tmp/arv-get-test/bar/baz')
  end

  protected
  def assert_arv_get(*args)
    expect = case args.first
             when true, false
               args.shift
             else
               true
             end
    assert_equal(expect,
                 system(['./bin/arv-get', 'arv-get'], *args),
                 "`arv-get #{args.join ' '}` " +
                 "should exit #{if expect then 0 else 'non-zero' end}")
  end

  def remove_tmp_foo
    begin
      File.unlink('tmp/foo')
    rescue Errno::ENOENT
    end
  end
end
