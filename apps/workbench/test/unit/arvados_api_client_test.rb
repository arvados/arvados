require 'test_helper'

class ArvadosApiClientTest < ActiveSupport::TestCase
  # We use a mock instead of making real API calls, so there's no need to reset.
  reset_api_fixtures :after_each_test, false

  test 'successful stubbed api request' do
    stub_api_calls_with_body '{"foo":"bar","baz":0}'
    use_token :active
    resp = ArvadosApiClient.new_or_current.api Link, ''
    assert_equal Hash, resp.class
    assert_equal 'bar', resp[:foo]
    assert_equal 0, resp[:baz]
  end

  test 'exception if server returns non-JSON' do
    stub_api_calls_with_invalid_json
    assert_raises ArvadosApiClient::InvalidApiResponseException do
      use_token :active
      resp = ArvadosApiClient.new_or_current.api Link, ''
    end
  end
end
