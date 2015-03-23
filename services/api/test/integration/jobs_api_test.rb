require 'test_helper'

class JobsApiTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "cancel job" do
    post "/arvados/v1/jobs/#{jobs(:running).uuid}/cancel", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:active).api_token}"}
    assert_response :success
    assert_equal "arvados#job", json_response['kind']
    assert_not_nil json_response['cancelled_at']
  end

  test "cancel someone else's visible job" do
    post "/arvados/v1/jobs/#{jobs(:runningbarbaz).uuid}/cancel", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    assert_response 403
  end

  test "cancel someone else's invisible job" do
    post "/arvados/v1/jobs/#{jobs(:running).uuid}/cancel", {:format => :json}, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    assert_response 404
  end

  test "task qsequence values automatically increase monotonically" do
    post_args = ["/arvados/v1/job_tasks",
                 {job_task: {
                     job_uuid: jobs(:running).uuid,
                     sequence: 1,
                   }},
                 auth(:active)]
    last_qsequence = -1
    (1..3).each do |task_num|
      @response = nil
      post(*post_args)
      assert_response :success
      qsequence = json_response["qsequence"]
      assert_not_nil(qsequence, "task not assigned qsequence")
      assert_operator(qsequence, :>, last_qsequence,
                      "qsequence did not increase between tasks")
      last_qsequence = qsequence
    end
  end
end
