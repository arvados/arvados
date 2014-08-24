require 'test_helper'

class PermissionTest < ActiveSupport::TestCase
  include CurrentApiClient

  test "Grant permissions on an object I own" do
    set_user_from_auth :active_trustedclient

    ob = Specimen.create
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

    ob = Specimen.create!
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
    ob = Specimen.create!
    perm_link = Link.create!(tail_uuid: users(:active).uuid,
                             head_uuid: ob.uuid,
                             link_class: 'permission',
                             name: 'can_read')
    assert_equal system_user_uuid, perm_link.owner_uuid
  end

  test "readable_by" do
    set_user_from_auth :active_trustedclient

    ob = Specimen.create!
    Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: ob.uuid,
                 link_class: 'permission',
                 name: 'can_read')
    assert Specimen.readable_by(users(:active)).where(uuid: ob.uuid).any?, "user does not have read permission"
  end

  test "writable_by" do
    set_user_from_auth :active_trustedclient

    ob = Specimen.create!
    Link.create!(tail_uuid: users(:active).uuid,
                 head_uuid: ob.uuid,
                 link_class: 'permission',
                 name: 'can_write')
    assert ob.writable_by.include?(users(:active).uuid), "user does not have write permission"
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

    owner_grp = Group.create!(owner_uuid: users(:active).uuid)

    sp_grp = Group.create!
    sp = Specimen.create!(owner_uuid: sp_grp.uuid)

    manage_perm = Link.create!(link_class: 'permission',
                               name: 'can_manage',
                               tail_uuid: owner_grp.uuid,
                               head_uuid: sp_grp.uuid)

    # active user owns owner_grp, which has can_manage permission on sp_grp
    # user should be able to add permissions on sp.
    set_user_from_auth :active_trustedclient
    test_perm = Link.create(tail_uuid: users(:active).uuid,
                            head_uuid: sp.uuid,
                            link_class: 'permission',
                            name: 'can_write')
    test_uuid = test_perm.uuid
    assert test_perm.save, "could not save new permission on target object"
    assert test_perm.destroy, "could not delete new permission on target object"
  end

  # TODO(twp): fix bug #3091, which should fix this test.
  test "can_manage permission on a non-group object" do
    skip
    set_user_from_auth :admin

    ob = Specimen.create!
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

    ob = Specimen.create!
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

  test "cannot create with owner = unwritable user" do
    set_user_from_auth :rominiadmin
    assert_raises ArvadosModel::PermissionDeniedError, "created with owner = unwritable user" do
      Specimen.create!(owner_uuid: users(:active).uuid)
    end
  end

  test "cannot change owner to unwritable user" do
    set_user_from_auth :rominiadmin
    ob = Specimen.create!
    assert_raises ArvadosModel::PermissionDeniedError, "changed owner to unwritable user" do
      ob.update_attributes!(owner_uuid: users(:active).uuid)
    end
  end

  test "cannot create with owner = unwritable group" do
    set_user_from_auth :rominiadmin
    assert_raises ArvadosModel::PermissionDeniedError, "created with owner = unwritable group" do
      Specimen.create!(owner_uuid: groups(:aproject).uuid)
    end
  end

  test "cannot change owner to unwritable group" do
    set_user_from_auth :rominiadmin
    ob = Specimen.create!
    assert_raises ArvadosModel::PermissionDeniedError, "changed owner to unwritable group" do
      ob.update_attributes!(owner_uuid: groups(:aproject).uuid)
    end
  end

end
