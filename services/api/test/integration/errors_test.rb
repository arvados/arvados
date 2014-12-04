require 'test_helper'

class ErrorsTest < ActionDispatch::IntegrationTest
  fixtures :api_client_authorizations

  %w(/arvados/v1/shoes /arvados/shoes /shoes /nodes /users).each do |path|
    test "non-existent route #{path}" do
      get path, {:format => :json}, auth(:active)
      assert_nil assigns(:objects)
      assert_nil assigns(:object)
      assert_not_nil json_response['errors']
      assert_response 404
    end
  end

  n=0
  Rails.application.routes.routes.each do |route|
    test "route #{n += 1} '#{route.path.spec.to_s}' is not an accident" do
      # Generally, new routes should appear under /arvados/v1/. If
      # they appear elsewhere, that might have been caused by default
      # rails generator behavior that we don't want.
      assert_match(/^\/(|\*a|arvados\/v1\/.*|auth\/.*|login|logout|database\/reset|discovery\/.*|static\/.*|themes\/.*)(\(\.:format\))?$/,
                   route.path.spec.to_s,
                   "Unexpected new route: #{route.path.spec}")
    end
  end
end
