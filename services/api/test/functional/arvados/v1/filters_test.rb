require 'test_helper'

class Arvados::V1::FiltersTest < ActionController::TestCase
  test '"not in" filter passes null values' do
    @controller = Arvados::V1::GroupsController.new
    authorize_with :admin
    get :index, {
      filters: [ ['group_class', 'not in', ['project']] ],
      controller: 'groups',
    }
    assert_response :success
    found = assigns(:objects)
    assert_includes(found.collect(&:group_class), nil,
                    "'group_class not in ['project']' filter should pass null")
  end

  test 'error message for non-array element in filters array' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [{bogus: 'filter'}],
    }
    assert_response 422
    assert_match(/Invalid element in filters array/,
                 json_response['errors'].join(' '))
  end

  test 'error message for full text search on a specific column' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [['uuid', '@@', 'abcdef']],
    }
    assert_response 422
    assert_match /not supported/, json_response['errors'].join(' ')
  end

  test 'difficult characters in full text search' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [['any', '@@', 'a|b"c']],
    }
    assert_response :success
    # (Doesn't matter so much which results are returned.)
  end

  test 'array operand in full text search' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index, {
      filters: [['any', '@@', ['abc', 'def']]],
    }
    assert_response 422
    assert_match /not supported/, json_response['errors'].join(' ')
  end
end
