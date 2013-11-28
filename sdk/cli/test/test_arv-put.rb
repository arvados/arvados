require 'minitest/autorun'
require 'digest/md5'

class TestArvPut < Minitest::Test
  def setup
    begin
      Dir.mkdir './tmp'
      Dir.mkdir './tmp/empty_dir'
    rescue Errno::EEXIST
    end
    File.open './tmp/empty_file', 'wb' do
    end
    File.open './tmp/foo', 'wb' do |f|
      f.write 'foo'
    end
  end

  def test_no_args
    out, err = capture_subprocess_io do
      assert_equal(false, arv_put,
                   'arv-put without args exits non-zero')
    end
    assert_equal '', out
    assert_match /^usage:/, err
  end

  def test_help
    out, err = capture_subprocess_io do
      assert_equal(true, arv_put('-h'),
                   'arv-put -h exits zero')
    end
    assert_equal '', err
    assert_match /^usage:/, out
  end

  def test_filename_arg_with_directory
    out, err = capture_subprocess_io do
      assert_equal(false, arv_put('--filename', 'foo', './tmp/empty_dir/.'),
                   'arv-put --filename refuses directory')
    end
    assert_match /^usage:.*error:/m, err
    assert_equal '', out
  end

  def test_filename_arg_with_multiple_files
    out, err = capture_subprocess_io do
      assert_equal(false, arv_put('--filename', 'foo',
                                  './tmp/empty_file',
                                  './tmp/empty_file'),
                   'arv-put --filename refuses directory')
    end
    assert_match /^usage:.*error:/m, err
    assert_equal '', out
  end

  def test_filename_arg_with_empty_file
    out, err = capture_subprocess_io do
      assert_equal true, arv_put('--filename', 'foo', './tmp/empty_file')
    end
    $stderr.write err
    assert_match '', err
    assert_equal "aa4f15cbf013142a7d98b1e273f9c661+45\n", out
  end

  def test_progress
    out, err = capture_subprocess_io do
      assert_equal true, arv_put('--progress', './tmp/foo')
    end
    assert_match /%/, err
    expect_foo_manifest(out)
  end

  def test_batch_progress
    out, err = capture_subprocess_io do
      assert_equal true, arv_put('--batch-progress', './tmp/foo')
    end
    assert_match /: 0 written 3 total/, err
    assert_match /: 3 written 3 total/, err
    expect_foo_manifest(out)
  end

  def test_progress_and_batch_progress
    out, err = capture_subprocess_io do
      assert_equal(false,
                   arv_put('--progress', '--batch-progress', './tmp/foo'),
                   'arv-put --progress --batch-progress is contradictory')
    end
    assert_match /^usage:.*error:/m, err
    assert_equal '', out
  end

  def test_read_from_dev_stdin
    test_read_from_stdin(specify_stdin_as='/dev/stdin')
  end

  def test_read_from_stdin(specify_stdin_as='-')
    out, err = capture_subprocess_io do
      r,w = IO.pipe
      wpid = fork do
        r.close
        w << 'foo'
      end
      w.close
      assert_equal true, arv_put('--filename', 'foo', specify_stdin_as,
                                 { in: r })
      r.close
      Process.waitpid wpid
    end
    $stderr.write err
    assert_match '', err
    expect_foo_manifest(out)
  end

  protected
  def arv_put(*args)
    system ['./bin/arv-put', 'arv-put'], *args
  end

  def expect_foo_manifest(out)
    expect_manifest = ". #{Digest::MD5.hexdigest('foo')}+3 0:3:foo\n"
    assert_equal(Digest::MD5.hexdigest(expect_manifest) +
                 "+#{expect_manifest.length}\n",
                 out)
  end
end
