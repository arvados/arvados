require 'minitest/autorun'
require 'digest/md5'
require 'active_support/core_ext'
require 'tempfile'

class TestCollectionCreate < Minitest::Test
  def setup
  end

  def test_small_collection
    uuid = Digest::MD5.hexdigest(foo_manifest) + '+' + foo_manifest.size.to_s
    out, err = capture_subprocess_io do
      assert_arv('--format', 'uuid', 'collection', 'create', '--collection', {
                   uuid: uuid,
                   manifest_text: foo_manifest
                 }.to_json)
    end
    assert /^([0-9a-z]{5}-4zz18-[0-9a-z]{15})?$/.match(out)
    assert_equal '', err
  end

  def test_read_resource_object_from_file
    tempfile = Tempfile.new('collection')
    begin
      tempfile.write({manifest_text: foo_manifest}.to_json)
      tempfile.close
      out, err = capture_subprocess_io do
        assert_arv('--format', 'uuid',
                   'collection', 'create', '--collection', tempfile.path)
      end
      assert /^([0-9a-z]{5}-4zz18-[0-9a-z]{15})?$/.match(out)
      assert_equal '', err
    ensure
      tempfile.unlink
    end
  end

  protected
  def assert_arv(*args)
    expect = case args.first
             when true, false
               args.shift
             else
               true
             end
    assert_equal(expect,
                 system(['./bin/arv', 'arv'], *args),
                 "`arv #{args.join ' '}` " +
                 "should exit #{if expect then 0 else 'non-zero' end}")
  end

  def foo_manifest
    ". #{Digest::MD5.hexdigest('foo')}+3 0:3:foo\n"
  end
end
