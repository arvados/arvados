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

  test 'api responses provide timestamps with nanoseconds' do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :active
    get :index
    assert_response :success
    assert_not_empty json_response['items']
    json_response['items'].each do |item|
      %w(created_at modified_at).each do |attr|
        # Pass fixtures with null timestamps.
        next if item[attr].nil?
        assert_match /^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d.\d{9}Z$/, item[attr]
      end
    end
  end

  %w(< > <= >= =).each do |operator|
    test "timestamp #{operator} filters work with nanosecond precision" do
      # Python clients like Node Manager rely on this exact format.
      # If you must change this format for some reason, make sure you
      # coordinate the change with them.
      expect_match = !!operator.index('=')
      mine = act_as_user users(:active) do
        Collection.create!(manifest_text: '')
      end
      timestamp = mine.modified_at.strftime('%Y-%m-%dT%H:%M:%S.%NZ')
      @controller = Arvados::V1::CollectionsController.new
      authorize_with :active
      get :index, {
        filters: [['modified_at', operator, timestamp],
                  ['uuid', '=', mine.uuid]],
      }
      assert_response :success
      uuids = json_response['items'].map { |item| item['uuid'] }
      if expect_match
        assert_includes uuids, mine.uuid
      else
        assert_not_includes uuids, mine.uuid
      end
    end
  end
end
