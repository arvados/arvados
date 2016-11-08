require 'test_helper'
require 'helpers/share_object_helper'

class DisabledApiTest < ActionController::TestCase
  [
    [:admin, false],
    [:admin, true],
  ].each do |user, crunch2|
    test "dashboard recent processes as #{user} with #{if crunch2 then 'crunch2' else 'crunch1' end}" do
      @controller = ProjectsController.new

      if crunch2
        dd = ArvadosApiClient.new_or_current.discovery.deep_dup
        dd[:resources][:pipeline_instances][:methods].delete(:index)
        ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)
      end

      get :index, {}, session_for(user)
      assert_includes @response.body, "zzzzz-xvhdp-cr4runningcntnr"
      if crunch2
        assert_not_includes @response.body, "zzzzz-d1hrv-1yfj6xkidf2muk3"
      else
        assert_includes @response.body, "zzzzz-d1hrv-1yfj6xkidf2muk3"
      end
    end
  end

  [
    [:jobs, JobsController.new],
    [:job_tasks, JobTasksController.new],
    [:pipeline_instances, PipelineInstancesController.new],
    [:pipeline_templates, PipelineTemplatesController.new],
  ].each do |ctrl_name, ctrl|
    test "#{ctrl_name} index page with crunch2" do
      @controller = ctrl

      dd = ArvadosApiClient.new_or_current.discovery.deep_dup
      dd[:resources][ctrl_name][:methods].delete(:index)
      ArvadosApiClient.any_instance.stubs(:discovery).returns(dd)

      get :index, {}, session_for(:active)
      assert_includes @response.body, "index method is not supported for #{ctrl_name.to_s}"
    end
  end
end
