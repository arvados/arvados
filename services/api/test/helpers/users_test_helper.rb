# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module UsersTestHelper
  include CurrentApiClient

  def verify_link(response_items, link_object_name, expect_link, link_class,
        link_name, head_uuid, tail_uuid, head_kind, fetch_object, class_name)
    link = find_obj_in_resp response_items, 'arvados#link', link_object_name

    if !expect_link
      assert_nil link, "Expected no link for #{link_object_name}"
      return
    end

    assert_not_nil link, "Expected link for #{link_object_name}"

    if fetch_object
      object = Object.const_get(class_name).where(name: head_uuid)
      assert [] != object, "expected #{class_name} with name #{head_uuid}"
      head_uuid = object.first[:uuid]
    end
    assert_equal link_class, link['link_class'],
        "did not find expected link_class for #{link_object_name}"

    assert_equal link_name, link['name'],
        "did not find expected link_name for #{link_object_name}"

    assert_equal tail_uuid, link['tail_uuid'],
        "did not find expected tail_uuid for #{link_object_name}"

    assert_equal head_kind, link['head_kind'],
        "did not find expected head_kind for #{link_object_name}"

    assert_equal head_uuid, link['head_uuid'],
        "did not find expected head_uuid for #{link_object_name}"
  end

  def verify_system_group_permission_link_for user_uuid
    assert_equal 1, Link.where(link_class: 'permission',
                               name: 'can_manage',
                               tail_uuid: system_group_uuid,
                               head_uuid: user_uuid).count
  end

  def verify_link_existence uuid, email, expect_oid_login_perms,
      expect_repo_perms, expect_vm_perms, expect_group_perms, expect_signatures
    # verify that all links are deleted for the user
    oid_login_perms = Link.where(tail_uuid: email,
                                 link_class: 'permission',
                                 name: 'can_login').where("head_uuid like ?", User.uuid_like_pattern)

    # these don't get added any more!  they shouldn't appear ever.
    assert !oid_login_perms.any?, "expected all oid_login_perms deleted"

    # these don't get added any more!  they shouldn't appear ever.
    repo_perms = Link.where(tail_uuid: uuid,
                            link_class: 'permission').where("head_uuid like ?", '_____-s0uqq-_______________')
    assert !repo_perms.any?, "expected all repo_perms deleted"

    vm_login_perms = Link.
      where(tail_uuid: uuid,
            link_class: 'permission',
            name: 'can_login').
      where("head_uuid like ?",
            VirtualMachine.uuid_like_pattern).
      where('uuid <> ?',
            links(:auto_setup_vm_login_username_can_login_to_test_vm).uuid)
    if expect_vm_perms
      assert vm_login_perms.any?, "expected vm_login_perms"
    else
      assert !vm_login_perms.any?, "expected all vm_login_perms deleted"
    end

    group_write_perms = Link.where(tail_uuid: uuid,
                                  head_uuid: all_users_group_uuid,
                                  link_class: 'permission',
                                  name: 'can_write')
    if expect_group_perms
      assert group_write_perms.any?, "expected all users group write perms"
    else
      assert !group_write_perms.any?, "expected all users group write perms deleted"
    end

    signed_uuids = Link.where(link_class: 'signature',
                              tail_uuid: uuid)

    if expect_signatures
      assert signed_uuids.any?, "expected signatures"
    else
      assert !signed_uuids.any?, "expected all signatures deleted"
    end

  end

end
