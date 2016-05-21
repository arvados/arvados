require 'test_helper'

class WorkUnitTest < ActiveSupport::TestCase
  [
    [Job, 'running_job_with_components', "jwu", 2, "Running", nil, 0.2],
    [PipelineInstance, 'pipeline_in_running_state', nil, 1, "Running", nil, 0.0],
    [PipelineInstance, 'has_component_with_completed_jobs', nil, 3, "Complete", true, 1.0],
    [PipelineInstance, 'pipeline_with_tagged_collection_input', "pwu", 1, "Ready", nil, 0.0],
  ].each do |type, name, label, num_children, state, success, progress|
    test "children of #{name}" do
      use_token 'admin'
      obj = find_fixture(type, name)
      wu = obj.work_unit(label)

      if label != nil
        assert_equal(label, wu.label)
      else
        assert_equal(obj.name, wu.label)
      end
      assert_equal(obj['uuid'], wu.uuid)
      assert_equal(state, wu.state_label)
      assert_equal(success, wu.success?)
      assert_equal(progress, wu.progress)

      assert_equal(num_children, wu.children.size)
      wu.children.each do |child|
        assert_equal(true, child.respond_to?(:script))
      end
    end
  end
end
