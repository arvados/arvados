require 'test_helper'

class PipelineInstanceTest < ActiveSupport::TestCase
  def attribute_editable_for?(token_name, pi_name, attr_name, ever=nil)
    use_token token_name
    find_fixture(PipelineInstance, pi_name).attribute_editable?(attr_name, ever)
  end

  test "admin can edit name" do
    assert(attribute_editable_for?(:admin, "new_pipeline_in_subproject",
                                   "name"),
           "admin not allowed to edit pipeline instance name")
  end

  test "project owner can edit name" do
    assert(attribute_editable_for?(:active, "new_pipeline_in_subproject",
                                   "name"),
           "project owner not allowed to edit pipeline instance name")
  end

  test "project admin can edit name" do
    assert(attribute_editable_for?(:subproject_admin,
                                   "new_pipeline_in_subproject", "name"),
           "project admin not allowed to edit pipeline instance name")
  end

  test "project viewer cannot edit name" do
    refute(attribute_editable_for?(:project_viewer,
                                   "new_pipeline_in_subproject", "name"),
           "project viewer allowed to edit pipeline instance name")
  end

  test "name editable on completed pipeline" do
    assert(attribute_editable_for?(:active, "has_component_with_completed_jobs",
                                   "name"),
           "name not editable on complete pipeline")
  end

  test "components editable on new pipeline" do
    assert(attribute_editable_for?(:active, "new_pipeline", "components"),
           "components not editable on new pipeline")
  end

  test "components not editable on completed pipeline" do
    refute(attribute_editable_for?(:active, "has_component_with_completed_jobs",
                                   "components"),
           "components not editable on new pipeline")
  end
end
