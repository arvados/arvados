# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class PermissionTest < ActiveSupport::TestCase
  include CurrentApiClient

  test "Grant permissions on an object I own" do
    set_user_from_auth :active_trustedclient

    ob = Collection.create
    assert ob.save

    # Ensure I have permission to manage this group even when its owner changes
    perm_link = Link.create(tail_uuid: users(:active).uuid,
                            head_uuid: ob.uuid,
                            link_class: 'permission',
                            name: 'can_manage')
    assert perm_link.save, "should give myself permission on my own object"
  end

  test "Delete permission links when deleting an object" do
    set_user_from_auth :active_trustedclient

    ob = Collection.create!
    Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: ob.uuid,
                 link_class: 'permission',
                 name: 'can_manage')
    ob_uuid = ob.uuid
    assert ob.destroy, "Could not destroy object with 1 permission link"
    assert_empty(Link.where(head_uuid: ob_uuid),
                 "Permission link was not deleted when object was deleted")
  end

  test "permission links owned by root" do
    set_user_from_auth :active_trustedclient
    ob = Collection.create!
    perm_link = Link.create!(tail_uuid: users(:active).uuid,
                             head_uuid: ob.uuid,
                             link_class: 'permission',
                             name: 'can_read')
    assert_equal system_user_uuid, perm_link.owner_uuid
  end

  test "readable_by" do
    set_user_from_auth :admin

    ob = Collection.create!
    Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: ob.uuid,
                 link_class: 'permission',
                 name: 'can_read')
    assert Collection.readable_by(users(:active)).where(uuid: ob.uuid).any?, "user does not have read permission"
  end

  test "writable_by" do
    set_user_from_auth :admin

    ob = Collection.create!
    Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: ob.uuid,
                 link_class: 'permission',
                 name: 'can_write')
    assert ob.writable_by.include?(users(:active).uuid), "user does not have write permission"
  end

  test "update permission link" do
    set_user_from_auth :admin

    grp = Group.create! name: "blah project", group_class: "project"
    ob = Collection.create! owner_uuid: grp.uuid

    assert !users(:active).can?(write: ob)
    assert !users(:active).can?(read: ob)

    l1 = Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_write')

    assert users(:active).can?(write: ob)
    assert users(:active).can?(read: ob)

    l1.update_attributes!(name: 'can_read')

    assert !users(:active).can?(write: ob)
    assert users(:active).can?(read: ob)

    l1.destroy

    assert !users(:active).can?(write: ob)
    assert !users(:active).can?(read: ob)
  end

  test "writable_by reports requesting user's own uuid for a writable project" do
    invited_to_write = users(:project_viewer)
    group = groups(:asubproject)

    # project_view can read, but cannot see write or see writers list
    set_user_from_auth :project_viewer
    assert_equal([group.owner_uuid],
                 group.writable_by,
                 "writers list should just have owner_uuid")

    # allow project_viewer to write for the remainder of the test
    set_user_from_auth :admin
    Link.create!(tail_uuid: invited_to_write.uuid,
                 head_uuid: group.uuid,
                 link_class: 'permission',
                 name: 'can_write')
    group.permissions.reload

    # project_viewer should see self in writers list (but not all writers)
    set_user_from_auth :project_viewer
    assert_not_nil(group.writable_by,
                    "can write but cannot see writers list")
    assert_includes(group.writable_by, invited_to_write.uuid,
                    "self missing from writers list")
    assert_includes(group.writable_by, group.owner_uuid,
                    "project owner missing from writers list")
    refute_includes(group.writable_by, users(:active).uuid,
                    "saw :active user in writers list")

    # active user should see full writers list
    set_user_from_auth :active
    assert_includes(group.writable_by, invited_to_write.uuid,
                    "permission just added, but missing from writers list")

    # allow project_viewer to manage for the remainder of the test
    set_user_from_auth :admin
    Link.create!(tail_uuid: invited_to_write.uuid,
                 head_uuid: group.uuid,
                 link_class: 'permission',
                 name: 'can_manage')
    # invite another writer we can test for
    Link.create!(tail_uuid: users(:spectator).uuid,
                 head_uuid: group.uuid,
                 link_class: 'permission',
                 name: 'can_write')
    group.permissions.reload

    set_user_from_auth :project_viewer
    assert_not_nil(group.writable_by,
                    "can manage but cannot see writers list")
    assert_includes(group.writable_by, users(:spectator).uuid,
                    ":spectator missing from writers list")
  end

  test "user owns group, group can_manage object's group, user can add permissions" do
    set_user_from_auth :admin

    owner_grp = Group.create!(owner_uuid: users(:active).uuid, group_class: "role")

    sp_grp = Group.create!(group_class: "project")

    Link.create!(link_class: 'permission',
                 name: 'can_manage',
                 tail_uuid: owner_grp.uuid,
                 head_uuid: sp_grp.uuid)

    sp = Collection.create!(owner_uuid: sp_grp.uuid)

    # active user owns owner_grp, which has can_manage permission on sp_grp
    # user should be able to add permissions on sp.
    set_user_from_auth :active_trustedclient
    test_perm = Link.create(tail_uuid: users(:active).uuid,
                            head_uuid: sp.uuid,
                            link_class: 'permission',
                            name: 'can_write')
    assert test_perm.save, "could not save new permission on target object"
    assert test_perm.destroy, "could not delete new permission on target object"
  end

  # bug #3091
  skip "can_manage permission on a non-group object" do
    set_user_from_auth :admin

    ob = Collection.create!
    # grant can_manage permission to active
    perm_link = Link.create!(tail_uuid: users(:active).uuid,
                             head_uuid: ob.uuid,
                             link_class: 'permission',
                             name: 'can_manage')
    # ob is owned by :admin, the link is owned by root
    assert_equal users(:admin).uuid, ob.owner_uuid
    assert_equal system_user_uuid, perm_link.owner_uuid

    # user "active" can modify the permission link
    set_user_from_auth :active_trustedclient
    perm_link.properties["foo"] = 'bar'
    assert perm_link.save, "could not save modified link"

    assert_equal 'bar', perm_link.properties['foo'], "link properties do not include foo = bar"
  end

  test "user without can_manage permission may not modify permission link" do
    set_user_from_auth :admin

    ob = Collection.create!
    # grant can_manage permission to active
    perm_link = Link.create!(tail_uuid: users(:active).uuid,
                             head_uuid: ob.uuid,
                             link_class: 'permission',
                             name: 'can_read')
    # ob is owned by :admin, the link is owned by root
    assert_equal ob.owner_uuid, users(:admin).uuid
    assert_equal perm_link.owner_uuid, system_user_uuid

    # user "active" may not modify the permission link
    set_user_from_auth :active_trustedclient
    perm_link.name = 'can_manage'
    assert_raises ArvadosModel::PermissionDeniedError do
      perm_link.save
    end
  end

  test "manager user gets permission to minions' articles via can_manage link" do
    Rails.configuration.Users.RoleGroupsVisibleToAll = false
    Rails.configuration.Users.ActivatedUsersAreVisibleToOthers = false
    manager = create :active_user, first_name: "Manage", last_name: "Er"
    minion = create :active_user, first_name: "Min", last_name: "Ion"
    minions_specimen = act_as_user minion do
      g = Group.create! name: "minon project", group_class: "project"
      Collection.create! owner_uuid: g.uuid
    end
    # Manager creates a group. (Make sure it doesn't magically give
    # anyone any additional permissions.)
    g = nil
    act_as_user manager do
      g = create :group, name: "NoBigSecret Lab", group_class: "role"
      assert_empty(User.readable_by(manager).where(uuid: minion.uuid),
                   "saw a user I shouldn't see")
      assert_raises(ArvadosModel::PermissionDeniedError,
                    ActiveRecord::RecordInvalid,
                    "gave can_read permission to a user I shouldn't see") do
        create(:permission_link,
               name: 'can_read', tail_uuid: minion.uuid, head_uuid: g.uuid)
      end
      %w(can_manage can_write can_read).each do |perm_type|
        assert_raises(ArvadosModel::PermissionDeniedError,
                      ActiveRecord::RecordInvalid,
                      "escalated privileges") do
          create(:permission_link,
                 name: perm_type, tail_uuid: g.uuid, head_uuid: minion.uuid)
        end
      end
      assert_empty(User.readable_by(manager).where(uuid: minion.uuid),
                   "manager saw minion too soon")
      assert_empty(User.readable_by(minion).where(uuid: manager.uuid),
                   "minion saw manager too soon")
      assert_empty(Group.readable_by(minion).where(uuid: g.uuid),
                   "minion saw manager's new NoBigSecret Lab group too soon")

      # Manager declares everybody on the system should be able to see
      # the NoBigSecret Lab group.
      create(:permission_link,
             name: 'can_read',
             tail_uuid: 'zzzzz-j7d0g-fffffffffffffff',
             head_uuid: g.uuid)
      # ...but nobody has joined the group yet. Manager still can't see
      # minion.
      assert_empty(User.readable_by(manager).where(uuid: minion.uuid),
                   "manager saw minion too soon")
    end

    act_as_user minion do
      # Minion can see the group.
      assert_not_empty(Group.readable_by(minion).where(uuid: g.uuid),
                       "minion could not see the NoBigSecret Lab group")
      # Minion joins the group.
      create(:permission_link,
             name: 'can_read',
             tail_uuid: g.uuid,
             head_uuid: minion.uuid)
    end

    act_as_user manager do
      # Now, manager can see minion.
      assert_not_empty(User.readable_by(manager).where(uuid: minion.uuid),
                       "manager could not see minion")
      # But cannot obtain further privileges this way.
      assert_raises(ArvadosModel::PermissionDeniedError,
                    "escalated privileges") do
        create(:permission_link,
               name: 'can_manage', tail_uuid: manager.uuid, head_uuid: minion.uuid)
      end
      assert_empty(Collection
                     .readable_by(manager)
                     .where(uuid: minions_specimen.uuid),
                   "manager saw the minion's private stuff")
      assert_raises(ArvadosModel::PermissionDeniedError,
                   "manager could update minion's private stuff") do
        minions_specimen.update_attributes(properties: {'x' => 'y'})
      end
    end

    act_as_system_user do
      # Root can give Manager more privileges over Minion.
      create(:permission_link,
             name: 'can_manage', tail_uuid: g.uuid, head_uuid: minion.uuid)
    end

    act_as_user manager do
      # Now, manager can read and write Minion's stuff.
      assert_not_empty(Collection
                         .readable_by(manager)
                         .where(uuid: minions_specimen.uuid),
                       "manager could not find minion's specimen by uuid")
      assert_equal(true,
                   minions_specimen.update_attributes(properties: {'x' => 'y'}),
                   "manager could not update minion's specimen object")
    end
  end

  test "users with bidirectional read permission in group can see each other, but cannot see each other's private articles" do
    Rails.configuration.Users.ActivatedUsersAreVisibleToOthers = false
    a = create :active_user, first_name: "A"
    b = create :active_user, first_name: "B"
    other = create :active_user, first_name: "OTHER"

    assert_empty(User.readable_by(b).where(uuid: a.uuid),
                     "#{b.first_name} should not be able to see 'a' in the user list")
    assert_empty(User.readable_by(a).where(uuid: b.uuid),
                     "#{a.first_name} should not be able to see 'b' in the user list")

    act_as_system_user do
      g = create :group, group_class: "role"
      [a,b].each do |u|
        create(:permission_link,
               name: 'can_read', tail_uuid: u.uuid, head_uuid: g.uuid)
        create(:permission_link,
               name: 'can_read', head_uuid: u.uuid, tail_uuid: g.uuid)
      end
    end

    assert_not_empty(User.readable_by(b).where(uuid: a.uuid),
                     "#{b.first_name} should be able to see 'a' in the user list")
    assert_not_empty(User.readable_by(a).where(uuid: b.uuid),
                     "#{a.first_name} should be able to see 'b' in the user list")

    a_specimen = act_as_user a do
      Collection.create!
    end
    assert_not_empty(Collection.readable_by(a).where(uuid: a_specimen.uuid),
                     "A cannot read own Collection, following test probably useless.")
    assert_empty(Collection.readable_by(b).where(uuid: a_specimen.uuid),
                 "B can read A's Collection")
    [a,b].each do |u|
      assert_empty(User.readable_by(u).where(uuid: other.uuid),
                   "#{u.first_name} can see OTHER in the user list")
      assert_empty(User.readable_by(other).where(uuid: u.uuid),
                   "OTHER can see #{u.first_name} in the user list")
      act_as_user u do
        assert_raises ArvadosModel::PermissionDeniedError, "wrote without perm" do
          other.update_attributes!(prefs: {'pwned' => true})
        end
        assert_equal(true, u.update_attributes!(prefs: {'thisisme' => true}),
                     "#{u.first_name} can't update its own prefs")
      end
      act_as_user other do
        assert_raises(ArvadosModel::PermissionDeniedError,
                        "OTHER wrote #{u.first_name} without perm") do
          u.update_attributes!(prefs: {'pwned' => true})
        end
        assert_equal(true, other.update_attributes!(prefs: {'thisisme' => true}),
                     "OTHER can't update its own prefs")
      end
    end
  end

  test "cannot create with owner = unwritable user" do
    set_user_from_auth :rominiadmin
    assert_raises ArvadosModel::PermissionDeniedError, "created with owner = unwritable user" do
      Collection.create!(owner_uuid: users(:active).uuid)
    end
  end

  test "cannot change owner to unwritable user" do
    set_user_from_auth :rominiadmin
    ob = Collection.create!
    assert_raises ArvadosModel::PermissionDeniedError, "changed owner to unwritable user" do
      ob.update_attributes!(owner_uuid: users(:active).uuid)
    end
  end

  test "cannot create with owner = unwritable group" do
    set_user_from_auth :rominiadmin
    assert_raises ArvadosModel::PermissionDeniedError, "created with owner = unwritable group" do
      Collection.create!(owner_uuid: groups(:aproject).uuid)
    end
  end

  test "cannot change owner to unwritable group" do
    set_user_from_auth :rominiadmin
    ob = Collection.create!
    assert_raises ArvadosModel::PermissionDeniedError, "changed owner to unwritable group" do
      ob.update_attributes!(owner_uuid: groups(:aproject).uuid)
    end
  end

  def container_logs(container, user)
    Log.readable_by(users(user)).
      where(object_uuid: containers(container).uuid, event_type: "test")
  end

  test "container logs created by dispatch are visible to container requestor" do
    set_user_from_auth :dispatch1
    Log.create!(object_uuid: containers(:running).uuid,
                event_type: "test")

    assert_not_empty container_logs(:running, :admin)
    assert_not_empty container_logs(:running, :active)
    assert_empty container_logs(:running, :spectator)
  end

  test "container logs created by dispatch are public if container request is public" do
    set_user_from_auth :dispatch1
    Log.create!(object_uuid: containers(:running_older).uuid,
                event_type: "test")

    assert_not_empty container_logs(:running_older, :anonymous)
  end

  test "add user to group, then remove them" do
    set_user_from_auth :admin
    grp = Group.create!(owner_uuid: system_user_uuid, group_class: "role")
    col = Collection.create!(owner_uuid: system_user_uuid)

    l0 = Link.create!(tail_uuid: grp.uuid,
                 head_uuid: col.uuid,
                 link_class: 'permission',
                 name: 'can_read')

    assert_empty Collection.readable_by(users(:active)).where(uuid: col.uuid)
    assert_empty User.readable_by(users(:active)).where(uuid: users(:project_viewer).uuid)

    l1 = Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_read')
    l2 = Link.create!(tail_uuid: grp.uuid,
                 head_uuid: users(:active).uuid,
                 link_class: 'permission',
                 name: 'can_read')

    l3 = Link.create!(tail_uuid: users(:project_viewer).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_read')
    l4 = Link.create!(tail_uuid: grp.uuid,
                 head_uuid: users(:project_viewer).uuid,
                 link_class: 'permission',
                 name: 'can_read')

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert User.readable_by(users(:active)).where(uuid: users(:project_viewer).uuid).first

    l1.destroy
    l2.destroy

    assert_empty Collection.readable_by(users(:active)).where(uuid: col.uuid)
    assert_empty User.readable_by(users(:active)).where(uuid: users(:project_viewer).uuid)

  end


  test "add user to group, then change permission level" do
    set_user_from_auth :admin
    grp = Group.create!(owner_uuid: system_user_uuid, group_class: "project")
    col = Collection.create!(owner_uuid: grp.uuid)
    assert_empty Collection.readable_by(users(:active)).where(uuid: col.uuid)
    assert_empty User.readable_by(users(:active)).where(uuid: users(:project_viewer).uuid)

    l1 = Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_manage')

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert users(:active).can?(read: col.uuid)
    assert users(:active).can?(write: col.uuid)
    assert users(:active).can?(manage: col.uuid)

    l1.name = 'can_read'
    l1.save!

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert users(:active).can?(read: col.uuid)
    assert !users(:active).can?(write: col.uuid)
    assert !users(:active).can?(manage: col.uuid)

    l1.name = 'can_write'
    l1.save!

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert users(:active).can?(read: col.uuid)
    assert users(:active).can?(write: col.uuid)
    assert !users(:active).can?(manage: col.uuid)
  end


  test "add user to group, then add overlapping permission link to group" do
    set_user_from_auth :admin
    grp = Group.create!(owner_uuid: system_user_uuid, group_class: "project")
    col = Collection.create!(owner_uuid: grp.uuid)
    assert_empty Collection.readable_by(users(:active)).where(uuid: col.uuid)
    assert_empty User.readable_by(users(:active)).where(uuid: users(:project_viewer).uuid)

    l1 = Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_manage')

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert users(:active).can?(read: col.uuid)
    assert users(:active).can?(write: col.uuid)
    assert users(:active).can?(manage: col.uuid)

    l3 = Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_read')

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert users(:active).can?(read: col.uuid)
    assert users(:active).can?(write: col.uuid)
    assert users(:active).can?(manage: col.uuid)

    # Creating l3 should have automatically deleted l1 and upgraded to
    # the max permission of {l1, l3}, i.e., can_manage (see #18693) so
    # there should be no can_read link now.
    refute Link.where(tail_uuid: l3.tail_uuid,
                      head_uuid: l3.head_uuid,
                      link_class: 'permission',
                      name: 'can_read').any?

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).first
    assert users(:active).can?(read: col.uuid)
    assert users(:active).can?(write: col.uuid)
    assert users(:active).can?(manage: col.uuid)
  end


  test "add user to group, then add overlapping permission link to subproject" do
    set_user_from_auth :admin
    grp = Group.create!(owner_uuid: system_user_uuid, group_class: "role")
    prj = Group.create!(owner_uuid: system_user_uuid, group_class: "project")

    l0 = Link.create!(tail_uuid: grp.uuid,
                 head_uuid: prj.uuid,
                 link_class: 'permission',
                 name: 'can_manage')

    assert_empty Group.readable_by(users(:active)).where(uuid: prj.uuid)
    assert_empty User.readable_by(users(:active)).where(uuid: users(:project_viewer).uuid)

    l1 = Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: grp.uuid,
                 link_class: 'permission',
                 name: 'can_manage')
    l2 = Link.create!(tail_uuid: grp.uuid,
                 head_uuid: users(:active).uuid,
                 link_class: 'permission',
                 name: 'can_read')

    assert Group.readable_by(users(:active)).where(uuid: prj.uuid).first
    assert users(:active).can?(read: prj.uuid)
    assert users(:active).can?(write: prj.uuid)
    assert users(:active).can?(manage: prj.uuid)

    l3 = Link.create!(tail_uuid: grp.uuid,
                 head_uuid: prj.uuid,
                 link_class: 'permission',
                 name: 'can_read')

    assert Group.readable_by(users(:active)).where(uuid: prj.uuid).first
    assert users(:active).can?(read: prj.uuid)
    assert users(:active).can?(write: prj.uuid)
    assert users(:active).can?(manage: prj.uuid)

    # Creating l3 should have automatically deleted l0 and upgraded to
    # the max permission of {l0, l3}, i.e., can_manage (see #18693) so
    # there should be no can_read link now.
    refute Link.where(tail_uuid: l3.tail_uuid,
                      head_uuid: l3.head_uuid,
                      link_class: 'permission',
                      name: 'can_read').any?

    assert Group.readable_by(users(:active)).where(uuid: prj.uuid).first
    assert users(:active).can?(read: prj.uuid)
    assert users(:active).can?(write: prj.uuid)
    assert users(:active).can?(manage: prj.uuid)
  end

  [system_user_uuid, anonymous_user_uuid].each do |u|
    test "cannot delete system user #{u}" do
      act_as_system_user do
        assert_raises ArvadosModel::PermissionDeniedError do
          User.find_by_uuid(u).destroy
        end
      end
    end
  end

  [system_group_uuid, anonymous_group_uuid, public_project_uuid].each do |g|
    test "cannot delete system group #{g}" do
      act_as_system_user do
        assert_raises ArvadosModel::PermissionDeniedError do
          Group.find_by_uuid(g).destroy
        end
      end
    end
  end

  # Show query plan for readable_by query. The plan for a test db
  # might not resemble the plan for a production db, but it doesn't
  # hurt to show the test db plan in test logs, and the .
  [false, true].each do |include_trash|
    test "query plan, include_trash=#{include_trash}" do
      sql = Collection.readable_by(users(:active), include_trash: include_trash).to_sql
      sql = "explain analyze #{sql}"
      STDERR.puts sql
      q = ActiveRecord::Base.connection.exec_query(sql)
      q.rows.each do |row| STDERR.puts(row) end
    end
  end
end
