require 'test_helper'

class JobTest < ActiveSupport::TestCase
  test "admin can edit description" do
    use_token :admin
    assert(find_fixture(Job, "job_in_subproject")
             .attribute_editable?("description"),
           "admin not allowed to edit job description")
  end

  test "project owner can edit description" do
    use_token :active
    assert(find_fixture(Job, "job_in_subproject")
             .attribute_editable?("description"),
           "project owner not allowed to edit job description")
  end

  test "project admin can edit description" do
    use_token :subproject_admin
    assert(find_fixture(Job, "job_in_subproject")
             .attribute_editable?("description"),
           "project admin not allowed to edit job description")
  end

  test "project viewer cannot edit description" do
    use_token :project_viewer
    refute(find_fixture(Job, "job_in_subproject")
             .attribute_editable?("description"),
           "project viewer allowed to edit job description")
  end
end
