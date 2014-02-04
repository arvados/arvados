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
    post :create, {
      collection: {
        manifest_text: ". d41d8cd98f00b204e9800998ecf8427e 0:0:foo.txt\n",
        uuid: "d30fe8ae534397864cb96c544f4cf102"
      }
    }
    assert_response :success
    assert_nil assigns(:objects)
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

end
