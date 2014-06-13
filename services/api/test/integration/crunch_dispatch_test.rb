require 'pp'
require 'test_helper'
load 'test/functional/arvados/v1/git_setup.rb'

class CrunchDispatchTest < ActionDispatch::IntegrationTest
  include CurrentApiClient
  include GitSetup

  fixtures :all

  test "job runs" do
    authorize_with :admin
    set_user_from_auth :admin
    post "/arvados/v1/jobs", {
      format: "json",
      job: {
        script: "log",
        repository: "bar",
        script_version: "143fec09e988160673c63457fa12a0f70b5b8a26",
        script_parameters: {}
      }
    }, {HTTP_AUTHORIZATION: "OAuth2 #{current_api_client_authorization}"}
    p "response: #{@response.body}"
    assert_response :success
    resp = JSON.parse(@response.body)
  end
end
