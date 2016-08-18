require 'test_helper'

class WorkflowTest < ActiveSupport::TestCase
  test "create workflow with no workflow yaml" do
    set_user_from_auth :active

    wf = {
      name: "test name",
    }

    w = Workflow.create!(wf)
    assert_not_nil w.uuid
  end

  test "create workflow with valid workflow yaml" do
    set_user_from_auth :active

    wf = {
      name: "test name",
      workflow: "k1:\n v1: x\n v2: y"
    }

    w = Workflow.create!(wf)
    assert_not_nil w.uuid
  end

  test "create workflow with simple string as workflow" do
    set_user_from_auth :active

    wf = {
      name: "test name",
      workflow: "this is valid yaml"
    }

    w = Workflow.create!(wf)
    assert_not_nil w.uuid
  end

  test "create workflow with invalid workflow yaml" do
    set_user_from_auth :active

    wf = {
      name: "test name",
      workflow: "k1:\n v1: x\n  v2: y"
    }

    assert_raises(ActiveRecord::RecordInvalid) do
      Workflow.create! wf
    end
  end

  test "update workflow with invalid workflow yaml" do
    set_user_from_auth :active

    w = Workflow.find_by_uuid(workflows(:workflow_with_workflow_yml).uuid)
    wf = "k1:\n v1: x\n  v2: y"

    assert_raises(ActiveRecord::RecordInvalid) do
      w.update_attributes!(workflow: wf)
    end
  end

  test "update workflow and verify name and description" do
    set_user_from_auth :active

    # Workflow name and desc should be set with values from workflow yaml
    # when it does not already have custom values for these fields
    w = Workflow.find_by_uuid(workflows(:workflow_with_no_name_and_desc).uuid)
    wf = "name: test name 1\ndescription: test desc 1\nother: some more"
    w.update_attributes!(workflow: wf)
    w.reload
    assert_equal "test name 1", w.name
    assert_equal "test desc 1", w.description

    # Workflow name and desc should be set with values from workflow yaml
    # when it does not already have custom values for these fields
    wf = "name: test name 2\ndescription: test desc 2\nother: some more"
    w.update_attributes!(workflow: wf)
    w.reload
    assert_equal "test name 2", w.name
    assert_equal "test desc 2", w.description

    # Workflow name and desc should be set with values from workflow yaml
    # even if it means emptying them out
    wf = "more: etc"
    w.update_attributes!(workflow: wf)
    w.reload
    assert_equal nil, w.name
    assert_equal nil, w.description

    # Workflow name and desc set using workflow yaml should be cleared
    # if workflow yaml is cleared
    wf = "name: test name 2\ndescription: test desc 2\nother: some more"
    w.update_attributes!(workflow: wf)
    w.reload
    wf = nil
    w.update_attributes!(workflow: wf)
    w.reload
    assert_equal nil, w.name
    assert_equal nil, w.description

    # Workflow name and desc should be set to provided custom values
    wf = "name: test name 3\ndescription: test desc 3\nother: some more"
    w.update_attributes!(name: "remains", description: "remains", workflow: wf)
    w.reload
    assert_equal "remains", w.name
    assert_equal "remains", w.description

    # Workflow name and desc should retain provided custom values
    # and should not be overwritten by values from yaml
    wf = "name: test name 4\ndescription: test desc 4\nother: some more"
    w.update_attributes!(workflow: wf)
    w.reload
    assert_equal "remains", w.name
    assert_equal "remains", w.description

    # Workflow name and desc should retain provided custom values
    # and not be affected by the clearing of the workflow yaml
    wf = nil
    w.update_attributes!(workflow: wf)
    w.reload
    assert_equal "remains", w.name
    assert_equal "remains", w.description
  end
end
