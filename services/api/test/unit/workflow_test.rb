# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class WorkflowTest < ActiveSupport::TestCase
  test "create workflow with no definition yaml" do
    set_user_from_auth :active

    wf = {
      name: "test name",
    }

    w = Workflow.create!(wf)
    assert_not_nil w.uuid
  end

  test "create workflow with valid definition yaml" do
    set_user_from_auth :active

    wf = {
      name: "test name",
      definition: "k1:\n v1: x\n v2: y"
    }

    w = Workflow.create!(wf)
    assert_not_nil w.uuid
  end

  test "create workflow with simple string as definition" do
    set_user_from_auth :active

    wf = {
      name: "test name",
      definition: "this is valid yaml"
    }

    w = Workflow.create!(wf)
    assert_not_nil w.uuid
  end

  test "create workflow with invalid definition yaml" do
    set_user_from_auth :active

    wf = {
      name: "test name",
      definition: "k1:\n v1: x\n  v2: y"
    }

    assert_raises(ActiveRecord::RecordInvalid) do
      Workflow.create! wf
    end
  end

  test "update workflow with invalid definition yaml" do
    set_user_from_auth :active

    w = Workflow.find_by_uuid(workflows(:workflow_with_definition_yml).uuid)
    definition = "k1:\n v1: x\n  v2: y"

    assert_raises(ActiveRecord::RecordInvalid) do
      w.update!(definition: definition)
    end
  end

  test "update workflow and verify name and description" do
    set_user_from_auth :active

    # Workflow name and desc should be set with values from definition yaml
    # when it does not already have custom values for these fields
    w = Workflow.find_by_uuid(workflows(:workflow_with_no_name_and_desc).uuid)
    definition = "name: test name 1\ndescription: test desc 1\nother: some more"
    w.update!(definition: definition)
    w.reload
    assert_equal "test name 1", w.name
    assert_equal "test desc 1", w.description

    # Workflow name and desc should be set with values from definition yaml
    # when it does not already have custom values for these fields
    definition = "name: test name 2\ndescription: test desc 2\nother: some more"
    w.update!(definition: definition)
    w.reload
    assert_equal "test name 2", w.name
    assert_equal "test desc 2", w.description

    # Workflow name and desc should be set with values from definition yaml
    # even if it means emptying them out
    definition = "more: etc"
    w.update!(definition: definition)
    w.reload
    assert_nil w.name
    assert_nil w.description

    # Workflow name and desc set using definition yaml should be cleared
    # if definition yaml is cleared
    definition = "name: test name 2\ndescription: test desc 2\nother: some more"
    w.update!(definition: definition)
    w.reload
    definition = nil
    w.update!(definition: definition)
    w.reload
    assert_nil w.name
    assert_nil w.description

    # Workflow name and desc should be set to provided custom values
    definition = "name: test name 3\ndescription: test desc 3\nother: some more"
    w.update!(name: "remains", description: "remains", definition: definition)
    w.reload
    assert_equal "remains", w.name
    assert_equal "remains", w.description

    # Workflow name and desc should retain provided custom values
    # and should not be overwritten by values from yaml
    definition = "name: test name 4\ndescription: test desc 4\nother: some more"
    w.update!(definition: definition)
    w.reload
    assert_equal "remains", w.name
    assert_equal "remains", w.description

    # Workflow name and desc should retain provided custom values
    # and not be affected by the clearing of the definition yaml
    definition = nil
    w.update!(definition: definition)
    w.reload
    assert_equal "remains", w.name
    assert_equal "remains", w.description
  end
end
