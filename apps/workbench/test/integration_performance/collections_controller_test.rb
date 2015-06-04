require 'test_helper'
require 'helpers/manifest_examples'
require 'helpers/time_block'

class Blob
end

class BigCollectionsControllerTest < ActionController::TestCase
  include ManifestExamples

  setup do
    Blob.stubs(:sign_locator).returns 'd41d8cd98f00b204e9800998ecf8427e+0'
  end

  test "combine two big and two small collections" do
    @controller = ActionsController.new
    bigmanifest1 = time_block 'build example' do
      make_manifest(streams: 100,
                    files_per_stream: 100,
                    blocks_per_file: 20,
                    bytes_per_block: 0)
    end
    bigmanifest2 = bigmanifest1.gsub '.txt', '.txt2'
    smallmanifest1 = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:small1.txt\n"
    smallmanifest2 = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:small2.txt\n"
    totalsize = bigmanifest1.length + bigmanifest2.length +
      smallmanifest1.length + smallmanifest2.length
    parts = time_block "create (total #{totalsize>>20}MiB)" do
      use_token :active do
        {
          big1: Collection.create(manifest_text: bigmanifest1),
          big2: Collection.create(manifest_text: bigmanifest2),
          small1: Collection.create(manifest_text: smallmanifest1),
          small2: Collection.create(manifest_text: smallmanifest2),
        }
      end
    end
    time_block 'combine' do
      post :combine_selected_files_into_collection, {
        selection: [parts[:big1].uuid,
                    parts[:big2].uuid,
                    parts[:small1].uuid + '/small1.txt',
                    parts[:small2].uuid + '/small2.txt',
                   ],
        format: :html
      }, session_for(:active)
    end
    assert_response :redirect
  end

  [:json, :html].each do |format|
    test "show collection with big manifest (#{format})" do
      bigmanifest = time_block 'build example' do
        make_manifest(streams: 100,
                      files_per_stream: 100,
                      blocks_per_file: 20,
                      bytes_per_block: 0)
      end
      @controller = CollectionsController.new
      c = time_block "create (manifest size #{bigmanifest.length>>20}MiB)" do
        use_token :active do
          Collection.create(manifest_text: bigmanifest)
        end
      end
      time_block 'show' do
        get :show, {id: c.uuid, format: format}, session_for(:active)
      end
      assert_response :success
    end
  end
end
