require 'test_helper'

class CollectionsControllerTest < ActionController::TestCase
  setup do
    @collection = collections(:one)
  end

  test "should get index" do
    get :index
    assert_response :success
    assert_not_nil assigns(:collections)
  end

  test "should get new" do
    get :new
    assert_response :success
  end

  test "should create collection" do
    assert_difference('Collection.count') do
      post :create, collection: @collection.attributes
    end

    assert_redirected_to collection_path(assigns(:collection))
  end

  test "should show collection" do
    get :show, id: @collection.to_param
    assert_response :success
  end

  test "should get edit" do
    get :edit, id: @collection.to_param
    assert_response :success
  end

  test "should update collection" do
    put :update, id: @collection.to_param, collection: @collection.attributes
    assert_redirected_to collection_path(assigns(:collection))
  end

  test "should destroy collection" do
    assert_difference('Collection.count', -1) do
      delete :destroy, id: @collection.to_param
    end

    assert_redirected_to collections_path
  end
end
