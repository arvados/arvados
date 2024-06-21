# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::GroupsControllerTest < ActionController::TestCase

  test "attempt to delete group that cannot be seen" do
    Rails.configuration.Users.RoleGroupsVisibleToAll = false
    authorize_with :active
    post :destroy, params: {id: groups(:empty_lonely_group).uuid}
    assert_response 404
  end

  test "attempt to delete group without read or write access" do
    authorize_with :active
    post :destroy, params: {id: groups(:empty_lonely_group).uuid}
    assert_response 403
  end

  test "attempt to delete group without write access" do
    authorize_with :active
    post :destroy, params: {id: groups(:all_users).uuid}
    assert_response 403
  end

  test "get list of projects" do
    authorize_with :active
    get :index, params: {filters: [['group_class', '=', 'project']], format: :json}
    assert_response :success
    group_uuids = []
    json_response['items'].each do |group|
      assert_equal 'project', group['group_class']
      group_uuids << group['uuid']
    end
    assert_includes group_uuids, groups(:aproject).uuid
    assert_includes group_uuids, groups(:asubproject).uuid
    assert_includes group_uuids, groups(:private).uuid
    assert_not_includes group_uuids, groups(:system_group).uuid
    assert_not_includes group_uuids, groups(:private_and_can_read_foofile).uuid
  end

  test "get list of groups that are not projects" do
    authorize_with :active
    get :index, params: {filters: [['group_class', '!=', 'project']], format: :json}
    assert_response :success
    group_uuids = []
    json_response['items'].each do |group|
      assert_not_equal 'project', group['group_class']
      group_uuids << group['uuid']
    end
    assert_not_includes group_uuids, groups(:aproject).uuid
    assert_not_includes group_uuids, groups(:asubproject).uuid
  end

  test "get list of groups with bogus group_class" do
    authorize_with :active
    get :index, params: {
      filters: [['group_class', '=', 'nogrouphasthislittleclass']],
      format: :json,
    }
    assert_response :success
    assert_equal [], json_response['items']
    assert_equal 0, json_response['items_available']
  end

  def check_project_contents_response
    assert_response :success
    assert_operator 2, :<=, json_response['items_available']
    assert_operator 2, :<=, json_response['items'].count
    kinds = json_response['items'].collect { |i| i['kind'] }.uniq
    expect_kinds = %w'arvados#group'
    assert_equal expect_kinds, (expect_kinds & kinds)

    json_response['items'].each do |i|
      if i['kind'] == 'arvados#group'
        assert(i['group_class'] == 'project',
               "group#contents returned a non-project group")
      end
    end
  end

  test 'get group-owned objects' do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      format: :json,
    }
    check_project_contents_response
  end

  test "user with project read permission can see project objects" do
    authorize_with :project_viewer
    get :contents, params: {
      id: groups(:aproject).uuid,
      format: :json,
    }
    check_project_contents_response
  end

  test "list objects across projects" do
    authorize_with :project_viewer
    get :contents, params: {
      format: :json,
      filters: [['uuid', 'is_a', 'arvados#collection']]
    }
    assert_response :success
    found_uuids = json_response['items'].collect { |i| i['uuid'] }
    [[:foo_collection_in_aproject, true],
     [:baz_collection_name_in_asubproject, true],
     [:collection_not_readable_by_active, false]].each do |collection_fixture, should_find|
      if should_find
        assert_includes found_uuids, collections(collection_fixture).uuid, "did not find collection fixture '#{collection_fixture}'"
      else
        refute_includes found_uuids, collections(collection_fixture).uuid, "found collection fixture '#{collection_fixture}'"
      end
    end
  end

  test "list trashed collections and projects" do
    authorize_with :active
    get(:contents, params: {
          format: :json,
          include_trash: true,
          filters: [
            ['uuid', 'is_a', ['arvados#collection', 'arvados#group']],
            ['is_trashed', '=', true],
          ],
          limit: 10000,
        })
    assert_response :success
    found_uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes found_uuids, groups(:trashed_project).uuid
    refute_includes found_uuids, groups(:aproject).uuid
    assert_includes found_uuids, collections(:expired_collection).uuid
    refute_includes found_uuids, collections(:w_a_z_file).uuid
  end

  test "list objects in home project" do
    authorize_with :active
    get :contents, params: {
      format: :json,
      limit: 200,
      id: users(:active).uuid
    }
    assert_response :success
    found_uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes found_uuids, collections(:collection_owned_by_active).uuid, "collection did not appear in home project"
    refute_includes found_uuids, collections(:foo_collection_in_aproject).uuid, "collection appeared unexpectedly in home project"
  end

  test "list collections in home project" do
    authorize_with :active
    get(:contents, params: {
          format: :json,
          filters: [
            ['uuid', 'is_a', 'arvados#collection'],
          ],
          limit: 200,
          id: users(:active).uuid,
        })
    assert_response :success
    found_uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes found_uuids, collections(:collection_owned_by_active).uuid, "collection did not appear in home project"
    refute_includes found_uuids, collections(:collection_owned_by_active_past_version_1).uuid, "collection appeared unexpectedly in home project"
  end

  test "list collections in home project, including old versions" do
    authorize_with :active
    get(:contents, params: {
          format: :json,
          include_old_versions: true,
          filters: [
            ['uuid', 'is_a', 'arvados#collection'],
          ],
          limit: 200,
          id: users(:active).uuid,
        })
    assert_response :success
    found_uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes found_uuids, collections(:collection_owned_by_active).uuid, "collection did not appear in home project"
    assert_includes found_uuids, collections(:collection_owned_by_active_past_version_1).uuid, "old collection version did not appear in home project"
  end

  test "user with project read permission can see project collections" do
    authorize_with :project_viewer
    get :contents, params: {
      id: groups(:asubproject).uuid,
      format: :json,
    }
    ids = json_response['items'].map { |item| item["uuid"] }
    assert_includes ids, collections(:baz_file_in_asubproject).uuid
  end

  [
    ['collections.name', 'asc', :<=, "name"],
    ['collections.name', 'desc', :>=, "name"],
    ['name', 'asc', :<=, "name"],
    ['name', 'desc', :>=, "name"],
    ['collections.created_at', 'asc', :<=, "created_at"],
    ['collections.created_at', 'desc', :>=, "created_at"],
    ['created_at', 'asc', :<=, "created_at"],
    ['created_at', 'desc', :>=, "created_at"],
  ].each do |column, order, operator, field|
    test "user with project read permission can sort projects on #{column} #{order}" do
      authorize_with :project_viewer
      get :contents, params: {
        id: groups(:asubproject).uuid,
        format: :json,
        filters: [['uuid', 'is_a', "arvados#collection"]],
        order: "#{column} #{order}"
      }
      sorted_values = json_response['items'].collect { |item| item[field] }
      if field == "name"
        # Here we avoid assuming too much about the database
        # collation. Both "alice"<"Bob" and "alice">"Bob" can be
        # correct. Hopefully it _is_ safe to assume that if "a" comes
        # before "b" in the ascii alphabet, "aX">"bY" is never true for
        # any strings X and Y.
        reliably_sortable_names = sorted_values.select do |name|
          name[0] >= 'a' && name[0] <= 'z'
        end.uniq do |name|
          name[0]
        end
        # Preserve order of sorted_values. But do not use &=. If
        # sorted_values has out-of-order duplicates, we want to preserve
        # them here, so we can detect them and fail the test below.
        sorted_values.select! do |name|
          reliably_sortable_names.include? name
        end
      end
      assert_sorted(operator, sorted_values)
    end
  end

  def assert_sorted(operator, sorted_items)
    actually_checked_anything = false
    previous = nil
    sorted_items.each do |entry|
      if !previous.nil?
        assert_operator(previous, operator, entry,
                        "Entries sorted incorrectly.")
        actually_checked_anything = true
      end
      previous = entry
    end
    assert actually_checked_anything, "Didn't even find two items to compare."
  end

  # Even though the project_viewer tests go through other controllers,
  # I'm putting them here so they're easy to find alongside the other
  # project tests.
  def check_new_project_link_fails(link_attrs)
    @controller = Arvados::V1::LinksController.new
    post :create, params: {
      link: {
        link_class: "permission",
        name: "can_read",
        head_uuid: groups(:aproject).uuid,
      }.merge(link_attrs)
    }
    assert_includes(403..422, response.status)
  end

  test "user with project read permission can't add users to it" do
    authorize_with :project_viewer
    check_new_project_link_fails(tail_uuid: users(:spectator).uuid)
  end

  test "user with project read permission can't add items to it" do
    authorize_with :project_viewer
    check_new_project_link_fails(tail_uuid: collections(:baz_file).uuid)
  end

  test "user with project read permission can't rename items in it" do
    authorize_with :project_viewer
    @controller = Arvados::V1::CollectionsController.new
    post :update, params: {
      id: collections(:collection_to_search_for_in_aproject).uuid,
      name: "Denied test name",
    }
    assert_includes(403..404, response.status)
  end

  test "user with project read permission can't remove items from it" do
    @controller = Arvados::V1::CollectionsController.new
    authorize_with :project_viewer
    post :update, params: {
      id: collections(:collection_to_search_for_in_aproject).uuid,
      collection: {
        owner_uuid: users(:project_viewer).uuid,
      }
    }
    assert_response 403
  end

  test "user with project read permission can't delete it" do
    authorize_with :project_viewer
    post :destroy, params: {id: groups(:aproject).uuid}
    assert_response 403
  end

  test 'get group-owned objects with limit' do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      limit: 1,
      format: :json,
    }
    assert_response :success
    assert_operator 1, :<, json_response['items_available']
    assert_equal 1, json_response['items'].count
  end

  test 'get group-owned objects with limit and offset' do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      limit: 1,
      offset: 12345,
      format: :json,
    }
    assert_response :success
    assert_operator 1, :<, json_response['items_available']
    assert_equal 0, json_response['items'].count
  end

  test 'get group-owned objects with select' do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      limit: 100,
      format: :json,
      select: ["uuid", "storage_classes_desired"]
    }
    assert_response :success
    assert_equal 6, json_response['items_available']
    assert_equal 6, json_response['items'].count
    json_response['items'].each do |item|
      # Expect collections to have a storage_classes field, other items should not.
      if item["kind"] == "arvados#collection"
        assert !item["storage_classes_desired"].nil?
      else
        assert item["storage_classes_desired"].nil?
      end
    end
  end

  test 'get group-owned objects with invalid field in select' do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      limit: 100,
      format: :json,
      select: ["uuid", "storage_classes_desire"]
    }
    assert_response 422
  end

  test 'get group-owned objects with additional filter matching nothing' do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      filters: [['uuid', 'in', ['foo_not_a_uuid','bar_not_a_uuid']]],
      format: :json,
    }
    assert_response :success
    assert_equal [], json_response['items']
    assert_equal 0, json_response['items_available']
  end

  %w(offset limit).each do |arg|
    ['foo', '', '1234five', '0x10', '-8'].each do |val|
      test "Raise error on bogus #{arg} parameter #{val.inspect}" do
        authorize_with :active
        get :contents, params: {
          :id => groups(:aproject).uuid,
          :format => :json,
          arg => val,
        }
        assert_response 422
      end
    end
  end

  test "Collection contents don't include manifest_text or unsigned_manifest_text" do
    authorize_with :active
    get :contents, params: {
      id: groups(:aproject).uuid,
      filters: [["uuid", "is_a", "arvados#collection"]],
      format: :json,
    }
    assert_response :success
    refute(json_response["items"].any? { |c| not c["portable_data_hash"] },
           "response included an item without a portable data hash")
    refute(json_response["items"].any? { |c| c.include?("manifest_text") },
           "response included an item with manifest_text")
    refute(json_response["items"].any? { |c| c.include?("unsigned_manifest_text") },
           "response included an item with unsigned_manifest_text")
  end

  test 'get writable_by list for owned group' do
    authorize_with :active
    get :show, params: {
      id: groups(:aproject).uuid,
      format: :json
    }
    assert_response :success
    assert_not_nil(json_response['writable_by'],
                   "Should receive uuid list in 'writable_by' field")
    assert_includes(json_response['writable_by'], users(:active).uuid,
                    "owner should be included in writable_by list")
  end

  test 'no writable_by list for group with read-only access' do
    authorize_with :rominiadmin
    get :show, params: {
      id: groups(:testusergroup_admins).uuid,
      format: :json
    }
    assert_response :success
    assert_equal([json_response['owner_uuid']],
                 json_response['writable_by'],
                 "Should only see owner_uuid in 'writable_by' field")
  end

  test 'get writable_by list by admin user' do
    authorize_with :admin
    get :show, params: {
      id: groups(:testusergroup_admins).uuid,
      format: :json
    }
    assert_response :success
    assert_not_nil(json_response['writable_by'],
                   "Should receive uuid list in 'writable_by' field")
    assert_includes(json_response['writable_by'],
                    users(:admin).uuid,
                    "Current user should be included in 'writable_by' field")
  end

  test 'creating subproject with duplicate name fails' do
    authorize_with :active
    post :create, params: {
      group: {
        name: 'A Project',
        owner_uuid: users(:active).uuid,
        group_class: 'project',
      },
    }
    assert_response 422
    response_errors = json_response['errors']
    assert_not_nil response_errors, 'Expected error in response'
    assert(response_errors.first.include?('duplicate key'),
           "Expected 'duplicate key' error in #{response_errors.first}")
  end

  test 'creating duplicate named subproject succeeds with ensure_unique_name' do
    authorize_with :active
    post :create, params: {
      group: {
        name: 'A Project',
        owner_uuid: users(:active).uuid,
        group_class: 'project',
      },
      ensure_unique_name: true
    }
    assert_response :success
    new_project = json_response
    assert_not_equal(new_project['uuid'],
                     groups(:aproject).uuid,
                     "create returned same uuid as existing project")
    assert_match(/^A Project \(#{new_project['uuid'][-15..-1]}\)$/,
                 new_project['name'])
  end

  [
    [['owner_uuid', '!=', 'zzzzz-tpzed-xurymjxw79nv3jz'], 200,
        'zzzzz-j7d0g-publicfavorites', 'zzzzz-xvhdp-cr4queuedcontnr'],
    [["container_requests.state", "not in", ["Final"]], 200,
        'zzzzz-xvhdp-cr4queuedcontnr', 'zzzzz-xvhdp-cr4completedctr'],
    [['container_requests.requesting_container_uuid', '=', nil], 200,
        'zzzzz-xvhdp-cr4queuedcontnr', 'zzzzz-xvhdp-cr4requestercn2'],
    [['container_requests.no_such_column', '=', nil], 422],
    [['container_requests.', '=', nil], 422],
    [['.requesting_container_uuid', '=', nil], 422],
    [['no_such_table.uuid', '!=', 'zzzzz-tpzed-xurymjxw79nv3jz'], 422],
  ].each do |filter, expect_code, expect_uuid, not_expect_uuid|
    test "get contents with '#{filter}' filter" do
      authorize_with :active
      get :contents, params: {filters: [filter], format: :json}
      assert_response expect_code
      if expect_code == 200
        assert_not_empty json_response['items']
        item_uuids = json_response['items'].collect {|item| item['uuid']}
        assert_includes(item_uuids, expect_uuid)
        assert_not_includes(item_uuids, not_expect_uuid)
      end
    end
  end

  test 'get contents with low max_index_database_read' do
    # Some result will certainly have at least 12 bytes in a
    # restricted column.
    #
    # We cannot use collections.manifest_text to test this, because
    # GroupsController refuses to select manifest_text, because
    # controller doesn't sign manifests in a groups#contents response.
    Rails.configuration.API.MaxIndexDatabaseRead = 12
    authorize_with :active
    get :contents, params: {
          uuid: users(:active).uuid,
          format: :json,
        }
    assert_response :success
    assert_not_empty(json_response['items'])
    assert_operator(json_response['items'].count,
                    :<, json_response['items_available'])
  end

  test 'get contents, recursive=true' do
    authorize_with :active
    params = {
      id: groups(:aproject).uuid,
      recursive: true,
      format: :json,
    }
    get :contents, params: params
    owners = json_response['items'].map do |item|
      item['owner_uuid']
    end
    assert_includes(owners, groups(:aproject).uuid)
    assert_includes(owners, groups(:asubproject).uuid)
  end

  [false, nil].each do |recursive|
    test "get contents, recursive=#{recursive.inspect}" do
      authorize_with :active
      params = {
        id: groups(:aproject).uuid,
        format: :json,
      }
      params[:recursive] = false if recursive == false
      get :contents, params: params
      owners = json_response['items'].map do |item|
        item['owner_uuid']
      end
      assert_includes(owners, groups(:aproject).uuid)
      refute_includes(owners, groups(:asubproject).uuid)
    end
  end

  test 'get home project contents, recursive=true' do
    authorize_with :active
    get :contents, params: {
          id: users(:active).uuid,
          recursive: true,
          format: :json,
        }
    owners = json_response['items'].map do |item|
      item['owner_uuid']
    end
    assert_includes(owners, users(:active).uuid)
    assert_includes(owners, groups(:aproject).uuid)
    assert_includes(owners, groups(:asubproject).uuid)
  end

  [:afiltergroup, :private_role].each do |grp|
    test "delete non-project group #{grp}" do
      authorize_with :admin
      assert_not_nil Group.find_by_uuid(groups(grp).uuid)
      assert !Group.find_by_uuid(groups(grp).uuid).is_trashed
      post :destroy, params: {
            id: groups(grp).uuid,
            format: :json,
          }
      assert_response :success
      # Should be trashed
      assert Group.find_by_uuid(groups(grp).uuid).is_trashed
    end
  end

  [
    [false, :inactive, :private_role, false],
    [false, :spectator, :private_role, false],
    [false, :admin, :private_role, true],
    [true, :inactive, :private_role, false],
    [true, :spectator, :private_role, true],
    [true, :admin, :private_role, true],
    # project (non-role) groups are invisible even when RoleGroupsVisibleToAll is true
    [true, :inactive, :private, false],
    [true, :spectator, :private, false],
    [true, :admin, :private, true],
  ].each do |visibleToAll, userFixture, groupFixture, visible|
    test "with RoleGroupsVisibleToAll=#{visibleToAll}, #{groupFixture} group is #{visible ? '' : 'in'}visible to #{userFixture} user" do
      Rails.configuration.Users.RoleGroupsVisibleToAll = visibleToAll
      authorize_with userFixture
      get :show, params: {id: groups(groupFixture).uuid, format: :json}
      if visible
        assert_response :success
      else
        assert_response 404
      end
    end
  end

  ### trashed project tests ###

  #
  # The structure is
  #
  # trashed_project         (zzzzz-j7d0g-trashedproject1)
  #   trashed_subproject    (zzzzz-j7d0g-trashedproject2)
  #   trashed_subproject3   (zzzzz-j7d0g-trashedproject3)
  #   zzzzz-xvhdp-cr5trashedcontr

  [:active,
   :admin].each do |auth|
    # project: to query,    to untrash,    is visible, parent contents listing success
    [
     [:trashed_project,     [],                 false, true],
     [:trashed_project,     [:trashed_project], true,  true],
     [:trashed_subproject,  [],                 false, false],
     [:trashed_subproject,  [:trashed_project], true,  true],
     [:trashed_subproject3, [:trashed_project], false, true],
     [:trashed_subproject3, [:trashed_subproject3], false, false],
     [:trashed_subproject3, [:trashed_project, :trashed_subproject3], true, true],
    ].each do |project, untrash, visible, success|

      test "contents listing #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :contents, params: {
              id: groups(project).owner_uuid,
              format: :json
            }
        if success
          assert_response :success
          item_uuids = json_response['items'].map do |item|
            item['uuid']
          end
          if visible
            assert_includes(item_uuids, groups(project).uuid)
          else
            assert_not_includes(item_uuids, groups(project).uuid)
          end
        else
          assert_response 404
        end
      end

      test "contents of #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :contents, params: {
              id: groups(project).uuid,
              format: :json
            }
        if visible
          assert_response :success
        else
          assert_response 404
        end
      end

      test "index #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :index, params: {
              format: :json,
            }
        assert_response :success
        item_uuids = json_response['items'].map do |item|
          item['uuid']
        end
        if visible
          assert_includes(item_uuids, groups(project).uuid)
        else
          assert_not_includes(item_uuids, groups(project).uuid)
        end
      end

      test "show #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :show, params: {
              id: groups(project).uuid,
              format: :json
            }
        if visible
          assert_response :success
        else
          assert_response 404
        end
      end

      test "show include_trash=false #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :show, params: {
              id: groups(project).uuid,
              format: :json,
              include_trash: false
            }
        if visible
          assert_response :success
        else
          assert_response 404
        end
      end

      test "show include_trash #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :show, params: {
              id: groups(project).uuid,
              format: :json,
              include_trash: true
            }
        assert_response :success
      end

      test "index include_trash #{project} #{untrash} as #{auth}" do
        authorize_with auth
        untrash.each do |pr|
          Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
        end
        get :index, params: {
              format: :json,
              include_trash: true
            }
        assert_response :success
        item_uuids = json_response['items'].map do |item|
          item['uuid']
        end
        assert_includes(item_uuids, groups(project).uuid)
      end
    end

    test "delete project #{auth}" do
      authorize_with auth
      [:trashed_project].each do |pr|
        Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
      end
      assert !Group.find_by_uuid(groups(:trashed_project).uuid).is_trashed
      post :destroy, params: {
            id: groups(:trashed_project).uuid,
            format: :json,
          }
      assert_response :success
      assert Group.find_by_uuid(groups(:trashed_project).uuid).is_trashed
    end

    test "untrash project #{auth}" do
      authorize_with auth
      assert Group.find_by_uuid(groups(:trashed_project).uuid).is_trashed
      post :untrash, params: {
            id: groups(:trashed_project).uuid,
            format: :json,
          }
      assert_response :success
      assert !Group.find_by_uuid(groups(:trashed_project).uuid).is_trashed
    end

    test "untrash project with name conflict #{auth}" do
      authorize_with auth
      [:trashed_project].each do |pr|
        Group.find_by_uuid(groups(pr).uuid).update! is_trashed: false
      end
      gc = Group.create!({owner_uuid: "zzzzz-j7d0g-trashedproject1",
                         name: "trashed subproject 3",
                         group_class: "project"})
      post :untrash, params: {
            id: groups(:trashed_subproject3).uuid,
            format: :json,
            ensure_unique_name: true
           }
      assert_response :success
      assert_match /^trashed subproject 3 \(#{json_response['uuid'][-15..-1]}\)$/, json_response['name']
    end

    test "move trashed subproject to new owner #{auth}" do
      authorize_with auth
      assert_nil Group.readable_by(users(auth)).where(uuid: groups(:trashed_subproject).uuid).first
      put :update, params: {
            id: groups(:trashed_subproject).uuid,
            group: {
              owner_uuid: users(:active).uuid
            },
            include_trash: true,
            format: :json,
          }
      assert_response :success
      assert_not_nil Group.readable_by(users(auth)).where(uuid: groups(:trashed_subproject).uuid).first
    end
  end

  # the group class overrides the destroy method. Make sure that the destroyed
  # object is returned
  [
    {group_class: "project"},
    {group_class: "role"},
    {group_class: "filter", properties: {"filters":[]}},
  ].each do |params|
    test "destroy group #{params} returns object" do
      authorize_with :active

      group = Group.create!(params)

      post :destroy, params: {
            id: group.uuid,
            format: :json,
          }
      assert_response :success
      assert_not_nil json_response
      assert_equal group.uuid, json_response["uuid"]
    end
  end

  test 'get shared owned by another user' do
    authorize_with :user_bar_in_sharing_group

    act_as_system_user do
      Link.create!(
        tail_uuid: users(:user_bar_in_sharing_group).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:project_owned_by_foo).uuid)
    end

    get :shared, params: {:filters => [["group_class", "=", "project"]], :include => "owner_uuid"}

    assert_equal 1, json_response['items'].length
    assert_equal json_response['items'][0]["uuid"], groups(:project_owned_by_foo).uuid

    assert_equal 1, json_response['included'].length
    assert_equal json_response['included'][0]["uuid"], users(:user_foo_in_sharing_group).uuid
  end

  test 'get shared, owned by unreadable project' do
    authorize_with :user_bar_in_sharing_group

    act_as_system_user do
      Group.find_by_uuid(groups(:project_owned_by_foo).uuid).update!(owner_uuid: groups(:aproject).uuid)
      Link.create!(
        tail_uuid: users(:user_bar_in_sharing_group).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:project_owned_by_foo).uuid)
    end

    get :shared, params: {:filters => [["group_class", "=", "project"]], :include => "owner_uuid"}

    assert_equal 1, json_response['items'].length
    assert_equal json_response['items'][0]["uuid"], groups(:project_owned_by_foo).uuid

    assert_equal 0, json_response['included'].length
  end

  test 'get shared, add permission link' do
    authorize_with :user_bar_in_sharing_group

    act_as_system_user do
      Link.create!(tail_uuid: groups(:group_for_sharing_tests).uuid,
                   head_uuid: groups(:project_owned_by_foo).uuid,
                   link_class: 'permission',
                   name: 'can_manage')
    end

    get :shared, params: {:filters => [["group_class", "=", "project"]], :include => "owner_uuid"}

    assert_equal 1, json_response['items'].length
    assert_equal groups(:project_owned_by_foo).uuid, json_response['items'][0]["uuid"]

    assert_equal 1, json_response['included'].length
    assert_equal users(:user_foo_in_sharing_group).uuid, json_response['included'][0]["uuid"]
  end

  ### contents with exclude_home_project

  test 'contents, exclude home owned by another user' do
    authorize_with :user_bar_in_sharing_group

    act_as_system_user do
      Link.create!(
        tail_uuid: users(:user_bar_in_sharing_group).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:project_owned_by_foo).uuid)
      Link.create!(
        tail_uuid: users(:user_bar_in_sharing_group).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: collections(:collection_owned_by_foo).uuid)
    end

    get :contents, params: {:include => "owner_uuid", :exclude_home_project => true}

    assert_equal 2, json_response['items'].length
    assert_equal json_response['items'][0]["uuid"], groups(:project_owned_by_foo).uuid
    assert_equal json_response['items'][1]["uuid"], collections(:collection_owned_by_foo).uuid

    assert_equal 1, json_response['included'].length
    assert_equal json_response['included'][0]["uuid"], users(:user_foo_in_sharing_group).uuid
  end

  test 'contents, exclude home, owned by unreadable project' do
    authorize_with :user_bar_in_sharing_group

    act_as_system_user do
      Group.find_by_uuid(groups(:project_owned_by_foo).uuid).update!(owner_uuid: groups(:aproject).uuid)
      Link.create!(
        tail_uuid: users(:user_bar_in_sharing_group).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:project_owned_by_foo).uuid)
    end

    get :contents, params: {:include => "owner_uuid", :exclude_home_project => true}

    assert_equal 1, json_response['items'].length
    assert_equal json_response['items'][0]["uuid"], groups(:project_owned_by_foo).uuid

    assert_equal 0, json_response['included'].length
  end

  test 'contents, exclude home, add permission link' do
    authorize_with :user_bar_in_sharing_group

    act_as_system_user do
      Link.create!(tail_uuid: groups(:group_for_sharing_tests).uuid,
                   head_uuid: groups(:project_owned_by_foo).uuid,
                   link_class: 'permission',
                   name: 'can_manage')
    end

    get :contents, params: {:include => "owner_uuid", :exclude_home_project => true}
    assert_response 200

    assert_equal 1, json_response['items'].length
    assert_equal groups(:project_owned_by_foo).uuid, json_response['items'][0]["uuid"]

    assert_equal 1, json_response['included'].length
    assert_equal users(:user_foo_in_sharing_group).uuid, json_response['included'][0]["uuid"]
  end

  test 'contents, exclude home, with parent specified' do
    authorize_with :active

    get :contents, params: {id: groups(:aproject).uuid, :include => "owner_uuid", :exclude_home_project => true}

    assert_response 422
  end

  [[false, 'owner_uuid'],
   [false, []],
   [false, ''],
   [true, 'container_uuid'],
   [true, ['container_uuid']],
   [true, ['owner_uuid', 'container_uuid'], ['uuid', 'container_uuid', 'state', 'output']],
  ].each do |check_container_included, include_param, select_param|
    test "contents, include=#{include_param.inspect}" do
      authorize_with :active
      get :contents, params: {
            :id => users(:active).uuid,
            :include => include_param,
            :limit => 1000,
            :select => select_param,
          }
      assert_response 200
      if include_param.empty?
        assert_equal false, json_response.include?('included')
        return
      end
      incl = {}
      json_response['included'].andand.each do |ctr|
        incl[ctr['uuid']] = ctr
      end
      next if !check_container_included
      checked_crs = 0
      json_response['items'].each do |item|
        next if !item['container_uuid']
        assert_equal item['container_uuid'], incl[item['container_uuid']]['uuid']
        assert_not_empty incl[item['container_uuid']]['state']
        checked_crs += 1
      end
      assert_operator 0, :<, checked_crs
    end
  end

  test "include_trash does not return trash inside frozen project" do
    authorize_with :active
    trashtime = Time.now - 1.second
    outerproj = Group.create!(group_class: 'project')
    innerproj = Group.create!(group_class: 'project', owner_uuid: outerproj.uuid)
    innercoll = Collection.create!(name: 'inner-not-trashed', owner_uuid: innerproj.uuid)
    innertrash = Collection.create!(name: 'inner-trashed', owner_uuid: innerproj.uuid, trash_at: trashtime)
    innertrashproj = Group.create!(group_class: 'project', name: 'inner-trashed-proj', owner_uuid: innerproj.uuid, trash_at: trashtime)
    outertrash = Collection.create!(name: 'outer-trashed', owner_uuid: outerproj.uuid, trash_at: trashtime)
    innerproj.update!(frozen_by_uuid: users(:active).uuid)
    get :contents, params: {id: outerproj.uuid, include_trash: true, recursive: true}
    assert_response :success
    uuids = json_response['items'].collect { |item| item['uuid'] }
    assert_includes uuids, outertrash.uuid
    assert_includes uuids, innerproj.uuid
    assert_includes uuids, innercoll.uuid
    refute_includes uuids, innertrash.uuid
    refute_includes uuids, innertrashproj.uuid
  end
end
