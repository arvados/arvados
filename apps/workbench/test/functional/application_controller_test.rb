require 'test_helper'

class ApplicationControllerTest < ActionController::TestCase

  setup do
    @user_dataclass = ArvadosBase.resource_class_for_uuid(api_fixture('users')['active']['uuid'])
  end

  test "links for object" do
    use_token :active

    ac = ApplicationController.new

    link_head_uuid = api_fixture('links')['foo_file_readable_by_active']['head_uuid']

    links = ac.send :links_for_object, link_head_uuid

    assert links, 'Expected links'
    assert links.is_a?(Array), 'Expected an array'
    assert links.size > 0, 'Expected at least one link'
    assert links[0][:uuid], 'Expected uuid for the head_link'
  end

  test "preload links for objects and uuids" do
    use_token :active

    ac = ApplicationController.new

    link1_head_uuid = api_fixture('links')['foo_file_readable_by_active']['head_uuid']
    link2_uuid = api_fixture('links')['bar_file_readable_by_active']['uuid']
    link3_head_uuid = api_fixture('links')['bar_file_readable_by_active']['head_uuid']

    link2_object = User.find(api_fixture('users')['active']['uuid'])
    link2_object_uuid = link2_object['uuid']

    uuids = [link1_head_uuid, link2_object, link3_head_uuid]
    links = ac.send :preload_links_for_objects, uuids

    assert links, 'Expected links'
    assert links.is_a?(Hash), 'Expected a hash'
    assert links.size == 3, 'Expected two objects in the preloaded links hash'
    assert links[link1_head_uuid], 'Expected links for the passed in link head_uuid'
    assert links[link2_object_uuid], 'Expected links for the passed in object uuid'
    assert links[link3_head_uuid], 'Expected links for the passed in link head_uuid'

    # invoke again for this same input. this time, the preloaded data will be returned
    links = ac.send :preload_links_for_objects, uuids
    assert links, 'Expected links'
    assert links.is_a?(Hash), 'Expected a hash'
    assert links.size == 3, 'Expected two objects in the preloaded links hash'
    assert links[link1_head_uuid], 'Expected links for the passed in link head_uuid'
  end

  [ [:preload_links_for_objects, [] ],
    [:preload_collections_for_objects, [] ],
    [:preload_log_collections_for_objects, [] ],
    [:preload_objects_for_dataclass, [] ],
  ].each do |input|
    test "preload data for empty array input #{input}" do
      use_token :active

      ac = ApplicationController.new

      if input[0] == :preload_objects_for_dataclass
        objects = ac.send input[0], @user_dataclass, input[1]
      else
        objects = ac.send input[0], input[1]
      end

      assert objects, 'Expected objects'
      assert objects.is_a?(Hash), 'Expected a hash'
      assert objects.size == 0, 'Expected no objects in the preloaded hash'
    end
  end

  [ [:preload_links_for_objects, 'input not an array'],
    [:preload_links_for_objects, nil],
    [:links_for_object, nil],
    [:preload_collections_for_objects, 'input not an array'],
    [:preload_collections_for_objects, nil],
    [:collections_for_object, nil],
    [:preload_log_collections_for_objects, 'input not an array'],
    [:preload_log_collections_for_objects, nil],
    [:log_collections_for_object, nil],
    [:preload_objects_for_dataclass, 'input not an array'],
    [:preload_objects_for_dataclass, nil],    
    [:object_for_dataclass, 'some_dataclass', nil],
    [:object_for_dataclass, nil, 'some_uuid'],
  ].each do |input|
    test "preload data for wrong type input #{input}" do
      use_token :active

      ac = ApplicationController.new

      if input[0] == :object_for_dataclass
        assert_raise ArgumentError do
          ac.send input[0], input[1], input[2]
        end
      else
        assert_raise ArgumentError do
          ac.send input[0], input[1]
        end
      end
    end
  end

  [ [:links_for_object, 'no-such-uuid' ],
    [:collections_for_object, 'no-such-uuid' ],
    [:log_collections_for_object, 'no-such-uuid' ],
    [:object_for_dataclass, 'no-such-uuid' ],
  ].each do |input|
    test "get data for no such uuid #{input}" do
      use_token :active

      ac = ApplicationController.new

      if input[0] == :object_for_dataclass
        object = ac.send input[0], @user_dataclass, input[1]
        assert_not object, 'Expected no object'
      else
        objects = ac.send input[0], input[1]
        assert objects, 'Expected objects'
        assert objects.is_a?(Array), 'Expected a array'
      end
    end
  end

  test "get 10 objects of data class user" do
    use_token :active

    ac = ApplicationController.new

    objects = ac.send :get_n_objects_of_class, @user_dataclass, 10

    assert objects, 'Expected objects'
    assert objects.is_a?(ArvadosResourceList), 'Expected an ArvadosResourceList'

    first_object = objects.first
    assert first_object, 'Expected at least one object'
    assert_equal 'User', first_object.class.name, 'Expected user object'

    # invoke it again. this time, the preloaded info will be returned
    objects = ac.send :get_n_objects_of_class, @user_dataclass, 10
    assert objects, 'Expected objects'
    assert_equal 'User', objects.first.class.name, 'Expected user object'
  end

  [ ['User', 10],
    [nil, 10],
    [@user_dataclass, 0],
    [@user_dataclass, -1],
    [@user_dataclass, nil] ].each do |input|
    test "get_n_objects for incorrect input #{input}" do
      use_token :active

      ac = ApplicationController.new

      assert_raise ArgumentError do
        ac.send :get_n_objects_of_class, input[0], input[1]
      end
    end
  end

  test "collections for object" do
    use_token :active

    ac = ApplicationController.new

    uuid = api_fixture('collections')['foo_file']['uuid']

    collections = ac.send :collections_for_object, uuid

    assert collections, 'Expected collections'
    assert collections.is_a?(Array), 'Expected an array'
    assert collections.size == 1, 'Expected one collection object'
    assert_equal collections[0][:uuid], uuid, 'Expected uuid not found in collections'
  end

  test "preload collections for given uuids" do
    use_token :active

    ac = ApplicationController.new

    uuid1 = api_fixture('collections')['foo_file']['uuid']
    uuid2 = api_fixture('collections')['bar_file']['uuid']

    uuids = [uuid1, uuid2]
    collections = ac.send :preload_collections_for_objects, uuids

    assert collections, 'Expected collection'
    assert collections.is_a?(Hash), 'Expected a hash'
    assert collections.size == 2, 'Expected two objects in the preloaded collection hash'
    assert collections[uuid1], 'Expected collections for the passed in uuid'
    assert_equal collections[uuid1].size, 1, 'Expected one collection for the passed in uuid'
    assert collections[uuid2], 'Expected collections for the passed in uuid'
    assert_equal collections[uuid2].size, 1, 'Expected one collection for the passed in uuid'

    # invoke again for this same input. this time, the preloaded data will be returned
    collections = ac.send :preload_collections_for_objects, uuids
    assert collections, 'Expected collection'
    assert collections.is_a?(Hash), 'Expected a hash'
    assert collections.size == 2, 'Expected two objects in the preloaded collection hash'
    assert collections[uuid1], 'Expected collections for the passed in uuid'
  end

  test "log collections for object" do
    use_token :active

    ac = ApplicationController.new

    uuid = api_fixture('logs')['log4']['object_uuid']

    collections = ac.send :log_collections_for_object, uuid

    assert collections, 'Expected collections'
    assert collections.is_a?(Array), 'Expected an array'
    assert collections.size == 1, 'Expected one collection object'
    assert_equal collections[0][:uuid], uuid, 'Expected uuid not found in collections'
  end

  test "preload log collections for given uuids" do
    use_token :active

    ac = ApplicationController.new

    uuid1 = api_fixture('logs')['log4']['object_uuid']
    uuid2 = api_fixture('collections')['bar_file']['uuid']

    uuids = [uuid1, uuid2]
    collections = ac.send :preload_log_collections_for_objects, uuids

    assert collections, 'Expected collection'
    assert collections.is_a?(Hash), 'Expected a hash'
    assert collections.size == 2, 'Expected two objects in the preloaded collection hash'
    assert collections[uuid1], 'Expected collections for the passed in uuid'
    assert_equal collections[uuid1].size, 1, 'Expected one collection for the passed in uuid'
    assert collections[uuid2], 'Expected collections for the passed in uuid'
    assert_equal collections[uuid2].size, 1, 'Expected one collection for the passed in uuid'

    # invoke again for this same input. this time, the preloaded data will be returned
    collections = ac.send :preload_log_collections_for_objects, uuids
    assert collections, 'Expected collection'
    assert collections.is_a?(Hash), 'Expected a hash'
    assert collections.size == 2, 'Expected two objects in the preloaded collection hash'
    assert collections[uuid1], 'Expected collections for the passed in uuid'
  end

end
