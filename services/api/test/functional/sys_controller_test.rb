# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class SysControllerTest < ActionController::TestCase
  include CurrentApiClient
  include DbCurrentTime

  test "trash_sweep - delete expired tokens" do
    assert_not_empty ApiClientAuthorization.where(uuid: api_client_authorizations(:expired).uuid)
    authorize_with :admin
    post :trash_sweep
    assert_response :success
    assert_empty ApiClientAuthorization.where(uuid: api_client_authorizations(:expired).uuid)
  end

  test "trash_sweep - fail with non-admin token" do
    authorize_with :active
    post :trash_sweep
    assert_response 403
  end

  test "trash_sweep - move collections to trash" do
    c = collections(:trashed_on_next_sweep)
    refute_empty Collection.where('uuid=? and is_trashed=false', c.uuid)
    assert_raises(ActiveRecord::RecordNotUnique) do
      act_as_user users(:active) do
        Collection.create!(owner_uuid: c.owner_uuid,
                           name: c.name)
      end
    end
    authorize_with :admin
    post :trash_sweep
    assert_response :success
    c = Collection.where('uuid=? and is_trashed=true', c.uuid).first
    assert c
    act_as_user users(:active) do
      assert Collection.create!(owner_uuid: c.owner_uuid,
                                name: c.name)
    end
  end

  test "trash_sweep - delete collections" do
    uuid = 'zzzzz-4zz18-3u1p5umicfpqszp' # deleted_on_next_sweep
    assert_not_empty Collection.where(uuid: uuid)
    authorize_with :admin
    post :trash_sweep
    assert_response :success
    assert_empty Collection.where(uuid: uuid)
  end

  test "trash_sweep - delete referring links" do
    uuid = collections(:trashed_on_next_sweep).uuid
    act_as_system_user do
      assert_raises ActiveRecord::RecordInvalid do
        # Cannot create because :trashed_on_next_sweep is already trashed
        Link.create!(head_uuid: uuid,
                     tail_uuid: system_user_uuid,
                     link_class: 'whatever',
                     name: 'something')
      end

      # Bump trash_at to now + 1 minute
      Collection.where(uuid: uuid).
        update(trash_at: db_current_time + (1).minute)

      # Not considered trashed now
      Link.create!(head_uuid: uuid,
                   tail_uuid: system_user_uuid,
                   link_class: 'whatever',
                   name: 'something')
    end
    past = db_current_time
    Collection.where(uuid: uuid).
      update_all(is_trashed: true, trash_at: past, delete_at: past)
    assert_not_empty Collection.where(uuid: uuid)
    authorize_with :admin
    post :trash_sweep
    assert_response :success
    assert_empty Collection.where(uuid: uuid)
  end

  test "trash_sweep - move projects to trash" do
    p = groups(:trashed_on_next_sweep)
    assert_empty Group.where('uuid=? and is_trashed=true', p.uuid)
    authorize_with :admin
    post :trash_sweep
    assert_response :success
    assert_not_empty Group.where('uuid=? and is_trashed=true', p.uuid)
  end

  test "trash_sweep - role groups are deleted" do
    p = groups(:trashed_role_on_next_sweep)
    assert_empty Group.where('uuid=? and is_trashed=true', p.uuid)
    assert_not_empty Link.where(uuid: links(:foo_file_readable_by_soon_to_be_trashed_role).uuid)
    authorize_with :admin
    post :trash_sweep
    assert_response :success
    assert_empty Group.where(uuid: p.uuid)
    assert_empty Link.where(uuid: links(:foo_file_readable_by_soon_to_be_trashed_role).uuid)
  end

  test "trash_sweep - delete projects and their contents" do
    g_foo = groups(:trashed_project)
    g_bar = groups(:trashed_subproject)
    g_baz = groups(:trashed_subproject3)
    col = collections(:collection_in_trashed_subproject)
    cr = container_requests(:cr_in_trashed_project)
    # Save how many objects were before the sweep
    user_nr_was = User.all.length
    coll_nr_was = Collection.all.length
    group_nr_was = Group.where('group_class<>?', 'project').length
    project_nr_was = Group.where(group_class: 'project').length
    cr_nr_was = ContainerRequest.all.length
    assert_not_empty Group.where(uuid: g_foo.uuid)
    assert_not_empty Group.where(uuid: g_bar.uuid)
    assert_not_empty Group.where(uuid: g_baz.uuid)
    assert_not_empty Collection.where(uuid: col.uuid)
    assert_not_empty ContainerRequest.where(uuid: cr.uuid)

    authorize_with :admin
    Group.find_by_uuid(g_foo.uuid).update!(delete_at: Time.now - 1.second)

    post :trash_sweep
    assert_response :success

    assert_empty Group.where(uuid: g_foo.uuid)
    assert_empty Group.where(uuid: g_bar.uuid)
    assert_empty Group.where(uuid: g_baz.uuid)
    assert_empty Collection.where(uuid: col.uuid)
    assert_empty ContainerRequest.where(uuid: cr.uuid)
    # No unwanted deletions should have happened
    assert_equal user_nr_was, User.all.length
    assert_equal coll_nr_was-2,        # collection_in_trashed_subproject
                 Collection.all.length # & deleted_on_next_sweep collections
    assert_equal group_nr_was-1,       # trashed_role_on_next_sweep
                 Group.where('group_class<>?', 'project').length
    assert_equal project_nr_was-3, Group.where(group_class: 'project').length
    assert_equal cr_nr_was-1, ContainerRequest.all.length
  end

  test "trash_sweep - delete unused uuid_locks" do
    uuid_active = "zzzzz-zzzzz-uuidlockstest11"
    uuid_inactive = "zzzzz-zzzzz-uuidlockstest00"

    ready = Queue.new
    insertsql = "INSERT INTO uuid_locks (uuid) VALUES ($1) ON CONFLICT (uuid) do UPDATE SET n = uuid_locks.n+1"
    url = ENV["DATABASE_URL"].sub(/\?.*/, '')
    Thread.new do
      conn = PG::Connection.new(url)
      conn.exec_params(insertsql, [uuid_active])
      conn.exec_params(insertsql, [uuid_inactive])
      conn.transaction do |conn|
        conn.exec_params(insertsql, [uuid_active])
        ready << true
        # If we keep this transaction open while trash_sweep runs, the
        # uuid_active row shouldn't get deleted.
        sleep 10
      rescue
        # Unblock main thread
        ready << false
        raise
      end
    end
    assert_equal true, ready.pop
    authorize_with :admin
    post :trash_sweep
    rows = ActiveRecord::Base.connection.exec_query("SELECT uuid FROM uuid_locks ORDER BY uuid", "", []).rows
    assert_includes(rows, [uuid_active], "row with active lock (still held by thread) should not have been deleted")
    refute_includes(rows, [uuid_inactive], "row with inactive lock should have been deleted")
  end
end
