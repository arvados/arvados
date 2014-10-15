require 'test_helper'

class PipelineInstanceTest < ActiveSupport::TestCase
  test "admin can edit name" do
    use_token :admin
    assert(find_fixture(PipelineInstance, "new_pipeline_in_subproject")
             .attribute_editable?("name"),
           "admin not allowed to edit pipeline instance name")
  end

  test "project owner can edit name" do
    use_token :active
    assert(find_fixture(PipelineInstance, "new_pipeline_in_subproject")
             .attribute_editable?("name"),
           "project owner not allowed to edit pipeline instance name")
  end

  test "project admin can edit name" do
    use_token :subproject_admin
    assert(find_fixture(PipelineInstance, "new_pipeline_in_subproject")
             .attribute_editable?("name"),
           "project admin not allowed to edit pipeline instance name")
  end

  test "project viewer cannot edit name" do
    use_token :project_viewer
    refute(find_fixture(PipelineInstance, "new_pipeline_in_subproject")
             .attribute_editable?("name"),
           "project viewer allowed to edit pipeline instance name")
  end
end
