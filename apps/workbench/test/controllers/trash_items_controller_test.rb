require 'test_helper'

class TrashItemsControllerTest < ActionController::TestCase
  test "untrash collection with same name as another collection" do
    collection = api_fixture('collections')['trashed_collection_to_test_name_conflict_on_untrash']
    items = [collection['uuid']]
    post :untrash_items, {
      selection: items,
      format: :js
    }, session_for(:active)

    assert_response :success
  end
end
