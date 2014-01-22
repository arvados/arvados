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
