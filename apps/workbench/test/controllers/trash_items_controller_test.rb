require 'test_helper'

class TrashItemsControllerTest < ActionController::TestCase
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  [
    :active,
    :admin,
  ].each do |user|
    test "trash index page as #{user}" do
      get :index, {partial: :trash_rows, format: :json}, session_for(user)
      assert_response :success

      items = []
      @response.body.scan(/tr\ data-object-uuid=\\"(.*?)\\"/).each do |uuid,|
        items << uuid
      end

      assert_includes(items, api_fixture('collections')['unique_expired_collection']['uuid'])
      if user == :admin
        assert_includes(items, api_fixture('collections')['unique_expired_collection2']['uuid'])
      else
        assert_not_includes(items, api_fixture('collections')['unique_expired_collection2']['uuid'])
      end
    end
  end
end
