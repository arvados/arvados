require 'test_helper'

class WorkUnitTest < ActiveSupport::TestCase
  setup do
    Rails.configuration.anonymous_user_token = api_fixture('api_client_authorizations')['anonymous']['api_token']
  end

  [
    [Job, 'running_job_with_components', "jwu", 2, "Running", nil, 0.5],
    [PipelineInstance, 'pipeline_in_running_state', nil, 1, "Running", nil, 0.0],
    [PipelineInstance, 'has_component_with_completed_jobs', nil, 3, "Complete", true, 1.0],
    [PipelineInstance, 'pipeline_with_tagged_collection_input', "pwu", 1, "Ready", nil, 0.0],
    [Container, 'requester', 'cwu', 1, "Complete", true, 1.0],
    [ContainerRequest, 'cr_for_requester', 'cwu', 1, "Complete", true, 1.0],
  ].each do |type, fixture, label, num_children, state, success, progress|
    test "children of #{fixture}" do
      use_token 'active'
      obj = find_fixture(type, fixture)
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

  [
    [Job, 'running_job_with_components', 1, 1, nil],
    [Job, 'queued', nil, nil, 1],
    [PipelineInstance, 'pipeline_in_running_state', 1, 1, nil],
    [PipelineInstance, 'has_component_with_completed_jobs', 60, 60, nil],
  ].each do |type, fixture, walltime, cputime, queuedtime|
    test "times for #{fixture}" do
      use_token 'active'
      obj = find_fixture(type, fixture)
      wu = obj.work_unit

      if walltime
        assert_equal true, (wu.walltime >= walltime)
      else
        assert_equal walltime, wu.walltime
      end

      if cputime
        assert_equal true, (wu.cputime >= cputime)
      else
        assert_equal cputime, wu.cputime
      end

      if queuedtime
        assert_equal true, (wu.queuedtime >= queuedtime)
      else
        assert_equal queuedtime, wu.queuedtime
      end
    end
  end

  [
    [Job, 'active', 'running_job_with_components', true],
    [Job, 'active', 'queued', false],
    [Job, nil, 'completed_job_in_publicly_accessible_project', true],
    [Job, 'active', 'completed_job_in_publicly_accessible_project', true],
    [PipelineInstance, 'active', 'pipeline_in_running_state', true],  # no log, but while running the log link points to pi Log tab
    [PipelineInstance, nil, 'pipeline_in_publicly_accessible_project_but_other_objects_elsewhere', false],
    [PipelineInstance, 'active', 'pipeline_in_publicly_accessible_project_but_other_objects_elsewhere', false], #no log for completed pi
    [Job, nil, 'job_in_publicly_accessible_project_but_other_objects_elsewhere', false, "Log unavailable"],
  ].each do |type, token, fixture, has_log, log_link|
    test "link_to_log for #{fixture} for #{token}" do
      use_token token if token
      obj = find_fixture(type, fixture)
      wu = obj.work_unit

      link = "#{wu.uri}#Log" if has_log
      link_to_log = wu.link_to_log

      if has_log
        assert_includes link_to_log, link
      else
        assert_equal log_link, link_to_log
      end
    end
  end
end
