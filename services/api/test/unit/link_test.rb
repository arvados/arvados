require 'test_helper'

class LinkTest < ActiveSupport::TestCase
  fixtures :all

  setup do
    set_user_from_auth :admin_trustedclient
  end

  test "cannot delete an object referenced by links" do
    ob = Specimen.create
    link = Link.create(tail_uuid: users(:active).uuid,
                       head_uuid: ob.uuid,
                       link_class: 'test',
                       name: 'test')
    assert_equal users(:admin).uuid, link.owner_uuid
    assert_raises(ActiveRecord::DeleteRestrictionError,
                  "should not delete #{ob.uuid} with link #{link.uuid}") do
      ob.destroy
    end
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

  test "link granting permission to nonexistent user is invalid" do
    refute new_active_link_valid?(tail_uuid:
                                  users(:active).uuid.sub(/-\w+$/, "-#{'z' * 15}"))
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
end
