require 'test_helper'

class PermissionTest < ActiveSupport::TestCase
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
end
