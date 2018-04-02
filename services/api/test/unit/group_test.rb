# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class GroupTest < ActiveSupport::TestCase

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

    g_foo = Group.create!(name: "foo")
    g_bar = Group.create!(name: "bar")

    g_foo.owner_uuid = g_bar.uuid
    assert g_foo.save, lambda { g_foo.errors.messages }
    g_bar.owner_uuid = g_foo.uuid
    assert g_bar.valid?, "ownership cycle should not prevent validation"
    assert_equal false, g_bar.save, "should not create an ownership loop"
    assert g_bar.errors.messages[:owner_uuid].join(" ").match(/ownership cycle/)
  end

  test "cannot create a single-object ownership cycle" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo")
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

  test "trash group hides contents" do
    set_user_from_auth :active_trustedclient

    g_foo = Group.create!(name: "foo")
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

    g_foo = Group.create!(name: "foo")
    g_bar = Group.create!(name: "bar", owner_uuid: g_foo.uuid)
    g_baz = Group.create!(name: "baz", owner_uuid: g_bar.uuid)

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

    g_foo = Group.create!(name: "foo")
    g_bar = Group.create!(name: "bar", owner_uuid: g_foo.uuid)
    g_baz = Group.create!(name: "baz", owner_uuid: g_bar.uuid)

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

    g_foo = Group.create!(name: "foo")
    g_bar = Group.create!(name: "bar", owner_uuid: g_foo.uuid)
    g_baz = Group.create!(name: "baz", owner_uuid: g_bar.uuid)

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

    g_foo = Group.create!(name: "foo")
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
end
