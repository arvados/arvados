require 'test_helper'

class Arvados::V1::FiltersTest < ActionController::TestCase
  test '"not in" filter passes null values' do
    @controller = Arvados::V1::GroupsController.new
    authorize_with :admin
    get :index, {
      filters: [ ['group_class', 'not in', ['folder']] ],
      controller: 'groups',
    }
    assert_response :success
    found = assigns(:objects)
    assert_includes(found.collect(&:group_class), nil,
                    "'group_class not in ['folder']' filter should pass null")
  end
end
