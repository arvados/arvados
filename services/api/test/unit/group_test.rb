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

  test "move projects to trash in SweepTrashedObjects" do
    p = groups(:trashed_on_next_sweep)
    assert_empty Group.where('uuid=? and is_trashed=true', p.uuid)
    SweepTrashedObjects.sweep_now
    assert_not_empty Group.where('uuid=? and is_trashed=true', p.uuid)
  end

  test "delete projects and their contents in SweepTrashedObjects" do
    g_foo = groups(:trashed_project)
    g_bar = groups(:trashed_subproject)
    g_baz = groups(:trashed_subproject3)
    col = collections(:collection_in_trashed_subproject)
    job = jobs(:job_in_trashed_project)
    cr = container_requests(:cr_in_trashed_project)
    # Save how many objects were before the sweep
    user_nr_was = User.all.length
    coll_nr_was = Collection.all.length
    group_nr_was = Group.where('group_class<>?', 'project').length
    project_nr_was = Group.where(group_class: 'project').length
    cr_nr_was = ContainerRequest.all.length
    job_nr_was = Job.all.length
    assert_not_empty Group.where(uuid: g_foo.uuid)
    assert_not_empty Group.where(uuid: g_bar.uuid)
    assert_not_empty Group.where(uuid: g_baz.uuid)
    assert_not_empty Collection.where(uuid: col.uuid)
    assert_not_empty Job.where(uuid: job.uuid)
    assert_not_empty ContainerRequest.where(uuid: cr.uuid)
    SweepTrashedObjects.sweep_now
    assert_empty Group.where(uuid: g_foo.uuid)
    assert_empty Group.where(uuid: g_bar.uuid)
    assert_empty Group.where(uuid: g_baz.uuid)
    assert_empty Collection.where(uuid: col.uuid)
    assert_empty Job.where(uuid: job.uuid)
    assert_empty ContainerRequest.where(uuid: cr.uuid)
    # No unwanted deletions should have happened
    assert_equal user_nr_was, User.all.length
    assert_equal coll_nr_was-2,        # collection_in_trashed_subproject
                 Collection.all.length # & deleted_on_next_sweep collections
    assert_equal group_nr_was, Group.where('group_class<>?', 'project').length
    assert_equal project_nr_was-3, Group.where(group_class: 'project').length
    assert_equal cr_nr_was-1, ContainerRequest.all.length
    assert_equal job_nr_was-1, Job.all.length
  end

  test "project names must be displayable in a filesystem" do
    set_user_from_auth :active
    ["", "{SOLIDUS}"].each do |subst|
      Rails.configuration.Collections.ForwardSlashNameSubstitution = subst
      proj = Group.create group_class: "project"
      role = Group.create group_class: "role"
      filt = Group.create group_class: "filter"
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
        assert_equal valid, proj.valid?, "#{name.inspect} should be #{valid ? "valid" : "invalid"}"
        filt.name = name
        assert_equal valid, filt.valid?, "#{name.inspect} should be #{valid ? "valid" : "invalid"}"
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
end
