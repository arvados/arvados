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

  test "users with bidirectional read permission in group can see each other, but cannot see each other's private articles" do
    a = create :active_user first_name: "A"
    b = create :active_user first_name: "B"
    other = create :active_user first_name: "OTHER"
    act_as_system_user do
      g = create :group
      [a,b].each do |u|
        create(:permission_link,
               name: 'can_read', tail_uuid: u.uuid, head_uuid: g.uuid)
        create(:permission_link,
               name: 'can_read', head_uuid: u.uuid, tail_uuid: g.uuid)
      end
    end
    a_specimen = act_as_user a do
      Specimen.create!
    end
    assert_not_empty(Specimen.readable_by(a).where(uuid: a_specimen.uuid),
                     "A cannot read own Specimen, following test probably useless.")
    assert_empty(Specimen.readable_by(b).where(uuid: a_specimen.uuid),
                 "B can read A's Specimen")
    [a,b].each do |u|
      assert_empty(User.readable_by(u).where(uuid: other.uuid),
                   "#{u.first_name} can see OTHER in the user list")
      assert_empty(User.readable_by(other).where(uuid: u.uuid),
                   "OTHER can see #{u.first_name} in the user list")
      act_as_user u do
        assert_raises ArvadosModel::PermissionDeniedError, "wrote without perm" do
          other.update_attributes!(prefs: {'pwned' => true})
        end
        assert_equal true, u.update_attributes!(prefs: {'thisisme' => true})
      end
      act_as_user other do
        ([other, a, b] - [u]).each do |x|
          assert_raises ArvadosModel::PermissionDeniedError, "wrote without perm" do
            x.update_attributes!(prefs: {'pwned' => true})
          end
        end
        assert_equal true, other.update_attributes!(prefs: {'thisisme' => true})
      end
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
