require 'test_helper'

class SerializedEncodingTest < ActionDispatch::IntegrationTest
  fixtures :all

  test "store json-encoded link with properties hash" do
    post "/arvados/v1/links", {
      :link => {
        :link_class => 'test',
        :name => 'test',
        :properties => {:foo => :bar}
      }.to_json,
      :format => :json
    }, auth(:active)
    assert_response :success
  end

  test "store json-encoded pipeline instance with components_summary hash" do
    post "/arvados/v1/pipeline_instances", {
      :pipeline_instance => {
        :components_summary => {:todo => 0}
      }.to_json,
      :format => :json
    }, auth(:active)
    assert_response :success
  end
end
