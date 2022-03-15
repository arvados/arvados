# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

require 'minitest/autorun'
require 'digest/md5'
require 'active_support'
require 'active_support/core_ext'
require 'tempfile'

class TestCollectionCreate < Minitest::Test
  def setup
  end

  def test_small_collection
    uuid = Digest::MD5.hexdigest(foo_manifest) + '+' + foo_manifest.size.to_s
    ok = nil
    out, err = capture_subprocess_io do
      ok = arv('--format', 'uuid', 'collection', 'create', '--collection', {
                     uuid: uuid,
                     manifest_text: foo_manifest
                   }.to_json)
    end
    assert_equal('', err)
    assert_equal(true, ok)
    assert_match(/^([0-9a-z]{5}-4zz18-[0-9a-z]{15})?$/, out)
  end

  def test_collection_replace_files
    ok = nil
    uuid, err = capture_subprocess_io do
      ok = arv('--format', 'uuid', 'collection', 'create', '--collection', '{}')
    end
    assert_equal('', err)
    assert_equal(true, ok)
    assert_match(/^([0-9a-z]{5}-4zz18-[0-9a-z]{15})?$/, uuid)
    uuid = uuid.strip

    out, err = capture_subprocess_io do
      ok = arv('--format', 'uuid',
                   'collection', 'update',
                   '--uuid', uuid,
                   '--collection', '{}',
                   '--replace-files', {
                     "/gpl.pdf": "b519d9cb706a29fc7ea24dbea2f05851+93/GNU_General_Public_License,_version_3.pdf",
                   }.to_json)
    end
    assert_equal('', err)
    assert_equal(true, ok)
    assert_equal(uuid, out.strip)

    ok = nil
    out, err = capture_subprocess_io do
      ok = arv('--format', 'json', 'collection', 'get', '--uuid', uuid)
    end
    assert_equal('', err)
    assert_equal(true, ok)
    assert_match(/\. 6a4ff0499484c6c79c95cd8c566bd25f\+249025.* 0:249025:gpl.pdf\\n/, out)
  end

  def test_read_resource_object_from_file
    tempfile = Tempfile.new('collection')
    begin
      tempfile.write({manifest_text: foo_manifest}.to_json)
      tempfile.close
      ok = nil
      out, err = capture_subprocess_io do
        ok = arv('--format', 'uuid',
                     'collection', 'create', '--collection', tempfile.path)
      end
      assert_equal('', err)
      assert_equal(true, ok)
      assert_match(/^([0-9a-z]{5}-4zz18-[0-9a-z]{15})?$/, out)
    ensure
      tempfile.unlink
    end
  end

  protected
  def arv(*args)
    system(['./bin/arv', 'arv'], *args)
  end

  def foo_manifest
    ". #{Digest::MD5.hexdigest('foo')}+3 0:3:foo\n"
  end
end
