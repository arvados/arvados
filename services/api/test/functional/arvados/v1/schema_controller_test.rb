require 'test_helper'

class Arvados::V1::SchemaControllerTest < ActionController::TestCase

  test "should get fresh discovery document" do
    MAX_SCHEMA_AGE = 60
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    assert_equal 'discovery#restDescription', discovery_doc['kind']
    assert_equal(true,
                 Time.now - MAX_SCHEMA_AGE.seconds < discovery_doc['generatedAt'],
                 "discovery document was generated >#{MAX_SCHEMA_AGE}s ago")
  end

  test "discovery document has defaultTrashLifetime" do
    get :index
    assert_response :success
    discovery_doc = JSON.parse(@response.body)
    assert_includes discovery_doc, 'defaultTrashLifetime'
    assert_equal discovery_doc['defaultTrashLifetime'], Rails.application.config.default_trash_lifetime
  end
end
