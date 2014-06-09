require 'test_helper'

class ApplicationControllerTest < ActionController::TestCase

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

  test "links for no such object" do
    use_token :active

    ac = ApplicationController.new

    links = ac.send :links_for_object, "no-such-uuid"

    assert links, 'Expected links'
    assert links.is_a?(Array), 'Expected an array'
    assert links.size == 0, 'Expected no links'
  end

  test "preload links for given uuids" do
    use_token :active

    ac = ApplicationController.new

    link_head_uuid1 = api_fixture('links')['foo_file_readable_by_active']['head_uuid']
    link_head_uuid2 = api_fixture('links')['bar_file_readable_by_active']['head_uuid']

    uuids = [link_head_uuid1, link_head_uuid2]
    links = ac.send :preload_links_for_objects, uuids

    assert links, 'Expected links'
    assert links.is_a?(Hash), 'Expected a hash'
    assert links.size == 2, 'Expected two objects in the preloaded links hash'
    assert links[link_head_uuid1], 'Expected links for the passed in head_uuid'
    assert links[link_head_uuid2], 'Expected links for the passed in head_uuid'
  end

  test "preload links for object and uuids" do
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
  end

  test "preload links for wrong typed input" do
    use_token :active

    ac = ApplicationController.new

    assert_raise ArgumentError do
      links = ac.send :preload_links_for_objects, 'input not an array'
    end
  end

  test "preload links for nil input" do
    use_token :active

    ac = ApplicationController.new

    assert_raise ArgumentError do
      links = ac.send :preload_links_for_objects, nil
    end
  end

  test "preload links for empty array input" do
    use_token :active

    ac = ApplicationController.new

    links = ac.send :preload_links_for_objects, []

    assert links, 'Expected links'
    assert links.is_a?(Hash), 'Expected a hash'
    assert links.size == 0, 'Expected no objects in the preloaded links hash'
  end

end
