# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'fix_roles_projects'

class GroupTest < ActiveSupport::TestCase
  include DbCurrentTime

  test "cannot set owner_uuid to object with existing ownership cycle" do
    set_user_from_auth :active_trustedclient

    # First make sure we have lots of permission on the bad group by
    # renaming it to "{current name} is mine all mine"
    g = groups(:bad_group_has_ownership_cycle_b)
    g.name += " is mine all mine"
    assert g.save, "active user should be able to modify group #{g.uuid}"

    # Use the group as the owner of a new object
    s = Specimen.
      create(owner_uuid: groups(:bad_group_has_ownership_cycle_b).uuid)
    assert s.valid?, "ownership should pass validation #{s.errors.messages}"
    assert_equal false, s.save, "should not save object with #{g.uuid} as owner"

    # Use the group as the new owner of an existing object
    s = specimens(:in_aproject)
    s.owner_uuid = groups(:bad_group_has_ownership_cycle_b).uuid
    assert s.valid?, "ownership should pass validation"
    assert_equal false, s.save, "should not save object with #{g.uuid} as owner"
  end

  test "cannot create a new ownership cycle" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo", group_class: "project")
    g_bar = Group.create!(name: "bar", group_class: "project")

    g_foo.owner_uuid = g_bar.uuid
    assert g_foo.save, lambda { g_foo.errors.messages }
    g_bar.owner_uuid = g_foo.uuid
    assert g_bar.valid?, "ownership cycle should not prevent validation"
    assert_equal false, g_bar.save, "should not create an ownership loop"
    assert g_bar.errors.messages[:owner_uuid].join(" ").match(/ownership cycle/)
  end

  test "cannot create a single-object ownership cycle" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo", group_class: "project")
    assert g_foo.save

    # Ensure I have permission to manage this group even when its owner changes
    perm_link = Link.create!(tail_uuid: users(:active).uuid,
                            head_uuid: g_foo.uuid,
                            link_class: 'permission',
                            name: 'can_manage')
    assert perm_link.save

    g_foo.owner_uuid = g_foo.uuid
    assert_equal false, g_foo.save, "should not create an ownership loop"
    assert g_foo.errors.messages[:owner_uuid].join(" ").match(/ownership cycle/)
  end

  test "cannot create a group that is not a 'role' or 'project' or 'filter'" do
    set_user_from_auth :active_trustedclient

    assert_raises(ActiveRecord::RecordInvalid) do
      Group.create!(name: "foo")
    end

    assert_raises(ActiveRecord::RecordInvalid) do
      Group.create!(name: "foo", group_class: "")
    end

    assert_raises(ActiveRecord::RecordInvalid) do
      Group.create!(name: "foo", group_class: "bogus")
    end
  end

  test "cannot change group_class on an already created group" do
    set_user_from_auth :active_trustedclient
    g = Group.create!(name: "foo", group_class: "role")
    assert_raises(ActiveRecord::RecordInvalid) do
      g.update_attributes!(group_class: "project")
    end
  end

  test "role cannot own things" do
    set_user_from_auth :active_trustedclient
    role = Group.create!(name: "foo", group_class: "role")
    assert_raises(ArvadosModel::PermissionDeniedError) do
      Collection.create!(name: "bzzz123", owner_uuid: role.uuid)
    end

    c = Collection.create!(name: "bzzz124")
    assert_raises(ArvadosModel::PermissionDeniedError) do
      c.update_attributes!(owner_uuid: role.uuid)
    end
  end

  test "trash group hides contents" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo", group_class: "project")
    col = Collection.create!(owner_uuid: g_foo.uuid)

    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).any?
    g_foo.update! is_trashed: true
    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).empty?
    assert Collection.readable_by(users(:active), {:include_trash => true}).where(uuid: col.uuid).any?
    g_foo.update! is_trashed: false
    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).any?
  end

  test "trash group" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo", group_class: "project")
    g_bar = Group.create!(name: "bar", owner_uuid: g_foo.uuid, group_class: "project")
    g_baz = Group.create!(name: "baz", owner_uuid: g_bar.uuid, group_class: "project")

    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).any?
    g_foo.update! is_trashed: true
    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).empty?

    assert Group.readable_by(users(:active), {:include_trash => true}).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active), {:include_trash => true}).where(uuid: g_bar.uuid).any?
    assert Group.readable_by(users(:active), {:include_trash => true}).where(uuid: g_baz.uuid).any?
  end


  test "trash subgroup" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo", group_class: "project")
    g_bar = Group.create!(name: "bar", owner_uuid: g_foo.uuid, group_class: "project")
    g_baz = Group.create!(name: "baz", owner_uuid: g_bar.uuid, group_class: "project")

    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).any?
    g_bar.update! is_trashed: true

    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).empty?

    assert Group.readable_by(users(:active), {:include_trash => true}).where(uuid: g_bar.uuid).any?
    assert Group.readable_by(users(:active), {:include_trash => true}).where(uuid: g_baz.uuid).any?
  end

  test "trash subsubgroup" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo", group_class: "project")
    g_bar = Group.create!(name: "bar", owner_uuid: g_foo.uuid, group_class: "project")
    g_baz = Group.create!(name: "baz", owner_uuid: g_bar.uuid, group_class: "project")

    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).any?
    g_baz.update! is_trashed: true
    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).empty?
    assert Group.readable_by(users(:active), {:include_trash => true}).where(uuid: g_baz.uuid).any?
  end


  test "trash group propagates to subgroups" do
    set_user_from_auth :active_trustedclient

    g_foo = groups(:trashed_project)
    g_bar = groups(:trashed_subproject)
    g_baz = groups(:trashed_subproject3)
    col = collections(:collection_in_trashed_subproject)

    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).empty?
    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).empty?

    set_user_from_auth :admin
    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).empty?
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).empty?
    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).empty?

    set_user_from_auth :active_trustedclient
    g_foo.update! is_trashed: false
    assert Group.readable_by(users(:active)).where(uuid: g_foo.uuid).any?
    assert Group.readable_by(users(:active)).where(uuid: g_bar.uuid).any?
    assert Collection.readable_by(users(:active)).where(uuid: col.uuid).any?

    # this one should still be trashed.
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).empty?

    g_baz.update! is_trashed: false
    assert Group.readable_by(users(:active)).where(uuid: g_baz.uuid).any?
  end

  test "trashed does not propagate across permission links" do
    set_user_from_auth :admin

    g_foo = Group.create!(name: "foo", group_class: "role")
    u_bar = User.create!(first_name: "bar")

    assert Group.readable_by(users(:admin)).where(uuid: g_foo.uuid).any?
    assert User.readable_by(users(:admin)).where(uuid:  u_bar.uuid).any?
    g_foo.update! is_trashed: true

    assert Group.readable_by(users(:admin)).where(uuid: g_foo.uuid).empty?
    assert User.readable_by(users(:admin)).where(uuid:  u_bar.uuid).any?

    g_foo.update! is_trashed: false
    ln = Link.create!(tail_uuid: g_foo.uuid,
                      head_uuid: u_bar.uuid,
                      link_class: "permission",
                      name: "can_read")
    g_foo.update! is_trashed: true

    assert Group.readable_by(users(:admin)).where(uuid: g_foo.uuid).empty?
    assert User.readable_by(users(:admin)).where(uuid:  u_bar.uuid).any?
  end

  test "project names must be displayable in a filesystem" do
    set_user_from_auth :active
    ["", "{SOLIDUS}"].each do |subst|
      Rails.configuration.Collections.ForwardSlashNameSubstitution = subst
      proj = Group.create group_class: "project"
      role = Group.create group_class: "role"
      filt = Group.create group_class: "filter", properties: {"filters":[]}
      [[nil, true],
       ["", true],
       [".", false],
       ["..", false],
       ["...", true],
       ["..z..", true],
       ["foo/bar", subst != ""],
       ["../..", subst != ""],
       ["/", subst != ""],
      ].each do |name, valid|
        role.name = name
        assert_equal true, role.valid?
        proj.name = name
        assert_equal valid, proj.valid?, "project: #{name.inspect} should be #{valid ? "valid" : "invalid"}"
        filt.name = name
        assert_equal valid, filt.valid?, "filter: #{name.inspect} should be #{valid ? "valid" : "invalid"}"
      end
    end
  end

  def insert_group uuid, owner_uuid, name, group_class
    q = ActiveRecord::Base.connection.exec_query %{
insert into groups (uuid, owner_uuid, name, group_class, created_at, updated_at)
       values ('#{uuid}', '#{owner_uuid}',
               '#{name}', #{if group_class then "'"+group_class+"'" else 'NULL' end},
               statement_timestamp(), statement_timestamp())
}
    uuid
  end

  test "migration to fix roles and projects" do
    g1 = insert_group Group.generate_uuid, system_user_uuid, 'group with no class', nil
    g2 = insert_group Group.generate_uuid, users(:active).uuid, 'role owned by a user', 'role'

    g3 = insert_group Group.generate_uuid, system_user_uuid, 'role that owns a project', 'role'
    g4 = insert_group Group.generate_uuid, g3, 'the project', 'project'

    g5 = insert_group Group.generate_uuid, users(:active).uuid, 'a project with an outgoing permission link', 'project'

    g6 = insert_group Group.generate_uuid, system_user_uuid, 'name collision', 'role'
    g7 = insert_group Group.generate_uuid, users(:active).uuid, 'name collision', 'role'

    g8 = insert_group Group.generate_uuid, users(:active).uuid, 'trashed with no class', nil
    g8obj = Group.find_by_uuid(g8)
    g8obj.trash_at = db_current_time
    g8obj.delete_at = db_current_time
    act_as_system_user do
      g8obj.save!(validate: false)
    end

    refresh_permissions

    act_as_system_user do
      l1 = Link.create!(link_class: 'permission', name: 'can_manage', tail_uuid: g3, head_uuid: g4)
      q = ActiveRecord::Base.connection.exec_query %{
update links set tail_uuid='#{g5}' where uuid='#{l1.uuid}'
}
    refresh_permissions
    end

    assert_equal nil, Group.find_by_uuid(g1).group_class
    assert_equal nil, Group.find_by_uuid(g8).group_class
    assert_equal users(:active).uuid, Group.find_by_uuid(g2).owner_uuid
    assert_equal g3, Group.find_by_uuid(g4).owner_uuid
    assert !Link.where(tail_uuid: users(:active).uuid, head_uuid: g2, link_class: "permission", name: "can_manage").any?
    assert !Link.where(tail_uuid: g3, head_uuid: g4, link_class: "permission", name: "can_manage").any?
    assert Link.where(link_class: 'permission', name: 'can_manage', tail_uuid: g5, head_uuid: g4).any?

    fix_roles_projects

    assert_equal 'role', Group.find_by_uuid(g1).group_class
    assert_equal 'role', Group.find_by_uuid(g8).group_class
    assert_equal system_user_uuid, Group.find_by_uuid(g2).owner_uuid
    assert_equal system_user_uuid, Group.find_by_uuid(g4).owner_uuid
    assert Link.where(tail_uuid: users(:active).uuid, head_uuid: g2, link_class: "permission", name: "can_manage").any?
    assert Link.where(tail_uuid: g3, head_uuid: g4, link_class: "permission", name: "can_manage").any?
    assert !Link.where(link_class: 'permission', name: 'can_manage', tail_uuid: g5, head_uuid: g4).any?
  end

  test "freeze project" do
    act_as_user users(:active) do
      Rails.configuration.API.UnfreezeProjectRequiresAdmin = false
      parent = Group.create!(group_class: 'project', name: 'freeze-test-parent', owner_uuid: users(:active).uuid)
      proj = Group.create!(group_class: 'project', name: 'freeze-test', owner_uuid: parent.uuid)
      proj_inner = Group.create!(group_class: 'project', name: 'freeze-test-inner', owner_uuid: proj.uuid)
      coll = Collection.create!(name: 'foo', manifest_text: '', owner_uuid: proj_inner.uuid)

      # Cannot set frozen_by_uuid to a different user
      assert_raises do
        proj.update_attributes!(frozen_by_uuid: users(:spectator).uuid)
      end
      proj.reload

      # Cannot set frozen_by_uuid without can_manage permission
      act_as_system_user do
        Link.create!(link_class: 'permission', name: 'can_write', tail_uuid: users(:spectator).uuid, head_uuid: proj.uuid)
      end
      act_as_user users(:spectator) do
        # First confirm we have write permission
        assert Collection.create(name: 'bar', owner_uuid: proj.uuid)
        assert_raises(ArvadosModel::PermissionDeniedError) do
          proj.update_attributes!(frozen_by_uuid: users(:spectator).uuid)
        end
      end
      proj.reload

      # Cannot set frozen_by_uuid without description (if so configured)
      Rails.configuration.API.FreezeProjectRequiresDescription = true
      err = assert_raises do
        proj.update_attributes!(frozen_by_uuid: users(:active).uuid)
      end
      assert_match /can only be set if description is non-empty/, err.inspect
      proj.reload
      err = assert_raises do
        proj.update_attributes!(frozen_by_uuid: users(:active).uuid, description: '')
      end
      assert_match /can only be set if description is non-empty/, err.inspect
      proj.reload

      # Cannot set frozen_by_uuid without properties (if so configured)
      Rails.configuration.API.FreezeProjectRequiresProperties['frobity'] = true
      err = assert_raises do
        proj.update_attributes!(
          frozen_by_uuid: users(:active).uuid,
          description: 'ready to freeze')
      end
      assert_match /can only be set if properties\[frobity\] value is non-empty/, err.inspect
      proj.reload

      # Cannot set frozen_by_uuid while project or its parent is
      # trashed
      [parent, proj].each do |trashed|
        trashed.update_attributes!(trash_at: db_current_time)
        err = assert_raises do
          proj.update_attributes!(
            frozen_by_uuid: users(:active).uuid,
            description: 'ready to freeze',
            properties: {'frobity' => 'bar baz'})
        end
        assert_match /cannot be set on a trashed project/, err.inspect
        proj.reload
        trashed.update_attributes!(trash_at: nil)
      end

      # Can set frozen_by_uuid if all conditions are met
      ok = proj.update_attributes(
        frozen_by_uuid: users(:active).uuid,
        description: 'ready to freeze',
        properties: {'frobity' => 'bar baz'})
      assert ok, proj.errors.messages.inspect

      # Once project is frozen, cannot create new items inside it or
      # its descendants
      [proj, proj_inner].each do |frozen|
        assert_raises do
          collections(:collection_owned_by_active).update_attributes!(owner_uuid: frozen.uuid)
        end
        assert_raises do
          Collection.create!(owner_uuid: frozen.uuid, name: 'inside-frozen-project')
        end
        assert_raises do
          Group.create!(owner_uuid: frozen.uuid, group_class: 'project', name: 'inside-frozen-project')
        end
        cr = ContainerRequest.new(
          command: ["echo", "foo"],
          container_image: links(:docker_image_collection_tag).name,
          cwd: "/tmp",
          environment: {},
          mounts: {"/out" => {"kind" => "tmp", "capacity" => 1000000}},
          output_path: "/out",
          runtime_constraints: {"vcpus" => 1, "ram" => 2},
          name: "foo",
          description: "bar",
          owner_uuid: frozen.uuid,
        )
        assert_raises ArvadosModel::PermissionDeniedError do
          cr.save
        end
        assert_match /frozen/, cr.errors.inspect
        # Check the frozen-parent condition is the only reason save failed.
        cr.owner_uuid = users(:active).uuid
        assert cr.save
        cr.destroy
      end

      # Once project is frozen, cannot change name/contents, move,
      # trash, or delete the project or anything beneath it
      [proj, proj_inner, coll].each do |frozen|
        assert_raises(StandardError, "should reject rename of #{frozen.uuid} with parent #{frozen.owner_uuid}") do
          frozen.update_attributes!(name: 'foo2')
        end
        frozen.reload

        if frozen.is_a?(Collection)
          assert_raises(StandardError, "should reject manifest change of #{frozen.uuid}") do
            frozen.update_attributes!(manifest_text: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n")
          end
        else
          assert_raises(StandardError, "should reject moving a project into #{frozen.uuid}") do
            groups(:private).update_attributes!(owner_uuid: frozen.uuid)
          end
        end
        frozen.reload

        assert_raises(StandardError, "should reject moving #{frozen.uuid} to a different parent project") do
          frozen.update_attributes!(owner_uuid: groups(:private).uuid)
        end
        frozen.reload
        assert_raises(StandardError, "should reject setting trash_at of #{frozen.uuid}") do
          frozen.update_attributes!(trash_at: db_current_time)
        end
        frozen.reload
        assert_raises(StandardError, "should reject setting delete_at of #{frozen.uuid}") do
          frozen.update_attributes!(delete_at: db_current_time)
        end
        frozen.reload
        assert_raises(StandardError, "should reject delete of #{frozen.uuid}") do
          frozen.destroy
        end
        frozen.reload
        if frozen != proj
          assert_equal [], frozen.writable_by
        end
      end

      # User with manage permission can unfreeze, then create items
      # inside it and its children
      assert proj.update_attributes(frozen_by_uuid: nil)
      assert Collection.create!(owner_uuid: proj.uuid, name: 'inside-unfrozen-project')
      assert Collection.create!(owner_uuid: proj_inner.uuid, name: 'inside-inner-unfrozen-project')

      # Re-freeze, and reconfigure so only admins can unfreeze.
      assert proj.update_attributes(frozen_by_uuid: users(:active).uuid)
      Rails.configuration.API.UnfreezeProjectRequiresAdmin = true

      # Owner cannot unfreeze, because not admin.
      err = assert_raises do
        proj.update_attributes!(frozen_by_uuid: nil)
      end
      assert_match /can only be changed by an admin user, once set/, err.inspect
      proj.reload

      # Cannot trash or delete a frozen project's ancestor
      assert_raises(StandardError, "should not be able to set trash_at on parent of frozen project") do
        parent.update_attributes!(trash_at: db_current_time)
      end
      parent.reload
      assert_raises(StandardError, "should not be able to set delete_at on parent of frozen project") do
        parent.update_attributes!(delete_at: db_current_time)
      end
      parent.reload

      act_as_user users(:admin) do
        # Even admin cannot change frozen_by_uuid to someone else's UUID.
        err = assert_raises do
          proj.update_attributes!(frozen_by_uuid: users(:project_viewer).uuid)
        end
        assert_match /can only be set to the current user's UUID/, err.inspect
        proj.reload

        # Admin can unfreeze.
        assert proj.update_attributes(frozen_by_uuid: nil), proj.errors.messages
      end
    end
  end
end
