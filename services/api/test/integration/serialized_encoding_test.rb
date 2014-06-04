require 'test_helper'
load 'test/functional/arvados/v1/git_setup.rb'

class SerializedEncodingTest < ActionDispatch::IntegrationTest
  include GitSetup

  fixtures :all

  {
    api_client_authorization: {scopes: []},

    human: {properties: {eye_color: 'gray'}},

    job: {
      repository: 'foo',
      runtime_constraints: {docker_image: 'arvados/jobs'},
      script: 'hash',
      script_version: 'master',
      script_parameters: {pattern: 'foobar'},
      tasks_summary: {todo: 0},
    },

    job_task: {parameters: {pattern: 'foo'}},

    link: {link_class: 'test', name: 'test', properties: {foo: :bar}},

    node: {info: {uptime: 1234}},

    pipeline_instance: {
      components: {"job1" => {parameters: {pattern: "xyzzy"}}},
      components_summary: {todo: 0},
      properties: {test: true},
    },

    pipeline_template: {
      components: {"job1" => {parameters: {pattern: "xyzzy"}}},
    },

    specimen: {properties: {eye_color: 'meringue'}},

    trait: {properties: {eye_color: 'brown'}},

    user: {prefs: {cookies: 'thin mint'}},
  }.each_pair do |resource, postdata|
    test "create json-encoded #{resource.to_s}" do
      post("/arvados/v1/#{resource.to_s.pluralize}",
           {resource => postdata.to_json}, auth(:admin_trustedclient))
      assert_response :success
    end
  end
end
