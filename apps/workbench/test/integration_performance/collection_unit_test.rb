require 'test_helper'
require 'helpers/manifest_examples'
require 'helpers/time_block'

class Blob
end

class BigCollectionTest < ActiveSupport::TestCase
  include ManifestExamples

  setup do
    Blob.stubs(:sign_locator).returns 'd41d8cd98f00b204e9800998ecf8427e+0'
  end

  teardown do
    Thread.current[:arvados_api_client] = nil
  end

  # You can try with compress=false here too, but at last check it
  # didn't make a significant difference.
  [true].each do |compress|
    test "crud cycle for collection with big manifest (compress=#{compress})" do
      Rails.configuration.api_response_compression = compress
      Thread.current[:arvados_api_client] = nil
      crudtest
    end
  end

  def crudtest
    use_token :active
    bigmanifest = time_block 'build example' do
      make_manifest(streams: 100,
                    files_per_stream: 100,
                    blocks_per_file: 20,
                    bytes_per_block: 0)
    end
    c = time_block "new (manifest size = #{bigmanifest.length>>20}MiB)" do
      Collection.new manifest_text: bigmanifest
    end
    time_block 'create' do
      c.save!
    end
    time_block 'read' do
      Collection.find c.uuid
    end
    time_block 'read(cached)' do
      Collection.find c.uuid
    end
    time_block 'list' do
      list = Collection.select(['uuid', 'manifest_text']).filter [['uuid','=',c.uuid]]
      assert_equal 1, list.count
      assert_equal c.uuid, list.first.uuid
      assert_not_nil list.first.manifest_text
    end
    time_block 'update(name-only)' do
      manifest_text_length = c.manifest_text.length
      c.update_attributes name: 'renamed during test case'
      assert_equal c.manifest_text.length, manifest_text_length
    end
    time_block 'update' do
      c.manifest_text += ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:empty.txt\n"
      c.save!
    end
    time_block 'delete' do
      c.destroy
    end
    time_block 'read(404)' do
      assert_empty Collection.filter([['uuid','=',c.uuid]])
    end
  end
end
