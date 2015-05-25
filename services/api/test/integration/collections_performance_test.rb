require 'test_helper'
require 'helpers/manifest_examples'
require 'helpers/time_block'

class CollectionsApiPerformanceTest < ActionDispatch::IntegrationTest
  include ManifestExamples

  test "crud cycle for a collection with a big manifest" do
    bigmanifest = time_block 'make example' do
      make_manifest(streams: 100,
                    files_per_stream: 100,
                    blocks_per_file: 10,
                    bytes_per_block: 2**26,
                    api_token: api_token(:active))
    end
    json = time_block "JSON encode #{bigmanifest.length>>20}MiB manifest" do
      Oj.dump({manifest_text: bigmanifest})
    end
    time_block 'create' do
      post '/arvados/v1/collections', {collection: json}, auth(:active)
      assert_response :success
    end
    uuid = json_response['uuid']
    time_block 'read' do
      get '/arvados/v1/collections/' + uuid, {}, auth(:active)
      assert_response :success
    end
    time_block 'list' do
      get '/arvados/v1/collections', {select: ['manifest_text'], filters: [['uuid', '=', uuid]].to_json}, auth(:active)
      assert_response :success
    end
    time_block 'update' do
      put '/arvados/v1/collections/' + uuid, {collection: json}, auth(:active)
      assert_response :success
    end
    time_block 'delete' do
      delete '/arvados/v1/collections/' + uuid, {}, auth(:active)
    end
  end
end
