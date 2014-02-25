require 'test_helper'

class Arvados::V1::CollectionsControllerTest < ActionController::TestCase

  test "should get index" do
    authorize_with :active
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
  end

  test "should create" do
    authorize_with :active
    test_collection = {
      manifest_text: <<-EOS
. d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo.txt
. acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar.txt
. acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar.txt
./baz acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar.txt
EOS
    }
    test_collection[:uuid] =
      Digest::MD5.hexdigest(test_collection[:manifest_text]) +
      '+' +
      test_collection[:manifest_text].length.to_s
    post :create, {
      collection: test_collection
    }
    assert_response :success
    assert_nil assigns(:objects)

    get :show, {
      id: test_collection[:uuid]
    }
    assert_response :success
    assert_not_nil assigns(:object)
    resp = JSON.parse(@response.body)
    assert_equal test_collection[:uuid], resp['uuid']
    assert_equal test_collection[:manifest_text], resp['manifest_text']
    assert_equal 9, resp['data_size']
    assert_equal [['.', 'foo.txt', 0],
                  ['.', 'bar.txt', 6],
                  ['./baz', 'bar.txt', 3]], resp['files']
  end

  test "list of files is correct for empty manifest" do
    authorize_with :active
    test_collection = {
      manifest_text: "",
      uuid: "d41d8cd98f00b204e9800998ecf8427e+0"
    }
    post :create, {
      collection: test_collection
    }
    assert_response :success

    get :show, {
      id: "d41d8cd98f00b204e9800998ecf8427e+0"
    }
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal [], resp['files']
  end

  test "create with owner_uuid set to owned group" do
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, {
      collection: {
        owner_uuid: 'zzzzz-j7d0g-rew6elm53kancon',
        manifest_text: manifest_text,
        uuid: "d30fe8ae534397864cb96c544f4cf102"
      }
    }
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal 'zzzzz-tpzed-000000000000000', resp['owner_uuid']
  end

  test "create with owner_uuid set to group i can_manage" do
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, {
      collection: {
        owner_uuid: 'zzzzz-j7d0g-8ulrifv67tve5sx',
        manifest_text: manifest_text,
        uuid: "d30fe8ae534397864cb96c544f4cf102"
      }
    }
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_equal 'zzzzz-tpzed-000000000000000', resp['owner_uuid']
  end

  test "create with owner_uuid set to group with no can_manage permission" do
    authorize_with :active
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, {
      collection: {
        owner_uuid: 'zzzzz-j7d0g-it30l961gq3t0oi',
        manifest_text: manifest_text,
        uuid: "d30fe8ae534397864cb96c544f4cf102"
      }
    }
    assert_response 403
  end

  test "admin create with owner_uuid set to group with no permission" do
    authorize_with :admin
    manifest_text = ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n"
    post :create, {
      collection: {
        owner_uuid: 'zzzzz-j7d0g-it30l961gq3t0oi',
        manifest_text: manifest_text,
        uuid: "d30fe8ae534397864cb96c544f4cf102"
      }
    }
    assert_response :success
  end

  test "should create with collection passed as json" do
    authorize_with :active
    post :create, {
      collection: <<-EOS
      {
        "manifest_text":". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",\
        "uuid":"d30fe8ae534397864cb96c544f4cf102"\
      }
      EOS
    }
    assert_response :success
  end

  test "should fail to create with checksum mismatch" do
    authorize_with :active
    post :create, {
      collection: <<-EOS
      {
        "manifest_text":". d41d8cd98f00b204e9800998ecf8427e 0:0:bar.txt\n",\
        "uuid":"d30fe8ae534397864cb96c544f4cf102"\
      }
      EOS
    }
    assert_response 422
  end

  test "get full provenance for baz file" do
    authorize_with :active
    get :provenance, uuid: 'ea10d51bcf88862dbcc36eb292017dfd+45'
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_not_nil resp['ea10d51bcf88862dbcc36eb292017dfd+45'] # baz
    assert_not_nil resp['fa7aeb5140e2848d39b416daeef4ffc5+45'] # bar
    assert_not_nil resp['1f4b0bc7583c2a7f9102c395f4ffc5e3+45'] # foo
    assert_not_nil resp['zzzzz-8i9sb-cjs4pklxxjykyuq'] # bar->baz
    assert_not_nil resp['zzzzz-8i9sb-aceg2bnq7jt7kon'] # foo->bar
  end

  test "get no provenance for foo file" do
    # spectator user cannot even see baz collection
    authorize_with :spectator
    get :provenance, uuid: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'
    assert_response 404
  end

  test "get partial provenance for baz file" do
    # spectator user can see bar->baz job, but not foo->bar job
    authorize_with :spectator
    get :provenance, uuid: 'ea10d51bcf88862dbcc36eb292017dfd+45'
    assert_response :success
    resp = JSON.parse(@response.body)
    assert_not_nil resp['ea10d51bcf88862dbcc36eb292017dfd+45'] # baz
    assert_not_nil resp['fa7aeb5140e2848d39b416daeef4ffc5+45'] # bar
    assert_not_nil resp['zzzzz-8i9sb-cjs4pklxxjykyuq']     # bar->baz
    assert_nil resp['zzzzz-8i9sb-aceg2bnq7jt7kon']         # foo->bar
    assert_nil resp['1f4b0bc7583c2a7f9102c395f4ffc5e3+45'] # foo
  end

end
