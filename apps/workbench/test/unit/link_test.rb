require 'test_helper'

class LinkTest < ActiveSupport::TestCase
  def uuid_for(fixture_name, object_name)
    api_fixture(fixture_name)[object_name]["uuid"]
  end

  test "active user can get permissions for owned project object" do
    use_token :active
    project = Group.find(uuid_for("groups", "aproject"))
    refute_empty(Link.permissions_for(project),
                 "no permissions found for managed project")
  end

  test "active user can get permissions for owned project by UUID" do
    use_token :active
    refute_empty(Link.permissions_for(uuid_for("groups", "aproject")),
                 "no permissions found for managed project")
  end

  test "admin can get permissions for project object" do
    use_token :admin
    project = Group.find(uuid_for("groups", "aproject"))
    refute_empty(Link.permissions_for(project),
                 "no permissions found for managed project")
  end

  test "admin can get permissions for project by UUID" do
    use_token :admin
    refute_empty(Link.permissions_for(uuid_for("groups", "aproject")),
                 "no permissions found for managed project")
  end

  test "project viewer can't get permissions for readable project object" do
    use_token :project_viewer
    project = Group.find(uuid_for("groups", "aproject"))
    assert_raises(ArvadosApiClient::AccessForbiddenException) do
      Link.permissions_for(project)
    end
  end

  test "project viewer can't get permissions for readable project by UUID" do
    use_token :project_viewer
    assert_raises(ArvadosApiClient::AccessForbiddenException) do
      Link.permissions_for(uuid_for("groups", "aproject"))
    end
  end
end
