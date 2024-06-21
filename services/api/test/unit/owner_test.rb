# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

# Test referential integrity: ensure we cannot leave any object
# without owners by deleting a user or group.
#
# "o" is an owner.
# "i" is an item.

class OwnerTest < ActiveSupport::TestCase
  fixtures :users, :groups

  setup do
    set_user_from_auth :admin_trustedclient
  end

  User.all
  Group.all
  [User, Group].each do |o_class|
    test "create object with legit #{o_class} owner" do
      if o_class == Group
        o = o_class.create! group_class: "project"
      else
        o = o_class.create!
      end
      i = Collection.create(owner_uuid: o.uuid)
      assert i.valid?, "new item should pass validation"
      assert i.uuid, "new item should have an ID"
      assert Collection.where(uuid: i.uuid).any?, "new item should really be in DB"
    end

    test "create object with non-existent #{o_class} owner" do
      assert_raises(ActiveRecord::RecordInvalid,
                    "create should fail with random owner_uuid") do
        Collection.create!(owner_uuid: o_class.generate_uuid)
      end

      i = Collection.create(owner_uuid: o_class.generate_uuid)
      assert !i.valid?, "object with random owner_uuid should not be valid?"

      i = Collection.new(owner_uuid: o_class.generate_uuid)
      assert !i.valid?, "new item should not pass validation"
      assert !i.uuid, "new item should not have an ID"
    end

    [User, Group].each do |new_o_class|
      test "change owner from legit #{o_class} to legit #{new_o_class} owner" do
        o = if o_class == Group
              o_class.create! group_class: "project"
            else
              o_class.create!
            end
        i = Collection.create!(owner_uuid: o.uuid)

        new_o = if new_o_class == Group
              new_o_class.create! group_class: "project"
            else
              new_o_class.create!
            end

        assert(Collection.where(uuid: i.uuid).any?,
               "new item should really be in DB")
        assert(i.update(owner_uuid: new_o.uuid),
               "should change owner_uuid from #{o.uuid} to #{new_o.uuid}")
      end
    end

    test "delete #{o_class} that owns nothing" do
      if o_class == Group
        o = o_class.create! group_class: "project"
      else
        o = o_class.create!
      end
      assert(o_class.where(uuid: o.uuid).any?,
             "new #{o_class} should really be in DB")
      assert(o.destroy, "should delete #{o_class} that owns nothing")
      assert_equal(false, o_class.where(uuid: o.uuid).any?,
                   "#{o.uuid} should not be in DB after deleting")
    end

    test "change uuid of #{o_class} that owns nothing" do
      # (we're relying on our admin credentials here)
      if o_class == Group
        o = o_class.create! group_class: "project"
      else
        o = o_class.create!
      end
      assert(o_class.where(uuid: o.uuid).any?,
             "new #{o_class} should really be in DB")
      old_uuid = o.uuid
      new_uuid = o.uuid.sub(/..........$/, rand(2**256).to_s(36)[0..9])
      assert(o.update(uuid: new_uuid),
              "should change #{o_class} uuid from #{old_uuid} to #{new_uuid}")
      assert_equal(false, o_class.where(uuid: old_uuid).any?,
                   "#{old_uuid} should disappear when renamed to #{new_uuid}")
    end
  end

  ['users(:active)', 'groups(:aproject)'].each do |ofixt|
    test "delete #{ofixt} that owns other objects" do
      o = eval ofixt
      assert_equal(true, Collection.where(owner_uuid: o.uuid).any?,
                   "need something to be owned by #{o.uuid} for this test")

      skip_check_permissions_against_full_refresh do
        assert_raises(ActiveRecord::DeleteRestrictionError,
                      "should not delete #{ofixt} that owns objects") do
          o.destroy
        end
      end
    end

    test "change uuid of #{ofixt} that owns other objects" do
      o = eval ofixt
      assert_equal(true, Collection.where(owner_uuid: o.uuid).any?,
                   "need something to be owned by #{o.uuid} for this test")
      new_uuid = o.uuid.sub(/..........$/, rand(2**256).to_s(36)[0..9])
      assert(!o.update(uuid: new_uuid),
             "should not change uuid of #{ofixt} that owns objects")
    end
  end

  test "delete User that owns self" do
    o = User.create!
    assert User.where(uuid: o.uuid).any?, "new User should really be in DB"
    assert_equal(true, o.update(owner_uuid: o.uuid),
                 "setting owner to self should work")

    skip_check_permissions_against_full_refresh do
      assert(o.destroy, "should delete User that owns self")
    end

    assert_equal(false, User.where(uuid: o.uuid).any?,
                 "#{o.uuid} should not be in DB after deleting")
    check_permissions_against_full_refresh
  end

end
