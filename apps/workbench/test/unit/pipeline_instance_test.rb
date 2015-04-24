require 'test_helper'

class PipelineInstanceTest < ActiveSupport::TestCase
  def find_pi_with(token_name, pi_name)
    use_token token_name
    find_fixture(PipelineInstance, pi_name)
  end

  def attribute_editable_for?(token_name, pi_name, attr_name, ever=nil)
    find_pi_with(token_name, pi_name).attribute_editable?(attr_name, ever)
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

  test "job_logs for partially complete pipeline" do
    log_uuid = api_fixture("collections", "real_log_collection", "uuid")
    pi = find_pi_with(:active, "running_pipeline_with_complete_job")
    assert_equal({previous: log_uuid, running: nil}, pi.job_log_ids)
  end

  test "job_logs for complete pipeline" do
    log_uuid = api_fixture("collections", "real_log_collection", "uuid")
    pi = find_pi_with(:active, "complete_pipeline_with_two_jobs")
    assert_equal({ancient: log_uuid, previous: log_uuid}, pi.job_log_ids)
  end

  test "job_logs for malformed pipeline" do
    pi = find_pi_with(:active, "components_is_jobspec")
    assert_empty(pi.job_log_ids.select { |_, log| not log.nil? })
  end

  def check_stderr_logs(token_name, pi_name, log_name)
    pi = find_pi_with(token_name, pi_name)
    actual_logs = pi.stderr_log_lines
    expected_text = api_fixture("logs", log_name, "properties", "text")
    expected_text.each_line do |log_line|
      assert_includes(actual_logs, log_line.chomp)
    end
  end

  test "stderr_logs for running pipeline" do
    check_stderr_logs(:active,
                      "pipeline_in_publicly_accessible_project",
                      "log_line_for_pipeline_in_publicly_accessible_project")
  end

  test "stderr_logs for job in complete pipeline" do
    check_stderr_logs(:active,
                      "failed_pipeline_with_two_jobs",
                      "crunchstat_for_previous_job")
  end

  test "has_readable_logs? for unrun pipeline" do
    pi = find_pi_with(:active, "new_pipeline")
    refute(pi.has_readable_logs?)
  end

  test "has_readable_logs? for running pipeline" do
    pi = find_pi_with(:active, "running_pipeline_with_complete_job")
    assert(pi.has_readable_logs?)
  end

  test "has_readable_logs? for complete pipeline" do
    pi = find_pi_with(:active, "pipeline_in_publicly_accessible_project_but_other_objects_elsewhere")
    assert(pi.has_readable_logs?)
  end

  test "has_readable_logs? for complete pipeline when jobs unreadable" do
    pi = find_pi_with(:anonymous, "pipeline_in_publicly_accessible_project_but_other_objects_elsewhere")
    refute(pi.has_readable_logs?)
  end
end
