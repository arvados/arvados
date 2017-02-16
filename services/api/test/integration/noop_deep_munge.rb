require 'test_helper'

class NoopDeepMunge < ActionDispatch::IntegrationTest

  test "that empty list round trips properly" do
    post "/arvados/v1/container_requests",
         {
           :container_request => {
             :name => "workflow",
             :state => "Uncommitted",
             :command => ["echo"],
             :container_image => "arvados/jobs",
             :output_path => "/",
             :mounts => {
               :foo => {
                 :kind => "json",
                 :content => {
                   :a => [],
                   :b => {}
                 }
               }
             }
           }
         }.to_json, {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}",
                    'CONTENT_TYPE' => 'application/json'}
    assert_response :success
    assert_equal "arvados#containerRequest", json_response['kind']
    content = {
      "a" => [],
      "b" => {}
    }
    assert_equal content, json_response['mounts']['foo']['content']

  end
end
