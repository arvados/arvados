require 'minitest/autorun'
require 'arvados'
require 'digest/md5'

class TestBigRequest < Minitest::Test
  def boring_manifest nblocks
    x = '.'
    (0..nblocks).each do |z|
      x += ' d41d8cd98f00b204e9800998ecf8427e+0'
    end
    x += " 0:0:foo.txt\n"
    x
  end

  def test_create_manifest nblocks=1
    skip "Test needs an API server to run against"
    manifest_text = boring_manifest nblocks
    uuid = Digest::MD5.hexdigest(manifest_text) + '+' + manifest_text.size.to_s
    c = Arvados.new.collection.create(collection: {
                                        uuid: uuid,
                                        manifest_text: manifest_text,
                                      })
    assert_equal uuid, c[:portable_data_hash]
  end

  def test_create_big_manifest
    # This ensures that manifest_text is passed in the request body:
    # it's too large to fit in the query string.
    test_create_manifest 9999
  end
end
