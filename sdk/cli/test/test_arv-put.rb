require 'minitest/autorun'
require 'digest/md5'

class TestArvPut < Minitest::Test
  def setup
    begin Dir.mkdir './tmp' rescue Errno::EEXIST end
    begin Dir.mkdir './tmp/empty_dir' rescue Errno::EEXIST end
    File.open './tmp/empty_file', 'wb' do
    end
    File.open './tmp/foo', 'wb' do |f|
      f.write 'foo'
    end
  end

  def test_help
    out, err = capture_subprocess_io do
      assert arv_put('-h'), 'arv-put -h exits zero'
    end
    $stderr.write err
    assert_empty err
    assert_match /^usage:/, out
  end

  def test_raw_stdin
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      r,w = IO.pipe
      wpid = fork do
        r.close
        w << 'foo'
      end
      w.close
      assert arv_put('--raw', {in: r})
      r.close
      Process.waitpid wpid
    end
    $stderr.write err
    assert_match '', err
    assert_equal "acbd18db4cc2f85cedef654fccc4a4d8+3\n", out
  end

  def test_raw_file
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      assert arv_put('--raw', './tmp/foo')
    end
    $stderr.write err
    assert_match '', err
    assert_equal "acbd18db4cc2f85cedef654fccc4a4d8+3\n", out
  end

  def test_raw_empty_file
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      assert arv_put('--raw', './tmp/empty_file')
    end
    $stderr.write err
    assert_match '', err
    assert_equal "d41d8cd98f00b204e9800998ecf8427e+0\n", out
  end

  def test_filename_arg_with_directory
    out, err = capture_subprocess_io do
      assert_equal(false, arv_put('--filename', 'foo', './tmp/empty_dir/.'),
                   'arv-put --filename refuses directory')
    end
    assert_match /^usage:.*error:/m, err
    assert_empty out
  end

  def test_filename_arg_with_multiple_files
    out, err = capture_subprocess_io do
      assert_equal(false, arv_put('--filename', 'foo',
                                  './tmp/empty_file',
                                  './tmp/empty_file'),
                   'arv-put --filename refuses directory')
    end
    assert_match /^usage:.*error:/m, err
    assert_empty out
  end

  def test_filename_arg_with_empty_file
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      assert arv_put('--filename', 'foo', './tmp/empty_file')
    end
    $stderr.write err
    assert_match '', err
    assert match_collection_uuid(out)
  end

  def test_as_stream
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      assert arv_put('--as-stream', './tmp/foo')
    end
    $stderr.write err
    assert_match '', err
    assert_equal foo_manifest, out
  end

  def test_progress
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      assert arv_put('--manifest', '--progress', './tmp/foo')
    end
    assert_match /%/, err
    assert match_collection_uuid(out)
  end

  def test_batch_progress
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      assert arv_put('--manifest', '--batch-progress', './tmp/foo')
    end
    assert_match /: 0 written 3 total/, err
    assert_match /: 3 written 3 total/, err
    assert match_collection_uuid(out)
  end

  def test_progress_and_batch_progress
    out, err = capture_subprocess_io do
      assert_equal(false,
                   arv_put('--progress', '--batch-progress', './tmp/foo'),
                   'arv-put --progress --batch-progress is contradictory')
    end
    assert_match /^usage:.*error:/m, err
    assert_empty out
  end

  def test_read_from_implicit_stdin
    skip "Waiting unitl #4534 is implemented"

    test_read_from_stdin(specify_stdin_as='--manifest')
  end

  def test_read_from_dev_stdin
    skip "Waiting unitl #4534 is implemented"

    test_read_from_stdin(specify_stdin_as='/dev/stdin')
  end

  def test_read_from_stdin(specify_stdin_as='-')
    skip "Waiting unitl #4534 is implemented"

    out, err = capture_subprocess_io do
      r,w = IO.pipe
      wpid = fork do
        r.close
        w << 'foo'
      end
      w.close
      assert arv_put('--filename', 'foo', specify_stdin_as,
                                 { in: r })
      r.close
      Process.waitpid wpid
    end
    $stderr.write err
    assert_match '', err
    assert match_collection_uuid(out)
  end

  def test_read_from_implicit_stdin_implicit_manifest
    skip "Waiting unitl #4534 is implemented"

    test_read_from_stdin_implicit_manifest(specify_stdin_as=nil,
                                           expect_filename='stdin')
  end

  def test_read_from_dev_stdin_implicit_manifest
    skip "Waiting unitl #4534 is implemented"

    test_read_from_stdin_implicit_manifest(specify_stdin_as='/dev/stdin')
  end

  def test_read_from_stdin_implicit_manifest(specify_stdin_as='-',
                                             expect_filename=nil)
    skip "Waiting unitl #4534 is implemented"

    expect_filename = expect_filename || specify_stdin_as.split('/').last
    out, err = capture_subprocess_io do
      r,w = IO.pipe
      wpid = fork do
        r.close
        w << 'foo'
      end
      w.close
      args = []
      args.push specify_stdin_as if specify_stdin_as
      assert arv_put(*args, { in: r })
      r.close
      Process.waitpid wpid
    end
    $stderr.write err
    assert_match '', err
    assert match_collection_uuid(out)
  end

  protected
  def arv_put(*args)
    system ['./bin/arv-put', 'arv-put'], *args
  end

  def foo_manifest(filename='foo')
    ". #{Digest::MD5.hexdigest('foo')}+3 0:3:#{filename}\n"
  end

  def foo_manifest_locator(filename='foo')
    Digest::MD5.hexdigest(foo_manifest(filename)) +
      "+#{foo_manifest(filename).length}"
  end

  def match_collection_uuid(uuid)
    /^([0-9a-z]{5}-4zz18-[0-9a-z]{15})?$/.match(uuid)
  end
end
