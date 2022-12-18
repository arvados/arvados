# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class LinkTest < ActiveSupport::TestCase
  fixtures :all

  setup do
    set_user_from_auth :admin_trustedclient
  end

  test "cannot delete an object referenced by unwritable links" do
    ob = act_as_user users(:active) do
      Specimen.create
    end
    link = act_as_user users(:admin) do
      Link.create(tail_uuid: users(:active).uuid,
                  head_uuid: ob.uuid,
                  link_class: 'test',
                  name: 'test')
    end
    assert_equal users(:admin).uuid, link.owner_uuid
    assert_raises(ArvadosModel::PermissionDeniedError,
                  "should not delete #{ob.uuid} with link #{link.uuid}") do
      act_as_user users(:active) do
        ob.destroy
      end
    end
    act_as_user users(:admin) do
      ob.destroy
    end
    assert_empty Link.where(uuid: link.uuid)
  end

  def new_active_link_valid?(link_attrs)
    set_user_from_auth :active
    begin
      Link.
        create({link_class: "permission",
                 name: "can_read",
                 head_uuid: groups(:aproject).uuid,
               }.merge(link_attrs)).
        valid?
    rescue ArvadosModel::PermissionDeniedError
      false
    end
  end

  test "non-admin project owner can make it public" do
    assert(new_active_link_valid?(tail_uuid: groups(:anonymous_group).uuid),
           "non-admin project owner can't make their project public")
  end

  test "link granting permission to nonexistent user is invalid" do
    refute new_active_link_valid?(tail_uuid:
                                  users(:active).uuid.sub(/-\w+$/, "-#{'z' * 15}"))
  end

  test "link granting permission to remote user is valid" do
    refute new_active_link_valid?(tail_uuid:
                                  users(:active).uuid.sub(/^\w+-/, "foooo-"))
    Rails.configuration.RemoteClusters = Rails.configuration.RemoteClusters.merge({foooo: ActiveSupport::InheritableOptions.new({Host: "bar.com"})})
    assert new_active_link_valid?(tail_uuid:
                                  users(:active).uuid.sub(/^\w+-/, "foooo-"))
  end

  test "link granting non-project permission to unreadable user is invalid" do
    refute new_active_link_valid?(tail_uuid: users(:admin).uuid,
                                  head_uuid: collections(:bar_file).uuid)
  end

  test "user can't add a Collection to a Project without permission" do
    refute new_active_link_valid?(link_class: "name",
                                  name: "Permission denied test name",
                                  tail_uuid: collections(:bar_file).uuid)
  end

  test "user can't add a User to a Project" do
    # Users *can* give other users permissions to projects.
    # This test helps ensure that that exception is specific to permissions.
    refute new_active_link_valid?(link_class: "name",
                                  name: "Permission denied test name",
                                  tail_uuid: users(:admin).uuid)
  end

  test "link granting project permissions to unreadable user is invalid" do
    refute new_active_link_valid?(tail_uuid: users(:admin).uuid)
  end

  test "permission link can't exist on past collection versions" do
    refute new_active_link_valid?(tail_uuid: groups(:public).uuid,
                                  head_uuid: collections(:w_a_z_file_version_1).uuid)
  end

  def create_overlapping_permissions(names=[], attrs={})
    names.map do |name|
      link = Link.create!({
                            link_class: "tmp",
                            tail_uuid: users(:active).uuid,
                            head_uuid: collections(:baz_file).uuid,
                            name: name,
                          }.merge(attrs).merge({name: name}))
      ActiveRecord::Base.connection.execute "update links set link_class='permission' where uuid='#{link.uuid}'"
      link.uuid
    end
  end

  test "updating permission causes any conflicting links to be deleted" do
    link1, link2 = create_overlapping_permissions(['can_read', 'can_manage'])
    Link.find_by_uuid(link2).update_attributes!(name: 'can_write')
    assert_empty Link.where(uuid: link1)
  end

  test "deleting permission causes any conflicting links to be deleted" do
    rlink, wlink = create_overlapping_permissions(['can_read', 'can_write'])
    Link.find_by_uuid(wlink).destroy
    assert_empty Link.where(uuid: rlink)
  end
end
