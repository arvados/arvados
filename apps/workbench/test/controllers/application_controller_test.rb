require 'test_helper'

class ApplicationControllerTest < ActionController::TestCase
  # These tests don't do state-changing API calls. Save some time by
  # skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

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
    [:preload_for_pdhs, [] ],
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
    [:preload_for_pdhs, 'input not an array'],
    [:preload_for_pdhs, nil],
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
    [:collection_for_pdh, 'no-such-pdh' ],
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
        assert_empty objects
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

    uuid = api_fixture('logs')['system_adds_foo_file']['object_uuid']

    collections = ac.send :log_collections_for_object, uuid

    assert collections, 'Expected collections'
    assert collections.is_a?(Array), 'Expected an array'
    assert collections.size == 1, 'Expected one collection object'
    assert_equal collections[0][:uuid], uuid, 'Expected uuid not found in collections'
  end

  test "preload log collections for given uuids" do
    use_token :active

    ac = ApplicationController.new

    uuid1 = api_fixture('logs')['system_adds_foo_file']['object_uuid']
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

  test "object for dataclass" do
    use_token :active

    ac = ApplicationController.new

    dataclass = ArvadosBase.resource_class_for_uuid(api_fixture('jobs')['running']['uuid'])
    uuid = api_fixture('jobs')['running']['uuid']

    obj = ac.send :object_for_dataclass, dataclass, uuid

    assert obj, 'Expected object'
    assert 'Job', obj.class
    assert_equal uuid, obj['uuid'], 'Expected uuid not found'
    assert_equal api_fixture('jobs')['running']['script_version'], obj['script_version'],
      'Expected script_version not found'
  end

  test "preload objects for dataclass" do
    use_token :active

    ac = ApplicationController.new

    dataclass = ArvadosBase.resource_class_for_uuid(api_fixture('jobs')['running']['uuid'])

    uuid1 = api_fixture('jobs')['running']['uuid']
    uuid2 = api_fixture('jobs')['running_cancelled']['uuid']

    uuids = [uuid1, uuid2]
    users = ac.send :preload_objects_for_dataclass, dataclass, uuids

    assert users, 'Expected objects'
    assert users.is_a?(Hash), 'Expected a hash'

    assert users.size == 2, 'Expected two objects in the preloaded hash'
    assert users[uuid1], 'Expected user object for the passed in uuid'
    assert users[uuid2], 'Expected user object for the passed in uuid'

    # invoke again for this same input. this time, the preloaded data will be returned
    users = ac.send :preload_objects_for_dataclass, dataclass, uuids
    assert users, 'Expected objects'
    assert users.is_a?(Hash), 'Expected a hash'
    assert users.size == 2, 'Expected two objects in the preloaded hash'

    # invoke again for this with one more uuid
    uuids << api_fixture('jobs')['foobar']['uuid']
    users = ac.send :preload_objects_for_dataclass, dataclass, uuids
    assert users, 'Expected objects'
    assert users.is_a?(Hash), 'Expected a hash'
    assert users.size == 3, 'Expected two objects in the preloaded hash'
  end

  test "preload one collection each for given portable_data_hash list" do
    use_token :active

    ac = ApplicationController.new

    pdh1 = api_fixture('collections')['foo_file']['portable_data_hash']
    pdh2 = api_fixture('collections')['bar_file']['portable_data_hash']

    pdhs = [pdh1, pdh2]
    collections = ac.send :preload_for_pdhs, pdhs

    assert collections, 'Expected collections map'
    assert collections.is_a?(Hash), 'Expected a hash'
    # Each pdh has more than one collection; however, we should get only one for each
    assert collections.size == 2, 'Expected two objects in the preloaded collection hash'
    assert collections[pdh1], 'Expected collections for the passed in pdh #{pdh1}'
    assert_equal collections[pdh1].size, 1, 'Expected one collection for the passed in pdh #{pdh1}'
    assert collections[pdh2], 'Expected collections for the passed in pdh #{pdh2}'
    assert_equal collections[pdh2].size, 1, 'Expected one collection for the passed in pdh #{pdh2}'
  end

  test "requesting a nonexistent object returns 404" do
    # We're really testing ApplicationController's find_object_by_uuid.
    # It's easiest to do that by instantiating a concrete controller.
    @controller = NodesController.new
    get(:show, {id: "zzzzz-zzzzz-zzzzzzzzzzzzzzz"}, session_for(:admin))
    assert_response 404
  end

  test "Workbench returns 4xx when API server is unreachable" do
    # We're really testing ApplicationController's render_exception.
    # Our primary concern is that it doesn't raise an error and
    # return 500.
    orig_api_server = Rails.configuration.arvados_v1_base
    begin
      # The URL should look valid in all respects, and avoid talking over a
      # network.  100::/64 is the IPv6 discard prefix, so it's perfect.
      Rails.configuration.arvados_v1_base = "https://[100::f]:1/"
      @controller = NodesController.new
      get(:index, {}, session_for(:active))
      assert_includes(405..422, @response.code.to_i,
                      "bad response code when API server is unreachable")
    ensure
      Rails.configuration.arvados_v1_base = orig_api_server
    end
  end

  [
    [CollectionsController.new, api_fixture('collections')['user_agreement_in_anonymously_accessible_project']],
    [CollectionsController.new, api_fixture('collections')['user_agreement_in_anonymously_accessible_project'], false],
    [JobsController.new, api_fixture('jobs')['running_job_in_publicly_accessible_project']],
    [JobsController.new, api_fixture('jobs')['running_job_in_publicly_accessible_project'], false],
    [PipelineInstancesController.new, api_fixture('pipeline_instances')['pipeline_in_publicly_accessible_project']],
    [PipelineInstancesController.new, api_fixture('pipeline_instances')['pipeline_in_publicly_accessible_project'], false],
    [PipelineTemplatesController.new, api_fixture('pipeline_templates')['pipeline_template_in_publicly_accessible_project']],
    [PipelineTemplatesController.new, api_fixture('pipeline_templates')['pipeline_template_in_publicly_accessible_project'], false],
    [ProjectsController.new, api_fixture('groups')['anonymously_accessible_project']],
    [ProjectsController.new, api_fixture('groups')['anonymously_accessible_project'], false],
  ].each do |controller, fixture, anon_config=true|
    test "#{controller} show method with anonymous config enabled" do
      if anon_config
        Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
      else
        Rails.configuration.anonymous_user_token = false
      end

      @controller = controller

      get(:show, {id: fixture['uuid']})

      if anon_config
        assert_response 200
        if controller.class == JobsController
          assert_includes @response.inspect, fixture['script']
        else
          assert_includes @response.inspect, fixture['name']
        end
      else
        assert_response :redirect
        assert_match /\/users\/welcome/, @response.redirect_url
      end
    end
  end

  [
    true,
    false,
  ].each do |config|
    test "invoke show with include_accept_encoding_header config #{config}" do
      Rails.configuration.include_accept_encoding_header_in_api_requests = config

      @controller = CollectionsController.new
      get(:show, {id: api_fixture('collections')['foo_file']['uuid']}, session_for(:admin))

      assert_equal([['.', 'foo', 3]], assigns(:object).files)
    end
  end

  test 'Edit name and verify that a duplicate is not created' do
    @controller = ProjectsController.new
    project = api_fixture("groups")["aproject"]
    post :update, {
      id: project["uuid"],
      project: {
        name: 'test name'
      },
      format: :json
    }, session_for(:active)
    assert_includes @response.body, 'test name'
    updated = assigns(:object)
    assert_equal updated.uuid, project["uuid"]
    assert_equal 'test name', updated.name
  end
end
